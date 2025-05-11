package anime

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type SubArguments struct {
	Anime  string `arg:"positional"`
	Subber string `arg:"positional"`
}

func (db *Database) Sub(msg *discordgo.MessageCreate, args *SubArguments) (string, error) {
	name, anime := db.search(args.Anime)
	if anime == nil {
		return UnknownAnime(args.Anime), nil
	}

	anime.Subgroup = args.Subber
	return fmt.Sprintf("Set %s sub=%s", name, args.Subber), nil
}

type InspectArguments struct {
	Anime string `arg:"positional"`
}

func (db *Database) Inspect(msg *discordgo.MessageCreate, args *SubArguments) (string, error) {
	name, anime := db.search(args.Anime)
	if anime == nil {
		return UnknownAnime(args.Anime), nil
	}

	b, err := json.MarshalIndent(anime, "  ", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("```%s = %s```", name, string(b)), nil
}

type SrcArguments struct {
	Anime  string  `arg:"positional"`
	Source *string `arg:"positional"`
	Clear  bool    `arg:"--remove"`
}

func (db *Database) Src(msg *discordgo.MessageCreate, args *SrcArguments) (string, error) {
	name, anime := db.search(args.Anime)
	if anime == nil {
		return UnknownAnime(args.Anime), nil
	}

	if args.Clear {
		anime.EpisodeSource = ""
		db.animes[name] = *anime

		return fmt.Sprintf("Set %s src=", name), nil
	}

	if args.Source != nil {
		anime.EpisodeSource = *args.Source
		db.animes[name] = *anime

		return fmt.Sprintf("Set %s src=%s", name, anime.EpisodeSource), nil
	}

	if anime.EpisodeSource == "" {
		return NoSource(name), nil
	}

	link, err := anime.GetSourceLink()
	if err != nil {
		return CannotSource(name, err), nil
	}

	return link, nil
}
