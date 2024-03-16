package handlers

import (
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
						"description": "How many messages to get. Maximum 8, minimum 1",
					},
					"who": Object{
						"type":        "string",
						"description": "The to get the previous messages for. May be empty, which will get all user's messages",
					},
				},
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
						"description": "Must be 1",
					},
				},
			},
		},
	}
}

type GetPreviousNMessagesFromUserArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

func GetPreviousNMessagesFromUser(guild string, channel string, args GetPreviousNMessagesFromUserArgs) ([]openai.ChatContent, bool) {
	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		return []openai.ChatContent{}, false
	}

	max := 5
	if args.Count < 5 && args.Count > 0 {
		max = args.Count
	}

	if max > len(messages)-1 {
		max = len(messages) - 1
	}

	messages = messages[:max]
	content := []openai.ChatContent{}
	needsVision := false

	for _, v := range messages {
		next, vision := convertMessageToContent(v, "")
		content = append(content, next...)

		needsVision = needsVision || vision
	}

	return content, needsVision
}

type GetLastImageArgs struct {
	Count int `json:"count"`
}

func GetLastImage(guild string, channel string, args GetLastImageArgs) ([]openai.ChatContent, bool) {
	return []openai.ChatContent{}, false
}
