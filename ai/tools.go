package ai

import (
	"log"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/sashabaranov/go-openai"
)

func SupportedFunctions() []openai.Tool {
	type Object map[string]interface{}

	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
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
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_last_image",
				Description: "Gets all images from a recent message",
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
		},
	}
}

type GetPreviousNMessagesFromUserArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

func GetPreviousNMessagesFromUser(guild string, channel string, args GetPreviousNMessagesFromUserArgs) []openai.ChatMessagePart {
	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		return []openai.ChatMessagePart{}
	}

	max := 5
	if args.Count < 5 && args.Count > 0 {
		max = args.Count
	}

	if max > len(messages)-1 {
		max = len(messages) - 1
	}

	messages = messages[:max]
	content := []openai.ChatMessagePart{}

	for _, v := range messages {
		next := MessageContent(v, ConversionOptions{
			Format:       "%s",
			IncludeMedia: true,
		})
		content = append(content, next...)
	}

	return content
}

type GetLastImageArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

func GetLastImage(guild string, channel string, args GetLastImageArgs) []openai.ChatMessagePart {
	log.Print("Getting last image with: ")
	log.Printf("%+v", args)

	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		log.Print("Found no relevant messages")
		return []openai.ChatMessagePart{}
	}

	if args.Count == 0 {
		args.Count = 1
	}

	max := 5
	if args.Count < 5 && args.Count > 0 {
		max = args.Count
	}

	if max > len(messages)-1 {
		max = len(messages) - 1
	}

	content := []openai.ChatMessagePart{}
	processedMessageCount := 0
	for _, v := range messages {
		if len(v.Attachments) == 0 && len(v.Embeds) == 0 {
			continue
		}

		log.Print(v.Content)

		if processedMessageCount < max {
			processedMessageCount++

			embeds := EmbedsContent(v)
			attachments := AttachmentsContent(v)

			content = append(content, embeds...)
			content = append(content, attachments...)
		}
	}

	return content
}
