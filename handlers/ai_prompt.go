package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/gomodule/redigo/redis"
	"github.com/sashabaranov/go-openai"
)

const defaultPrompt = `
Respond as a competitive, slightly flustered, barely tsundere, cute anime school girl.
Don't use emotes. Respond without references to school.
Don't be timid. Answer the question directly.
You like starcraft, fighting games, FFXIV, and anime.
You have green eyes, blonde hair, and your favorite color is blue.
If there is a choice, you must choose one option.
Death or killing is only in reference to video games.
`

func getCurrentTimePromptFragment() string {
	now := time.Now()
	where, err := time.LoadLocation("America/Chicago")

	today := fmt.Sprintf("Respond as if it is %s", now.Format("Monday, Jan 02 2006 15:04:05"))
	if err == nil {
		today = fmt.Sprintf("Respond as if it is %s", now.In(where).Format("Monday, Jan 02 2006 15:04:05"))
	}

	return today
}

func getRedisPromptFragment() string {
	conn := Redis.Get()
	defer conn.Close()

	redisPrompt, err := redis.String(conn.Do("GET", "chat_gpt_prompt"))
	if err != nil {
		return defaultPrompt
	}

	if redisPrompt == "" {
		return defaultPrompt
	}

	return redisPrompt
}

func GetPrompt() string {
	parts := []string{
		getCurrentTimePromptFragment(),
		getRedisPromptFragment(),
	}

	return strings.Join(parts, "\n\n")
}

func GenerateMessagesWithContext(guild string, channel string, contextSize int) []openai.ChatCompletionMessage {
	result := []openai.ChatCompletionMessage{}
	result = append(result, openai.ChatCompletionMessage{
		Role:         "developer",
		MultiContent: textContent(GetPrompt()),
	})

	messages := chat.GetPreviousMessageFromUser(guild, channel, "")
	count := len(messages)
	if contextSize < count {
		count = contextSize
	}

	for i := count - 1; i >= 0; i-- {
		contents, _ := convertMessageToContent(messages[i], fmt.Sprintf("%s > %%s", messages[i].Author.Username))
		if messages[i].Author.Bot {
			withoutMedia := []openai.ChatMessagePart{}
			for _, c := range contents {
				if c.Type == openai.ChatMessagePartTypeText {
					withoutMedia = append(withoutMedia, c)
				}
			}

			result = append(result, openai.ChatCompletionMessage{
				Role:         openai.ChatMessageRoleAssistant,
				MultiContent: withoutMedia,
			})
		} else {
			result = append(result, openai.ChatCompletionMessage{
				Role:         openai.ChatMessageRoleUser,
				MultiContent: contents,
			})
		}
	}

	return result
}
