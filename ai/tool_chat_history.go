package ai

import (
	"log"

	openai "github.com/sashabaranov/go-openai"

	"github.com/albert-wang/rawr-discordbot/chat"
)

type GetPreviousNMessagesFromUserArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

type GetLastImageArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

func init() {
	type Object = map[string]any

	registerTool(DefineTool[GetPreviousNMessagesFromUserArgs](
		openai.FunctionDefinition{
			Name:        "get_previous_n_messages_from_user",
			Description: "Gets the previous N messages for a given user",
			Strict:      true,
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"count": Object{
						"type":        "integer",
						"description": "How many messages to get. Maximum 5, minimum 1",
					},
					"who": Object{
						"type":        "string",
						"description": "Who to get the message for. May be the empty string to get everyone's entries",
					},
				},
				"required": []string{"count", "who"},
			},
		},
		getPreviousNMessagesFromUser,
	))

	registerTool(DefineTool[GetLastImageArgs](
		openai.FunctionDefinition{
			Name:        "get_last_image",
			Description: "Gets all images from a recent message",
			Strict:      true,
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"count": Object{
						"type":        "integer",
						"description": "Number from 1 to 5, inclusive, indicating which message to get images for. 1 is the most recent image, and 5 being the 5th least recent.",
					},
					"who": Object{
						"type":        "string",
						"description": "The username to get the previous messages for. May not be empty.",
					},
				},
				"required": []string{"count", "who"},
			},
		},
		getLastImage,
	))
}

// clampCount clamps to [1, min(5, available)]. Returns 0 if nothing fits,
// matching the prior "off by one" behavior where max <= len(messages)-1.
func clampCount(requested, available int) int {
	max := 5
	if requested > 0 && requested < 5 {
		max = requested
	}
	if max > available-1 {
		max = available - 1
	}
	return max
}

func getPreviousNMessagesFromUser(guild, channel string, args GetPreviousNMessagesFromUserArgs) []openai.ChatMessagePart {
	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		return nil
	}

	max := clampCount(args.Count, len(messages))
	messages = messages[:max]

	content := []openai.ChatMessagePart{}
	for _, v := range messages {
		content = append(content, MessageContent(v, ConversionOptions{
			Format:       "%s",
			IncludeMedia: true,
		})...)
	}
	return content
}

func getLastImage(guild, channel string, args GetLastImageArgs) []openai.ChatMessagePart {
	log.Printf("Getting last image with: %+v", args)

	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		log.Print("Found no relevant messages")
		return nil
	}

	if args.Count == 0 {
		args.Count = 1
	}
	max := clampCount(args.Count, len(messages))

	content := []openai.ChatMessagePart{}
	processed := 0
	for _, v := range messages {
		if len(v.Attachments) == 0 && len(v.Embeds) == 0 {
			continue
		}
		if processed >= max {
			break
		}
		processed++
		content = append(content, EmbedsContent(v)...)
		content = append(content, AttachmentsContent(v)...)
	}
	return content
}
