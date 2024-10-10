package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var discord *discordgo.Session

type Config struct {
	Config struct {
		Port     int    `toml:"port"`
		GuildID  string `toml:"guildID"`
		ShopLink string `toml:"shopLink"`
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
		}
	})

	defer cleanupGuildCommands()
	defer discord.Close()

	router := gin.Default()

	router.POST("/addrole", addRoleHandler)
	router.POST("/removerole", removeRoleHandler)
	router.POST("/sendmessage", sendMessageHandler)
	router.POST("/dmuser", dmUserHandler)

	serverAddress := fmt.Sprintf(":%d", config.Config.Port)
	log.Printf("Webhook server is running on port %d", config.Config.Port)
	log.Fatal(router.Run(serverAddress))
}

func addRoleHandler(c *gin.Context) {
	userID := c.PostForm("userID")
	roleID := c.PostForm("roleID")

	err := discord.GuildMemberRoleAdd(config.Config.GuildID, userID, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error adding role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role added successfully"})
}

func removeRoleHandler(c *gin.Context) {
	userID := c.PostForm("userID")
	roleID := c.PostForm("roleID")

	err := discord.GuildMemberRoleRemove(config.Config.GuildID, userID, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error removing role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role removed successfully"})
}

func sendMessageHandler(c *gin.Context) {
	channelID := c.PostForm("channelID")
	message := c.PostForm("message")

	_, err := discord.ChannelMessageSend(channelID, message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error sending message: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

func dmUserHandler(c *gin.Context) {
	userID := c.PostForm("userID")
	message := c.PostForm("message")

	channel, err := discord.UserChannelCreate(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating DM channel: %v", err)})
		fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
		return
	}

	_, err = discord.ChannelMessageSend(channel.ID, message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error sending message: %v", err)})
		fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

func registerCmds() {
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
	}

	for _, cmd := range commands {
		_, err := discord.ApplicationCommandCreate(discord.State.User.ID, config.Config.GuildID, cmd)
		if err != nil {
			log.Fatalf("Cannot create '%s' command: %v", cmd.Name, err)
		}
		fmt.Printf("'%s' command created\n", cmd.Name)
	}
}

func cleanupGuildCommands() {
	commands, _ := discord.ApplicationCommands(discord.State.User.ID, config.Config.GuildID)
	for _, cmd := range commands {
		_ = discord.ApplicationCommandDelete(discord.State.User.ID, config.Config.GuildID, cmd.ID)
	}
	fmt.Println("Guild commands cleaned up")
}
