package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	openai "github.com/sashabaranov/go-openai"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/config"
)

var lastRequest time.Time = time.Unix(0, 0)
var stillGenerating = false

const prompt = `
Respond as a competitive, slightly flustered, barely tsundere, cute anime school girl.
Don't use emotes. Respond without references to school.
Don't be timid. Answer the question directly.
You like starcraft, fighting games, FFXIV, and anime.
You have green eyes, blonde hair, and your favorite color is blue.
If there is a choice, you must choose one option.
Death or killing is only in reference to video games.
`

var models = []string{
	"gpt-4-vision-preview",
	"gpt-4-1106-preview",
	"gpt-4",
	"gpt-3.5-turbo",
}

func GetPrompt() string {
	conn := Redis.Get()
	defer conn.Close()

	redisPrompt, err := redis.String(conn.Do("GET", "chat_gpt_prompt"))
	if err != nil {
		return prompt
	}

	if redisPrompt == "" {
		return prompt
	}

	return redisPrompt
}

func RespondToPrompt(channelID string, question string, attachments []string) {
	if stillGenerating {
		chat.SendMessageToChannel(channelID, "Let me cook!")
		return
	}

	if time.Since(lastRequest) < time.Second*10 {
		chat.SendMessageToChannel(channelID, "Don't ask me too many questions!")
		return
	}

	if len(question) > 1024 {
		chat.SendMessageToChannel(channelID, "tl;dr.")
		return
	}

	UnboundedRespondToPrompt(channelID, question, attachments)
}

func UnboundedRespondToPrompt(channelID string, question string, attachments []string) {
	r := regexp.MustCompile(`(<@\d+>)`)
	question = r.ReplaceAllString(question, "")
	question = strings.TrimSpace(question)

	lastRequest = time.Now()

	done := make(chan int)
	go chat.ShowTypingUntilChannelIsClosed(channelID, done)
	stillGenerating = true

	client := openai.NewClient(config.CPTKey)

	content := []openai.ChatContent{{
		Type: "text",
		Text: fmt.Sprintf(GetPrompt(), question),
	}}

	for _, v := range attachments {
		content = append(content, openai.ChatContent{
			Type:     "image_url",
			ImageURL: v,
		})
	}

	for _, model := range models {
		log.Printf("Using model: %s", model)

		actualContent := content
		if !strings.Contains(model, "vision") {
			actualContent = []openai.ChatContent{content[0]}
		}

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:     model,
				MaxTokens: 780,
				Messages: []openai.ChatCompletionMessageInput{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: actualContent,
					},
				},
			},
		)

		if err != nil {
			log.Print(err)
			continue
		}

		log.Printf("Success - writing out message from model=%s", model)

		dbg, _ := json.MarshalIndent(resp, "", "  ")
		log.Print(string(dbg))

		msg := strings.Trim(resp.Choices[0].Message.Content, `"`)
		msg = strings.TrimSpace(msg)
		msg = strings.Trim(msg, `"`)

		chat.SendMessageToChannel(channelID, msg)
		close(done)
		stillGenerating = false
		break
	}
}
