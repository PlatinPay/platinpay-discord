package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	discord      *discordgo.Session
	tokenManager *TokenManager
)

type Config struct {
	Config struct {
		Port           int      `toml:"port"`
		GuildID        string   `toml:"guildID"`
		ShopLink       string   `toml:"shopLink"`
		LocalOnly      bool     `toml:"localOnly"`
		WhitelistOnly  bool     `toml:"whitelistOnly"`
		WhitelistedIPs []string `toml:"whitelistedIPs"`
		UseSigning     bool     `toml:"useSigning"`
	} `toml:"config"`
}

var config Config

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	if _, err := os.Stat("config.toml"); os.IsNotExist(err) {
		fmt.Println("config.toml does not exist!")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatalf("DISCORD_TOKEN not set in .env file")
	}

	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		log.Fatalf("Error loading config.toml file: %v", err)
	}

	if config.Config.UseSigning {
		var err error
		tokenManager, err = NewTokenManager("public_key.pem")
		if err != nil {
			log.Printf("Public key not found. Use /settoken to set the public key.")
		}
	}

	discord, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	err = discord.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}

	registerCmds()
	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.ApplicationCommandData().Name {
		case "shop", "store", "platinpay":
			embed := &discordgo.MessageEmbed{
				Description: fmt.Sprintf("Here's the shop link: %s", config.Config.ShopLink),
				Color:       0xFFFFFF,
			}
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
			if err != nil {
				log.Printf("Error responding to /shop command: %v", err)
			}
		case "settoken":
			member := i.Member
			if member == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Error: Member data not found.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			permissions, err := s.UserChannelPermissions(member.User.ID, i.ChannelID)
			if err != nil {
				log.Printf("Error getting user permissions: %v", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Error retrieving your permissions.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			if permissions&discordgo.PermissionAdministrator == 0 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You do not have permission to use this command.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			var token string
			for _, opt := range options {
				if opt.Name == "token" {
					token = opt.StringValue()
					break
				}
			}
			if token == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Token is required.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			err = ioutil.WriteFile("public_key.pem", []byte(token), 0644)
			if err != nil {
				log.Printf("Error writing public key: %v", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Error saving token: %v", err),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			if tokenManager != nil {
				err = tokenManager.ReloadPublicKey()
				if err != nil {
					log.Printf("Error reloading public key: %v", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Error reloading token: %v", err),
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}
			} else {
				tokenManager, err = NewTokenManager("public_key.pem")
				if err != nil {
					log.Printf("Error initializing TokenManager: %v", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("Error initializing TokenManager: %v", err),
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Token set successfully.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	})

	defer cleanupGuildCommands()
	defer discord.Close()

	router := gin.Default()

	router.Use(IPAuthMiddleware())
	router.Use(SignatureVerificationMiddleware())

	router.POST("/addrole", addRoleHandler)
	router.POST("/removerole", removeRoleHandler)
	router.POST("/sendmessage", sendMessageHandler)
	router.POST("/dmuser", dmUserHandler)

	serverAddress := fmt.Sprintf(":%d", config.Config.Port)
	log.Printf("Webhook server is running on port %d", config.Config.Port)
	log.Fatal(router.Run(serverAddress))
}

func IPAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		remoteIP := c.ClientIP()
		allowed := false

		if config.Config.LocalOnly {
			if remoteIP == "127.0.0.1" || remoteIP == "::1" {
				allowed = true
			}
		} else if config.Config.WhitelistOnly {
			for _, ip := range config.Config.WhitelistedIPs {
				if remoteIP == ip {
					allowed = true
					break
				}
			}
		} else {
			allowed = true
		}

		if !allowed {
			log.Printf("Unauthorized request from IP: %s", remoteIP)
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func SignatureVerificationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("SignatureVerificationMiddleware called")

		var signedReq struct {
			Signature string `json:"signature"`
			Data      string `json:"data"`
		}

		if err := c.ShouldBindJSON(&signedReq); err != nil {
			log.Printf("Error parsing JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			c.Abort()
			return
		}

		if config.Config.UseSigning {
			isValid, err := verifySignature([]byte(signedReq.Data), signedReq.Signature)
			if err != nil || !isValid {
				log.Printf("Invalid signature: %v", err)
				c.JSON(http.StatusForbidden, gin.H{"error": "Invalid signature"})
				c.Abort()
				return
			}
			log.Println("Signature verified and data set in context")
		} else {
			log.Println("UseSigning is disabled, skipping signature verification")
		}

		c.Set("data", signedReq.Data)
		c.Next()
	}
}

func addRoleHandler(c *gin.Context) {
	dataRaw, exists := c.Get("data")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data not found"})
		return
	}

	dataString, ok := dataRaw.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid data type"})
		return
	}

	var reqData struct {
		Timestamp float64 `json:"timestamp"`
		UserID    string  `json:"userID"`
		RoleID    string  `json:"roleID"`
	}

	if err := json.Unmarshal([]byte(dataString), &reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	err := discord.GuildMemberRoleAdd(config.Config.GuildID, reqData.UserID, reqData.RoleID)
	if err != nil {
		log.Printf("Error adding role: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error adding role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role added successfully"})
}

func removeRoleHandler(c *gin.Context) {
	dataRaw, exists := c.Get("data")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data not found"})
		return
	}

	dataString, ok := dataRaw.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid data type"})
		return
	}

	var reqData struct {
		Timestamp float64 `json:"timestamp"`
		UserID    string  `json:"userID"`
		RoleID    string  `json:"roleID"`
	}

	if err := json.Unmarshal([]byte(dataString), &reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	err := discord.GuildMemberRoleRemove(config.Config.GuildID, reqData.UserID, reqData.RoleID)
	if err != nil {
		log.Printf("Error removing role: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error removing role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role removed successfully"})
}

func sendMessageHandler(c *gin.Context) {
	dataRaw, exists := c.Get("data")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data not found"})
		return
	}

	dataString, ok := dataRaw.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid data type"})
		return
	}

	var reqData struct {
		Timestamp float64 `json:"timestamp"`
		ChannelID string  `json:"channelID"`
		Message   string  `json:"message"`
	}

	if err := json.Unmarshal([]byte(dataString), &reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	_, err := discord.ChannelMessageSend(reqData.ChannelID, reqData.Message)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error sending message: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

func dmUserHandler(c *gin.Context) {
	dataRaw, exists := c.Get("data")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data not found"})
		return
	}

	dataString, ok := dataRaw.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid data type"})
		return
	}

	var reqData struct {
		Timestamp float64 `json:"timestamp"`
		UserID    string  `json:"userID"`
		Message   string  `json:"message"`
	}

	if err := json.Unmarshal([]byte(dataString), &reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data format"})
		return
	}

	channel, err := discord.UserChannelCreate(reqData.UserID)
	if err != nil {
		log.Printf("Error creating DM channel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating DM channel: %v", err)})
		return
	}

	_, err = discord.ChannelMessageSend(channel.ID, reqData.Message)
	if err != nil {
		log.Printf("Error sending DM: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error sending message: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

func registerCmds() {
	adminPermission := int64(discordgo.PermissionAdministrator)
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "shop",
			Description: "Sends a shop link.",
		},
		{
			Name:        "store",
			Description: "Sends a shop link.",
		},
		{
			Name:        "platinpay",
			Description: "Sends a shop link.",
		},
		{
			Name:        "settoken",
			Description: "Set the public key/token for signature verification",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "token",
					Description: "The public key/token",
					Required:    true,
				},
			},
			DefaultMemberPermissions: &adminPermission,
		},
	}

	for _, cmd := range commands {
		_, err := discord.ApplicationCommandCreate(discord.State.User.ID, config.Config.GuildID, cmd)
		if err != nil {
			log.Fatalf("Cannot create '%s' command: %v", cmd.Name, err)
		}
		fmt.Printf("'%s' command created\n", cmd.Name)
	}
}

