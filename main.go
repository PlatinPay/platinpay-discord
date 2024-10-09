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
		Port    int    `toml:"port"`
		GuildID string `toml:"guildID"`
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
	defer discord.Close()

	router := gin.Default()

	router.POST("/addrole", addRoleHandler)
	router.POST("/removerole", removeRoleHandler)
	router.POST("/sendmessage", sendMessageHandler)

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
