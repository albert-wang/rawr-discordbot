package ai

import (
	"encoding/json"
	"log"

	openai "github.com/sashabaranov/go-openai"

	"github.com/albert-wang/rawr-discordbot/ai/jikan"
)

type GetAnimeInformationArgs struct {
	Anime string `json:"anime"`
}

type GetAnimeDetailsArgs struct {
	MALID int `json:"anime"`
}

func init() {
	type Object = map[string]any

	DefineTool(
		openai.FunctionDefinition{
			Name: "get_anime_information",
			Description: `Gets, given an anime name in english, information about that anime. Sometimes, the anime will have multiple seasons. If there are mulitple seasons, try to
				look up information for the most recent season that has aired or is currently airing.`,
			Strict: true,
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"anime": Object{
						"type":        "string",
						"description": "The title, in english, of the anime to get information for.",
					},
				},
				"required": []string{"anime"},
			},
		},
		getAnimeInformation,
	)

	DefineTool(
		openai.FunctionDefinition{
			Name: "get_anime_details",
			Description: `Gets, given an anime id from get_anime_information, detailed information about that anime.
				This includes staff, characters and voice actors.
				When using this tool and getting information, make sure to also use the provided URL to provide a link for more context.`,
			Strict: true,
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"anime": Object{
						"type":        "integer",
						"description": "The ID of the anime to get details for",
					},
				},
				"required": []string{"anime"},
			},
		},
		getAnimeDetails,
	)
}

func getAnimeInformation(guild, channel string, args GetAnimeInformationArgs) []openai.ChatMessagePart {
	anime, err := jikan.GetAnime(args.Anime)
	if err != nil {
		log.Print(err)
		return nil
	}
	if len(anime) == 0 {
		return TextContent("No anime found")
	}

	formatted, err := json.MarshalIndent(anime, "", "  ")
	if err != nil {
		log.Print(err)
		return nil
	}
	return TextContent(string(formatted))
}

func getAnimeDetails(guild, channel string, args GetAnimeDetailsArgs) []openai.ChatMessagePart {
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
	return TextContent(string(formatted))
}