type TokenManager struct {
	PublicKey     ed25519.PublicKey
	PublicKeyPath string
}

func NewTokenManager(publicKeyPath string) (*TokenManager, error) {
	tm := &TokenManager{PublicKeyPath: publicKeyPath}
	err := tm.ReloadPublicKey()
	if err != nil {
		return nil, err
	}
	return tm, nil
}

func (tm *TokenManager) ReloadPublicKey() error {
	data, err := ioutil.ReadFile(tm.PublicKeyPath)
	if err != nil {
		return err
	}

	data = bytes.TrimSpace(data)

	derBytes, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("error decoding base64 public key: %v", err)
	}

	parsedKey, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return fmt.Errorf("error parsing DER public key: %v", err)
	}

	pubKey, ok := parsedKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not of type ed25519")
	}

	tm.PublicKey = pubKey
	return nil
}

func verifySignature(data []byte, signature string) (bool, error) {
	if tokenManager == nil || tokenManager.PublicKey == nil {
		log.Println("Public key not loaded")
		return false, fmt.Errorf("public key not loaded")
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		log.Printf("Error decoding signature: %v", err)
		return false, fmt.Errorf("error decoding signature: %v", err)
	}

	if len(signatureBytes) != ed25519.SignatureSize {
		log.Println("Invalid signature size")
		return false, fmt.Errorf("invalid signature size")
	}

	isValid := ed25519.Verify(tokenManager.PublicKey, data, signatureBytes)
	if !isValid {
		log.Println("Signature verification failed")
		return false, nil
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		log.Printf("Error unmarshalling data for timestamp check: %v", err)
		return false, fmt.Errorf("error unmarshalling data: %v", err)
	}

	timestampVal, ok := dataMap["timestamp"]
	if !ok {
		log.Println("Missing timestamp")
		return false, fmt.Errorf("missing timestamp")
	}

	timestamp, ok := timestampVal.(float64)
	if !ok {
		log.Println("Invalid timestamp format")
		return false, fmt.Errorf("invalid timestamp format")
	}

	currentTime := float64(time.Now().UnixMilli())
	if math.Abs(currentTime-timestamp) > 5000 {
		log.Println("Timestamp is too old or in the future")
		return false, fmt.Errorf("timestamp is too old or in the future")
	}

	log.Println("Signature verification and timestamp validation succeeded")
	return true, nil
}

func cleanupGuildCommands() {
	commands, _ := discord.ApplicationCommands(discord.State.User.ID, config.Config.GuildID)
	for _, cmd := range commands {
		_ = discord.ApplicationCommandDelete(discord.State.User.ID, config.Config.GuildID, cmd.ID)
	}
	fmt.Println("Guild commands cleaned up")
}
