package ai

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"

	"github.com/albert-wang/rawr-discordbot/ai/jikan"
)

// jikanFailed reports a Jikan API failure to the model. Returning nil here
// would send back an empty tool result, which the model reads as "give up";
// instead, tell it what happened and point it at web search as a fallback.
func jikanFailed(err error) []responses.ResponseInputContentUnionParam {
	log.Print(err)
	return TextContent(fmt.Sprintf(
		`The anime database (Jikan/MyAnimeList) request failed: %v.
		Do not give up and do not tell the user you failed yet — answer their question a different way.
		Use the web_search tool instead (myanimelist.net, anilist.co, or Wikipedia are good sources).`, err))
}

func init() {
	type Object = map[string]any

	DefineTool(
		responses.FunctionToolParam{
			Name: "get_anime_information",
			Description: param.NewOpt(`Use this when a user wants information about an anime by name, in English.
				This returns information in JSON format.
				The name may be romanicized japanese.
				Sometimes, the returned anime will have multiple seasons with the same name. Prefer getting information
				about the most recent season unless otherwise specified.`),
			Strict: param.NewOpt(true),
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"anime": Object{
						"type":        "string",
						"description": "The title of the anime to look up. May be in English or romanized Japanese.",
					},
				},
				"required": []string{"anime"},
			},
		},
		getAnimeInformation,
	)

	DefineTool(
		responses.FunctionToolParam{
			Name: "get_anime_details",
			Description: param.NewOpt(`Use this to extend your knowledge about an anime, given an anime id from get_anime_information.
				Returns staff, characters, voice actors, and other data for the anime.
				When using this tool and getting information, make sure to also use the provided URL to provide a link for more context.`),
			Strict: param.NewOpt(true),
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

	DefineTool(
		responses.FunctionToolParam{
			Name: "get_seasonal_anime",
			Description: param.NewOpt(`Use this to obtain a list of anime that are airing for a given season.
				Seasons are designated by a numerical year, and a season - one of 'winter', 'spring', 'summer', or 'fall'.
				Know that winter generally means January to March, spring April to June, summer is June to September, and
				fall is October to December.

				This returns 32 entries, ordered by popularity.

				Each result has a 'url' field — link it from the anime's title when you mention it.
			`),
			Strict: param.NewOpt(true),
			Parameters: Object{
				"type":                 "object",
				"additionalProperties": false,
				"properties": Object{
					"year": Object{
						"type":        "integer",
						"description": "The year to query for",
					},
					"season": Object{
						"type":        "string",
						"description": "The season. One of 'winter', 'spring', 'summer', or 'fall'",
					},
				},
				"required": []string{"year", "season"},
			},
		},
		getSeasonalAnime,
	)
}

type GetAnimeInformationArgs struct {
	Anime string `json:"anime"`
}

func getAnimeInformation(guild, channel string, args GetAnimeInformationArgs) []responses.ResponseInputContentUnionParam {
	anime, err := jikan.GetAnime(args.Anime)
	if err != nil {
		return jikanFailed(err)
	}
	if len(anime) == 0 {
		return TextContent(`No anime found for that title.
			Retry this tool with a different spelling or romanization, or fall back to the web_search tool.`)
	}

	formatted, err := json.MarshalIndent(anime, "", "  ")
	if err != nil {
		log.Print(err)
		return nil
	}
	return TextContent(string(formatted))
}

type GetAnimeDetailsArgs struct {
	MALID int `json:"anime"`
}

func getAnimeDetails(guild, channel string, args GetAnimeDetailsArgs) []responses.ResponseInputContentUnionParam {
	details, err := jikan.GetAnimeDetails(args.MALID)
	if err != nil {
		return jikanFailed(err)
	}

	formatted, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		log.Print(err)
		return nil
	}
	return TextContent(string(formatted))
}

type GetSeasonalAnimeArgs struct {
	Year   int    `json:"year"`
	Season string `json:"season"`
}

func getSeasonalAnime(guild, channel string, args GetSeasonalAnimeArgs) []responses.ResponseInputContentUnionParam {
	if args.Year < 1995 {
		return TextContent("Year must be 1995 or later")
	}

	switch args.Season {
	case "winter", "spring", "summer", "fall":
	default:
		return TextContent(`season must be one of "winter", "spring", "summer", "fall"`)
	}

	seasonal, err := jikan.GetSeason(args.Year, args.Season)
	if err != nil {
		return jikanFailed(err)
	}

	slices.SortFunc(seasonal, func(a jikan.AnimeInformation, b jikan.AnimeInformation) int {
		return cmp.Compare(b.Popularity, a.Popularity)
	})

	if len(seasonal) > 32 {
		seasonal = seasonal[:32]
	}

	formatted, err := json.MarshalIndent(seasonal, "", "  ")
	if err != nil {
		log.Print(err)
		return nil
	}
	return TextContent(string(formatted))
}
