package handlers

import (
	"time"

	"github.com/albert-wang/rawr-discordbot/ai"
	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
	openai "github.com/sashabaranov/go-openai"
)

type AIResponder struct {
	lastRequest     time.Time
	stillGenerating bool
}

func (a *AIResponder) Invoke(m *discordgo.MessageCreate, args []string) error {
	if a.stillGenerating {
		chat.SendMessageToChannel(m.ChannelID, "I..its not like I'm still working for you or anything!")
		return nil
	}

	if time.Since(a.lastRequest) < time.Second*10 {
		chat.SendMessageToChannel(m.ChannelID, "Don't ask me too many questions!")
		return nil
	}

	complete := chat.ShowTyping(m.ChannelID)
	defer complete()

	messages := ai.GetContextInChannel(m.GuildID, m.ChannelID, 32)

	content := ai.MessageContent(m.Message, ai.ConversionOptions{
		IncludeMedia: true,
	})

	messages = append(messages, openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleUser,
		MultiContent: content,
	})

	a.lastRequest = time.Now()
	a.stillGenerating = true

	splitMessages := ai.UnboundedRespondToContent(m.GuildID, m.ChannelID, messages)
	complete()

	for _, msg := range splitMessages {
		chat.SendMessageToChannel(m.ChannelID, msg)
	}

	a.stillGenerating = false
	return nil
}
