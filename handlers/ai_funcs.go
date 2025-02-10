package handlers

import (
	"log"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/sashabaranov/go-openai"
)

func SupportedFunctions() []openai.FunctionDefinition {
	type Object map[string]interface{}

	return []openai.FunctionDefinition{
		{
			Name:        "get_previous_n_messages_from_user",
			Description: "Gets the previous N messages for a given user",
			Parameters: Object{
				"type": "object",
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
		{
			Name:        "get_last_image",
			Description: "Gets the most recent uploaded or embedded image",
			Parameters: Object{
				"type": "object",
				"properties": Object{
					"count": Object{
						"type":        "integer",
						"description": "Number from 1 to 5, inclusive, indicating which iamge to get. 1 is the most recent image, and 5 being the 5th most recent.",
					},
					"who": Object{
						"type":        "string",
						"description": "The username to get the previous messages for. May not be empty.",
					},
				},
				"required": []string{"count", "who"},
			},
		},
	}
}

type GetPreviousNMessagesFromUserArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

func GetPreviousNMessagesFromUser(guild string, channel string, args GetPreviousNMessagesFromUserArgs) ([]openai.ChatMessagePart, bool) {
	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		return []openai.ChatMessagePart{}, false
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
	needsVision := false

	for _, v := range messages {
		next, vision := convertMessageToContent(v, "%s")
		content = append(content, next...)

		needsVision = needsVision || vision
	}

	return content, needsVision
}

type GetLastImageArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

func GetLastImage(guild string, channel string, args GetLastImageArgs) ([]openai.ChatMessagePart, bool) {
	log.Print("Getting last image with: ")
	log.Printf("%+v", args)

	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		log.Print("Found no relevant messages")
		return []openai.ChatMessagePart{}, false
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

			embeds := convertEmbedsToContent(v)
			attachments := convertAttachmentsToContent(v)

			content = append(content, embeds...)
			content = append(content, attachments...)
		}
	}

	return content, true
}
