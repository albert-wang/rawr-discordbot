package handlers

import (
	"log"
	"strings"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
)

// Resolves all non bot-user mentions to the nickname in the server.
func ResolveMentionsToNicks(message string, guild string, mentions []*discordgo.User) string {
	for _, user := range mentions {
		if user.Bot {
			message = strings.NewReplacer(
				"<@"+user.ID+">", "",
				"<@!"+user.ID+">", "",
			).Replace(message)
		} else {
			nick := chat.GetNick(guild, user.ID)
			message = strings.NewReplacer(
				"<@"+user.ID+">", "@"+nick,
				"<@!"+user.ID+">", "@"+nick,
			).Replace(message)
		}
	}

	return message
}

func Test(message *discordgo.MessageCreate, args []string) error {
	str := ResolveMentionsToNicks(message.Content, message.GuildID, message.Mentions)
	log.Print(str)

	return nil
}
