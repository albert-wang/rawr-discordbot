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
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
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

func GetContextInChannel(guild string, channel string, contextSize int) []responses.ResponseInputItemUnionParam {
	result := []responses.ResponseInputItemUnionParam{}

	messages := chat.GetPreviousMessageFromUser(guild, channel, "")
	count := len(messages)
	if contextSize < count {
		count = contextSize
	}

	for i := 0; i < count; i++ {
		if messages[i].Author.Bot {
			content := ""
			for _, p := range MessageContent(messages[i], ConversionOptions{IncludeMedia: false}) {
				if p.OfInputText == nil {
					continue
				}
				if len(content) > 0 {
					content += "\n"
				}
				content += p.OfInputText.Text
			}

			result = append(result, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleAssistant,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: param.NewOpt(content),
					},
				},
			})
		} else {
			result = append(result, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Content: responses.EasyInputMessageContentUnionParam{
						OfInputItemContentList: responses.ResponseInputMessageContentListParam(MessageContent(messages[i], ConversionOptions{IncludeMedia: true})),
					},
				},
			})
		}

	}

	slices.Reverse(result)
	return result
}
