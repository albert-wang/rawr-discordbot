package ai

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/config"
)

func makeOpenAPIRequest(guild string, channel string, model AIModel, recursiveDepth int, client *openai.Client, messages *[]openai.ChatCompletionMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:               model.Name,
		MaxCompletionTokens: 1024 * 4,
		Messages:            *messages,
	}

	if recursiveDepth > 0 {
		req.Tools = ToolDefinitions()
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		req,
	)

	if err != nil {
		log.Print(err)

		dbgreq, _ := json.MarshalIndent(req, "", "  ")
		log.Print(string(dbgreq))

		dbg, _ := json.MarshalIndent(resp, "", "  ")
		log.Print(string(dbg))

		return "", err
	}

	if len(resp.Choices) == 0 {
		return "Empty response :(", nil
	}

	choice := resp.Choices[0]

	if len(choice.Message.ToolCalls) > 0 {
		for _, call := range choice.Message.ToolCalls {
			fn, _ := json.Marshal(call.Function)
			*messages = append(*messages, openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant,
				MultiContent: []openai.ChatMessagePart{{
					Type: "text",
					Text: string(fn),
				}},
			})

			additionalContext := InvokeTool(guild, channel, call.Function.Name, call.Function.Arguments)
			if len(additionalContext) > 0 {
				*messages = append(*messages, openai.ChatCompletionMessage{
					Role:         openai.ChatMessageRoleUser,
					Name:         call.Function.Name,
					MultiContent: additionalContext,
				})
			}
		}

		// Recursive call with function call results.
		return makeOpenAPIRequest(guild, channel, PrimaryModel, recursiveDepth-1, client, messages)
	}

	msg := strings.Trim(choice.Message.Content, `"`)
	msg = strings.TrimSpace(msg)
	msg = strings.Trim(msg, `"`)

	msg = strings.TrimPrefix(msg, "NVG-Tan >")

	msg = strings.TrimSpace(msg)
	msg = strings.Trim(msg, `"`)

	return msg, nil
}

func UnboundedRespondToContent(guildID string, channelID string, messages []openai.ChatCompletionMessage) {
	client := openai.NewClient(config.CPTKey)

	msg, err := makeOpenAPIRequest(guildID, channelID, PrimaryModel, 3, client, &messages)
	if err != nil {
		chat.SendMessageToChannel(channelID, "Error while generating message, "+err.Error())
		log.Print(err)
		return
	}

	splitMessages := chat.SplitMessage(msg)
	for _, msg := range splitMessages {
		chat.SendMessageToChannel(channelID, msg)
	}
}
