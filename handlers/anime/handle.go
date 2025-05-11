package anime

import (
	"bytes"
	"log"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/alexflint/go-arg"
	"github.com/bwmarrin/discordgo"
)

type Handler func(database *Database, cmd string, msg *discordgo.MessageCreate, args []string)

func Handle(command string, m *discordgo.MessageCreate, args []string) {
	complete := chat.ShowTyping(m.ChannelID)
	defer complete()

	db := LoadDatabase()

	err := db.Handle(command, m, args)
	if err != nil {
		log.Println(err)
		return
	}

	db.Save()
}

func parseArguments[T any](m *discordgo.MessageCreate, args []string) (*T, error) {
	buffer := bytes.Buffer{}
	cfg := arg.Config{
		Program:           "nvg-tan",
		IgnoreEnv:         true,
		StrictSubcommands: true,
		Out:               &buffer,
	}

	result := new(T)
	parser, err := arg.NewParser(cfg, result)
	if err != nil {
		log.Printf("args=%+v\n", args)
		log.Printf("result=%+v\n", result)

		log.Print(err)
		chat.SendMessageToChannel(m.ChannelID, buffer.String())
		return result, err
	}

	err = parser.Parse(args)
	if err != nil {
		log.Print(err)
		return result, err
	}

	return result, nil
}

func genericHandle[T any](db *Database, m *discordgo.MessageCreate, args []string, cb func(db *Database, m *discordgo.MessageCreate, args *T) (string, error)) error {
	typedArgs, err := parseArguments[T](m, args)
	if err != nil {
		log.Print(err)
		return err
	}

	resp, err := cb(db, m, typedArgs)
	if err != nil {
		log.Print(err)
		return err
	}

	if resp != "" {
		chat.SendMessageToChannel(m.ChannelID, resp)
	}

	return nil
}
