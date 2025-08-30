package ai

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"slices"
	"text/template"
	"time"

	"github.com/albert-wang/rawr-discordbot/chat"
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

func templateFragment(path string) (string, error) {
	prefix, err := os.ReadFile(path)
	if err != nil {
		log.Print(err)
		return "", err
	}

	now := time.Now()
	where, err := time.LoadLocation("America/Chicago")
	today := fmt.Sprintf("Respond as if it is %s", now.Format("Monday, Jan 02 2006 15:04:05"))
	if err == nil {
		today = fmt.Sprintf("Respond as if it is %s", now.In(where).Format("Monday, Jan 02 2006 15:04:05"))
	}

	tpl := template.New("prefix")
	tpl.Parse(string(prefix))

	buffer := bytes.Buffer{}
	err = tpl.Execute(&buffer, map[string]any{
		"Now": today,
	})

	if err != nil {
		log.Print(err)

		return "", err
	}

	return buffer.String(), nil
}

func GetPrompt() string {
	prompt, err := templateFragment("./config/prompt.tpl")
	if err != nil {
		log.Printf("err: %+v", err)
		return fmt.Sprintf(`%s\n\n%s`, getCurrentTimePromptFragment(), defaultPrompt)
	}

	return prompt
}

func GetContextInChannel(guild string, channel string, contextSize int) []openai.ChatCompletionMessage {
	result := []openai.ChatCompletionMessage{}
	result = append(result, openai.ChatCompletionMessage{
		Role:         "developer",
		MultiContent: TextContent(GetPrompt()),
	})

	messages := chat.GetPreviousMessageFromUser(guild, channel, "")
	count := len(messages)
	if contextSize < count {
		count = contextSize
	}

	imagesSent := 0
	for i := 0; i < count; i++ {
		hasMedia := imagesSent < 2
		if messages[i].Author.Bot {
			hasMedia = false
		}

		if messages[i].Author.Bot {
			contents := MessageContent(messages[i], ConversionOptions{
				Format:       fmt.Sprintf("%s > %%s", messages[i].Author.Username),
				IncludeMedia: false,
			})

			result = append(result, openai.ChatCompletionMessage{
				Role:         openai.ChatMessageRoleAssistant,
				MultiContent: contents,
			})
		} else {
			contents := MessageContent(messages[i], ConversionOptions{
				Format:       fmt.Sprintf("%s > %%s", messages[i].Author.Username),
				IncludeMedia: hasMedia && true,
			})

			for _, v := range contents {
				if v.Type != openai.ChatMessagePartTypeText {
					imagesSent++
				}
			}

			result = append(result, openai.ChatCompletionMessage{
				Role:         openai.ChatMessageRoleUser,
				MultiContent: contents,
			})
		}
	}

	slices.Reverse(result)
	return result
}
