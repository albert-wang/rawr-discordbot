package main

import (
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
	"github.com/albert-wang/rawr-discordbot/storage"
)

type ChatOnMessage struct {
	argumentSplitter *regexp.Regexp
	commands         map[string]handlers.CommandHandler
	stickers         map[string]handlers.CommandHandler
	ai               *handlers.AIResponder
}

func (c *ChatOnMessage) Help(m *discordgo.MessageCreate, args []string) error {
	msg := "This is NVG-Tan. A listing of commands follows."
	res := []string{}
	for k, _ := range c.commands {
		res = append(res, k)
	}

	msg = msg + " " + strings.Join(res, ", ")

	chat.SendMessageToChannel(m.ChannelID, msg)
	return nil
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

	if len(m.Attachments) != 0 || len(m.Embeds) != 0 {
		log.Print("Assigning attachment...")
		handlers.RegisterAttachmentFromMessage(m)
	}

	if len(m.StickerItems) != 0 {
		cmd := m.StickerItems[0].Name
		handler, ok := c.stickers[cmd]
		if ok {
			go handler(m, []string{})
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

	unprefixedCommand := cmd[1:]
	if unprefixedCommand == "help" {
		go c.Help(m, args)
		return
	}

	handler, ok := c.commands[unprefixedCommand]
	if ok {
		go func() {
			err := handler(m, args)
			if err != nil {
				log.Print(err)
			}
		}()
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
		commands: map[string]handlers.CommandHandler{
			"smug":      handlers.RandomS3FileFrom("img.rawr.moe", "smug/"),
			"kajiura":   handlers.RandomS3FileFrom("img.rawr.moe", "music/"),
			"countdown": handlers.Countdown,
			"anime":     handlers.Anime,
			"rotate":    handlers.RotateLastImages,
		},
		stickers: map[string]handlers.CommandHandler{
			"landscape": func(m *discordgo.MessageCreate, args []string) error {
				return handlers.RotateLastImages(m, []string{"-90"})
			},
		},
		ai: &handlers.AIResponder{},
	}

	chat.ConnectToWebsocket(config.BotToken, func(s *discordgo.Session, m *discordgo.MessageCreate) {
		h.OnMessage(s, m)
	})

	log.Print("Listening and running...")

	go chat.ShowTypingForever()

	runtime.Goexit()
}
