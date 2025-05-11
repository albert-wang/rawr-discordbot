package handlers

import (
	"github.com/bwmarrin/discordgo"
)

type CommandHandler func(*discordgo.MessageCreate, []string) error

type StatefulCommandHandler interface {
	Invoke(*discordgo.MessageCreate, []string) error
}

func StatefulHandler(st StatefulCommandHandler) CommandHandler {
	return func(m *discordgo.MessageCreate, args []string) error {
		return st.Invoke(m, args)
	}
}
