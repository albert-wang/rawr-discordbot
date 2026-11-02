package main

import (
	"fmt"
	"log"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gomodule/redigo/redis"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/config"
	"github.com/albert-wang/rawr-discordbot/handlers"
	"github.com/albert-wang/rawr-discordbot/handlers/anime"
	"github.com/albert-wang/rawr-discordbot/storage"
)

type ChatOnMessage struct {
	argumentSplitter *regexp.Regexp
	definitions      []*discordgo.ApplicationCommand
	stickers         map[string]func(*discordgo.MessageCreate)
	ai               *handlers.AIResponder
}

func (c *ChatOnMessage) Help(i *discordgo.InteractionCreate) error {
	msg := "This is NVG-Tan. A listing of commands follows."
	for _, cmd := range c.definitions {
		msg = msg + fmt.Sprintf("\n/%s - %s", cmd.Name, cmd.Description)
	}

	chat.SendFollowup(i.Interaction, msg)
	return nil
}

func (c *ChatOnMessage) OnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Only /anime has autocomplete, for the slug of the show.
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		go func() {
			typed := ""
			for _, opt := range i.ApplicationCommandData().Options[0].Options {
				if opt.Focused {
					typed = opt.StringValue()
				}
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: anime.Autocomplete(typed),
				},
			})

			if err != nil {
				log.Print(err)
			}
		}()
		return
	}

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	// Shows "NVG-Tan is thinking..." since Discord only waits 3 seconds
	// for a response. The handlers reply with followups.
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	if err != nil {
		log.Print(err)
	}

	go func() {
		var err error
		switch i.ApplicationCommandData().Name {
		case "smug":
			err = handlers.RandomS3FileFrom(i, "img.rawr.moe", "smug/")
		case "kajiura":
			err = handlers.RandomS3FileFrom(i, "img.rawr.moe", "music/")
		case "countdown":
			err = handlers.Countdown(i)
		case "anime":
			err = handlers.Anime(i)
		case "rotate":
			err = handlers.Rotate(i)
		case "help":
			err = c.Help(i)
		}

		if err != nil {
			log.Print(err)
			chat.SendFollowup(i.Interaction, err.Error())
		}
	}()
}

func (c *ChatOnMessage) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if chat.IsBotUser(m.Author) {
		return
	}

	if len(m.Mentions) != 0 {
		canAI := false
		for _, v := range m.Mentions {
			if chat.IsBotUser(v) {
				canAI = true
			}
		}

		if canAI {
			go c.ai.Invoke(m, []string{})
			return
		}
	}

	// A thread started from one of her messages is a conversation with her,
	// so she answers everything in it. The thread's ID is the starter's ID.
	channel, err := s.State.Channel(m.ChannelID)
	if err == nil && channel.IsThread() {
		starter, err := s.ChannelMessage(channel.ParentID, channel.ID)
		if err == nil && chat.IsBotUser(starter.Author) {
			go c.ai.Invoke(m, []string{})
			return
		}
	}

	if len(m.Attachments) != 0 || len(m.Embeds) != 0 {
		log.Print("Assigning attachment...")
		handlers.RegisterAttachmentFromMessage(m)
	}

	if len(m.StickerItems) != 0 {
		cmd := m.StickerItems[0].Name
		handler, ok := c.stickers[cmd]
		if ok {
			go handler(m)
			return
		}
	}

	args := c.argumentSplitter.FindAllString(m.Content, -1)
	if len(args) == 0 {
		return
	}

	cmd := args[0]
	args = args[1:]

	for i, arg := range args {
		if strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") {
			args[i] = args[i][1 : len(args[i])-1]
		} else if strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") {
			args[i] = args[i][1 : len(args[i])-1]
		}
	}

	if !strings.HasPrefix(cmd, "!") {
		return
	}

	// The ! commands are slash commands now. !anime list still works since
	// it doesn't change anything, the rest just point at their replacement.
	unprefixedCommand := cmd[1:]
	if unprefixedCommand == "anime" && len(args) > 0 && args[0] == "list" {
		go func() {
			complete := chat.ShowTyping(m.ChannelID)
			defer complete()

			resp, err := anime.Handle("list", args[1:])
			if err != nil {
				log.Print(err)
				return
			}

			complete()
			for _, msg := range chat.SplitMessage(resp + "\nUse `/anime list` next time!") {
				chat.SendMessageToChannel(m.ChannelID, msg)
			}
		}()
		return
	}

	for _, def := range c.definitions {
		if def.Name == unprefixedCommand {
			chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("Use /%s instead!", unprefixedCommand))
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var err error
	config.LoadConfigFromFileAndENV("./config/config.json")

	storage.Redis = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", config.RedisServerAddress)
			if err != nil {
				return nil, err
			}

			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	auth, err := aws.GetAuth(config.AWSAccessKey, config.AWSSecret)
	if err != nil {
		log.Fatal(err)
	}

	storage.S3Client = s3.New(auth, aws.USEast)

	h := ChatOnMessage{
		argumentSplitter: regexp.MustCompile("'.+'|\".+\"|\\S+"),
		definitions: []*discordgo.ApplicationCommand{
			{Name: "smug", Description: "Post a random smug anime face"},
			{Name: "kajiura", Description: "Post a random Kajiura song"},
			{
				Name:        "countdown",
				Description: "Count down in chat",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "from",
						Description: "Number to count down from, defaults to 3, at most 30",
					},
				},
			},
			handlers.AnimeCommand,
			{
				Name:        "rotate",
				Description: "Rotate the last images posted in this channel",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "degrees",
						Description: "Degrees clockwise, defaults to 90",
					},
				},
			},
			{Name: "help", Description: "List NVG-Tan's commands"},
		},
		stickers: map[string]func(*discordgo.MessageCreate){
			"landscape": func(m *discordgo.MessageCreate) {
				chat.SendImagesToChannel(m.ChannelID, handlers.RotateLastImages(m.GuildID, m.ChannelID, "-90"))
			},
		},
		ai: &handlers.AIResponder{},
	}

	chat.ConnectToWebsocket(config.BotToken, &h, h.definitions)

	log.Print("Listening and running...")

	go chat.ShowTypingForever()

	runtime.Goexit()
}
