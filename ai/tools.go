package ai

import (
	"encoding/json"
	"log"

	"github.com/albert-wang/rawr-discordbot/ai/jikan"
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
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name: "get_anime_information",
				Description: `Gets, given an anime name in english, information about that anime. Sometimes, the anime will have multiple seasons. If there are mulitple seasons, try to
				look up information for the most recent season that has aired or is currently airing.
					`,
				Parameters: Object{
					"type":                 "object",
					"additionalProperties": false,
					"properties": Object{
						"anime": Object{
							"type":        "string",
							"description": "The title, in english, of the anime to get information for.",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name: "get_anime_details",
				Description: `
					Gets, given an anime id from get_anime_information, detailed information about that anime.
					This includes staff, characters and voice actors.
					When using this tool and getting information, make sure to also use the provided URL to provide a link for more context.
				`,
				Parameters: Object{
					"type":                 "object",
					"additionalProperties": false,
					"properties": Object{
						"anime": Object{
							"type":        "integer",
							"description": "The ID of the anime to get details for",
						},
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

type GetAnimeInformationArgs struct {
	Anime string `json:"anime"`
}

func GetAnimeInformation(guild string, channel string, args GetAnimeInformationArgs) []openai.ChatMessagePart {
	anime, err := jikan.GetAnime(args.Anime)
	if err != nil {
		log.Print(err)
		return nil
	}

	if len(anime) == 0 {
		return []openai.ChatMessagePart{
			{
				Type: "text",
				Text: "No anime found",
			},
		}
	}

	formatted, err := json.MarshalIndent(anime, "", "  ")
	if err != nil {
		log.Print(err)
		return nil
	}

	return []openai.ChatMessagePart{
		{
			Type: "text",
			Text: string(formatted),
		},
	}
}

type GetAnimeDetailsArgs struct {
	MALID int `json:"anime"`
}

func GetAnimeDetails(guild string, channel string, args GetAnimeDetailsArgs) []openai.ChatMessagePart {
	details, err := jikan.GetAnimeDetails(args.MALID)
	if err != nil {
		log.Print(err)
		return nil
	}

	formatted, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		log.Print(err)
		return nil
	}

	return []openai.ChatMessagePart{
		{
			Type: "text",
			Text: string(formatted),
		},
	}
}
