package chat

import (
	"fmt"
	"log"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/albert-wang/tracederror"
	"github.com/bwmarrin/discordgo"
)

var client *discordgo.Session
var self *discordgo.User

func IsBotUser(user *discordgo.User) bool {
	if self != nil && user != nil {
		return user.ID == self.ID
	}

	return false
}

// ConnectToWebsocket connects to the discord websocket with the given token.
// This makes the bot appear online, and will begin receiving messages.
func ConnectToWebsocket(token string, onMessage func(*discordgo.Session, *discordgo.MessageCreate)) error {
	var err error
	token = fmt.Sprintf("Bot %s", token)
	client, err = discordgo.New(token)
	if err != nil {
		log.Print("Failed to create discord client")
		return tracederror.New(err)
	}

	client.AddHandler(onMessage)
	err = client.Open()
	if err != nil {
		log.Print("Failed to open connection to discord websocket API")
		log.Print(err)
		return tracederror.New(err)
	}

	user, err := client.User("@me")
	if err != nil {
		log.Print("Failed to get self")
		log.Print(err)
		return tracederror.New(err)
	}

	self = user
	return nil
}

func GetChannelInformation(channelID string) (*discordgo.Channel, error) {
	return client.Channel(channelID)
}

// SendMessageToChannel sends a message to a channelID.
func SendMessageToChannel(channelID string, message string) {
	_, err := client.ChannelMessageSend(channelID, message)
	if err != nil {
		log.Print("==============ERROR==============")
		log.Print(err)
		log.Print("==============Message============== [", len(message), "]")
		log.Print(message)
		log.Print("==============")
	}
}

func SendImagesToChannel(channelID string, files []*discordgo.File) {
	_, err := client.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Files: files,
	})

	if err != nil {
		log.Print(err)
	}
}

func SendPrivateMessageTo(user string, message string) {
	ch, err := client.UserChannelCreate(user)
	if err != nil {
		log.Print(err)
	}

	SendMessageToChannel(ch.ID, message)
}

func GetNick(guildID string, user string) string {
	member, err := client.GuildMember(guildID, user)
	if err != nil {
		log.Print(err)
		return fmt.Sprintf("<User:%s>", user)
	}

	if member.Nick == "" {
		return member.User.Username
	}

	return member.Nick
}

func GetPreviousMessageFromUser(guildID string, channelID string, user string) []*discordgo.Message {
	messages, err := client.ChannelMessages(channelID, 30, "", "", "")
	if err != nil {
		log.Print(err)
		return []*discordgo.Message{}
	}

	if user == "" {
		return messages
	}

	user = strings.ToLower(user)
	idToNick := map[string]string{}
	results := []*discordgo.Message{}
	for _, v := range messages {
		nick, ok := idToNick[v.Author.ID]
		if !ok {
			member, err := client.GuildMember(guildID, v.Author.ID)
			if err != nil {
				log.Print(err)
				continue
			}

			nick = member.Nick
			idToNick[v.Author.ID] = nick
		}

		normalizedUsername := strings.ToLower(v.Author.Username)
		normalizedNick := strings.ToLower(nick)
		if strings.Contains(normalizedUsername, user) || strings.Contains(normalizedNick, user) || user == "user" {
			// Just going to steal the email field here, lol
			v.Author.Email = nick
			results = append(results, v)
		}
	}

	return results
}

var typingStatuses map[string]int = map[string]int{}
var typingStatusMutex sync.Mutex = sync.Mutex{}

func ShowTypingForever() {
	ticker := time.NewTicker(time.Second / 2 * 5)

	for _ = range ticker.C {

		typingStatusMutex.Lock()
		copy := maps.Clone(typingStatuses)
		typingStatusMutex.Unlock()

		for channel, typing := range copy {
			if typing > 0 {
				client.ChannelTyping(channel)
			}
		}
	}
}

// ShowTyping will display the bot as typing something in
// the given channel until the returned function is called.
func ShowTyping(channelID string) func() {
	typingStatusMutex.Lock()
	typingStatuses[channelID] = typingStatuses[channelID] + 1
	typingStatusMutex.Unlock()

	client.ChannelTyping(channelID)

	return func() {
		typingStatusMutex.Lock()

		result := typingStatuses[channelID]
		result -= 1
		typingStatuses[channelID] = result

		if result <= 0 {
			delete(typingStatuses, channelID)
		}

		typingStatusMutex.Unlock()
	}
}
