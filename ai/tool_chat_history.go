package ai

import (
	"log"

	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"

	"github.com/albert-wang/rawr-discordbot/chat"
)

type GetPreviousNMessagesFromUserArgs struct {
	Count int    `json:"count"`
	Who   string `json:"who"`
}

type GetLastImageArgs struct {
	MessageID string `json:"message_id"`
}

type GetMessageByIdArgs struct {
	MessageID string `json:"message_id"`
}

func init() {
	type Object = map[string]any

	DefineTool(
		responses.FunctionToolParam{
			Name:        "get_previous_n_messages_from_user",
			Description: param.NewOpt(`Use this when you want context around a message. This gets the previous N messages for a given user, or all users.`),
			Strict:      param.NewOpt(true),
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"count": Object{
						"type":        "integer",
						"description": "How many messages to get. Maximum 30, minimum 5",
					},
					"who": Object{
						"type":        "string",
						"description": "Who to get the message for. May be the empty string to get everyone's entries. This is the username of the person to get messages for",
					},
				},
				"required": []string{"count", "who"},
			},
		},
		getPreviousNMessagesFromUser,
	)

	DefineTool(
		responses.FunctionToolParam{
			Name:        "get_images_for_message",
			Description: param.NewOpt("Fetch the actual image content for a message. Messages with images have image_count > 0"),
			Strict:      param.NewOpt(true),
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"message_id": Object{
						"type":        "string",
						"description": "The Discord message ID to fetch images from. Read it from the message_id attribute in the tag that prefixes each message in the conversation.",
					},
				},
				"required": []string{"message_id"},
			},
		},
		getMessageImages,
	)

	DefineTool(
		responses.FunctionToolParam{
			Name:        "get_message_by_id",
			Description: param.NewOpt(`Use this when a message references a previous message to get the contents of the previous message`),
			Strict:      param.NewOpt(true),
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"message_id": Object{
						"type":        "string",
						"description": "The message ID to get. Read it from the reference attribute in the tag that prefixes each message in the conversation.",
					},
				},
				"required": []string{"message_id"},
			},
		},
		getMessageById,
	)
}

func clampCount(requested, available int) int {
	if requested < 5 {
		requested = 5
	}

	if requested > 30 {
		requested = 30
	}

	if requested < available {
		return requested
	}

	if requested >= available {
		return available
	}

	return requested
}

func getPreviousNMessagesFromUser(guild, channel string, args GetPreviousNMessagesFromUserArgs) []responses.ResponseInputContentUnionParam {
	messages := chat.GetPreviousMessageFromUser(guild, channel, args.Who)
	if len(messages) == 0 {
		return nil
	}

	max := clampCount(args.Count, len(messages))
	messages = messages[:max]

	content := []responses.ResponseInputContentUnionParam{}
	for _, v := range messages {
		content = append(content, MessageContent(v, ConversionOptions{
			IncludeMedia: true,
		})...)
	}
	return content
}

func getMessageImages(guild, channel string, args GetLastImageArgs) []responses.ResponseInputContentUnionParam {
	log.Printf("Getting last image with: %+v", args)

	message := chat.GetMessage(guild, channel, args.MessageID)
	if message == nil {
		log.Print("Found no relevant messages")
		return nil
	}

	content := []responses.ResponseInputContentUnionParam{}
	content = append(content, EmbedsContent(message)...)
	content = append(content, AttachmentsContent(message)...)
	return content
}

func getMessageById(guild, channel string, args GetMessageByIdArgs) []responses.ResponseInputContentUnionParam {
	message := chat.GetMessage(guild, channel, args.MessageID)
	if message == nil {
		log.Print("Found no relevant messages")
		return nil
	}

	return MessageContent(message, ConversionOptions{
		IncludeMedia: true,
	})
}
