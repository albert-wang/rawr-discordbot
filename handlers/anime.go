package handlers

import (
	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/handlers/anime"
	"github.com/bwmarrin/discordgo"
)

func Anime(m *discordgo.MessageCreate, args []string) error {
	if len(args) < 1 {
		chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime <del|mv|incr|decr|set|list> <name> [<value>]")
	}

	anime.Handle(args[0], m, args[1:])
	return nil
}
