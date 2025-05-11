package anime

import (
	"github.com/albert-wang/rawr-discordbot/ai"
	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
)

type SuggestArguments struct {
	All bool `arg:"-a,--all" help:"Even show shows that are over 3 months old"`
}

func (db *Database) Suggest(msg *discordgo.MessageCreate, args *SuggestArguments) (string, error) {
	tplText := `
		Given the following list of items, pick four titles to watch. Take into account how recently they have been watched, with ones
		that have not been watched recently having slightly higher priority.

		{{ .Message }}

		{{ .Table }}
	`

	table, err := db.List(msg, &ListArguments{
		All: args.All,
	})

	if err != nil {
		return "", err
	}

	resolved := chat.ResolveMentionsToNicks(msg.Content, msg.GuildID, msg.Mentions)

	messages := ai.GetContextInChannel(msg.GuildID, msg.ChannelID, 4)
	messages = append(messages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser,
		MultiContent: ai.TemplateContent(tplText, map[string]interface{}{
			"Message": resolved,
			"Table":   table,
		}),
	})

	ai.UnboundedRespondToContent(msg.GuildID, msg.ChannelID, messages)
	return "", nil
}
