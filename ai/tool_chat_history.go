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

func init() {
	type Object = map[string]any

	DefineTool(
		responses.FunctionToolParam{
			Name:        "get_previous_n_messages_from_user",
			Description: param.NewOpt("Gets the previous N messages for a given user"),
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
}

// clampCount clamps to [1, min(5, available)]. Returns 0 if nothing fits,
// matching the prior "off by one" behavior where max <= len(messages)-1.
func clampCount(requested, available int) int {
	max := 30
	if requested > 0 && requested < 5 {
		max = requested
	}
	if max > available-1 {
		max = available - 1
	}
	return max
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
