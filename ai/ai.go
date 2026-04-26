package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/config"
)

func makeOpenAPIRequest(guild string, channel string, model AIModel, recursiveDepth int, client openai.Client, messages []responses.ResponseInputItemUnionParam) (string, error) {
	ctx := context.Background()
	toolCalls := int64(32)

	req := responses.ResponseNewParams{
		Model:           model.Name,
		MaxOutputTokens: param.NewOpt[int64](1024 * 4),
		Reasoning: shared.ReasoningParam{
			Effort: "low",
		},
		Instructions: param.NewOpt(GetPrompt()),
		Tools:        ToolDefinitions(),
		MaxToolCalls: param.NewOpt(toolCalls),

		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: messages,
		},

		Store: param.NewOpt(true),
	}

	start := time.Now()

	for steps := 0; steps < 10; steps++ {
		resp, err := client.Responses.New(ctx, req)
		if err != nil {
			log.Print(err)

			dbgreq, _ := json.MarshalIndent(req, "", "  ")
			log.Print(string(dbgreq))

			dbg, _ := json.MarshalIndent(resp, "", "  ")
			log.Print(string(dbg))

			return "", err
		}

		log.Printf(" -- step %d took %s time", steps, time.Since(start))
		start = time.Now()

		next := []responses.ResponseInputItemUnionParam{}
		for _, output := range resp.Output {
			switch output.Type {
			case "function_call":
				{
					toolCalls -= 1

					functionCall := output.AsFunctionCall()
					next = append(next, responses.ResponseInputItemParamOfFunctionCallOutput(
						functionCall.CallID,
						toFunctionCallOutputs(InvokeTool(guild, channel, functionCall.Name, functionCall.Arguments)),
					))
					break
				}
			case "message":
				{
					msg := output.AsMessage()
					switch msg.Status {
					case "completed":
						totalContent := ""
						for _, content := range msg.Content {
							if content.Type == "output_text" {
								totalContent += content.AsOutputText().Text
							}
						}

						return cleanupMessage(totalContent), nil
					case "failed", "incomplete", "cancelled":
						{
							return "", fmt.Errorf("error: %s", msg.Status)
						}
					}
				}
			default:
				// Ignored
			}
		}

		if toolCalls < 0 {
			toolCalls = 0
		}

		req.PreviousResponseID = param.NewOpt(resp.ID)
		req.MaxToolCalls = param.NewOpt(toolCalls)
		req.Input = responses.ResponseNewParamsInputUnion{
			OfInputItemList: next,
		}
	}

	return "", fmt.Errorf("error: exhausted steps")
}

func cleanupMessage(msg string) string {
	var msgOpenTagRegex = regexp.MustCompile(`<msg\s+[^>]*>`)
	msg = msgOpenTagRegex.ReplaceAllString(msg, "")
	msg = strings.ReplaceAll(msg, "</msg>", "")
	msg = strings.TrimSpace(msg)
	msg = strings.Trim(msg, `"`)
	msg = strings.TrimPrefix(msg, "NVG-Tan >")
	msg = strings.TrimSpace(msg)
	msg = strings.Trim(msg, `"`)
	return msg
}

func toFunctionCallOutputs(parts []responses.ResponseInputContentUnionParam) responses.ResponseFunctionCallOutputItemListParam {
	out := []responses.ResponseFunctionCallOutputItemUnionParam{}
	for _, p := range parts {
		var item responses.ResponseFunctionCallOutputItemUnionParam
		switch {
		case p.OfInputText != nil:
			item.OfInputText = &responses.ResponseInputTextContentParam{
				Text: p.OfInputText.Text,
			}
		case p.OfInputImage != nil:
			in := p.OfInputImage
			item.OfInputImage = &responses.ResponseInputImageContentParam{
				FileID:   in.FileID,
				ImageURL: in.ImageURL,
				Detail:   responses.ResponseInputImageContentDetail(in.Detail),
			}
		default:
			continue
		}
		out = append(out, item)
	}
	return out
}

func UnboundedRespondToContent(guildID string, channelID string, messages []responses.ResponseInputItemUnionParam) []string {
	client := openai.NewClient(
		option.WithAPIKey(config.CPTKey),
	)

	msg, err := makeOpenAPIRequest(guildID, channelID, PrimaryModel, 3, client, messages)
	if err != nil {
		chat.SendMessageToChannel(channelID, "Error while generating message, "+err.Error())
		log.Print(err)
		return []string{}
	}

	return chat.SplitMessage(msg)
}
