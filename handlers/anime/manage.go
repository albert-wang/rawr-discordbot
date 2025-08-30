package anime

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
)

type DeleteArguments struct {
	Name string `arg:"positional"`
}

func (db *Database) Delete(msg *discordgo.MessageCreate, args *DeleteArguments) (string, error) {
	name, status := db.search(args.Name)
	if status == nil {
		return UnknownAnime(args.Name), nil
	}

	delete(db.animes, name)
	return fmt.Sprintf("Deleted %s", name), nil
}

type MoveArguments struct {
	From string `arg:"positional"`
	To   string `arg:"positional"`
}

func (db *Database) Move(msg *discordgo.MessageCreate, args *MoveArguments) (string, error) {
	first, ok := db.search(args.From)
	if ok == nil {
		return UnknownAnime(args.From), nil
	}

	_, to := db.animes[args.To]
	if to {
		return "mv cannot overwrite elements", nil
	}

	db.animes[args.To] = *ok
	delete(db.animes, first)

	return fmt.Sprintf("Moved %s -> %s", first, args.To), nil
}

type SetArguments struct {
	Anime   string `arg:"positional"`
	Episode int    `arg:"positional"`
}

func (db *Database) Set(msg *discordgo.MessageCreate, args *SetArguments) (string, error) {
	if args.Episode < -10 || args.Episode > 1000 {
		return fmt.Sprintf("Invalid episode number: %d", args.Episode), nil
	}

	name, target := db.search(args.Anime)
	if target == nil {
		target = &Status{
			Name:           args.Anime,
			CurrentEpisode: int64(args.Episode),
			LastModified:   time.Now(),
			EpisodeSource:  "",
			Subgroup:       "",
		}

		db.animes[args.Anime] = *target
	} else {
		target.CurrentEpisode = int64(args.Episode)
		target.LastModified = time.Now()

		db.animes[name] = *target
	}

	return target.Short(), nil
}

type ListArguments struct {
	SortByTime bool `arg:"-u,--updated"`
	All        bool `arg:"-a,--all" help:"Even show shows that are over 3 months old"`
}

func Filter[T any](seq []T, p func(a *T, i int) bool) []T {
	res := make([]T, 0, len(seq))
	for i, _ := range seq {
		if p(&seq[i], i) {
			res = append(res, seq[i])
		}
	}

	return res
}

func (db *Database) List(msg *discordgo.MessageCreate, args *ListArguments) (string, error) {
	sorted := []Status{}
	for _, v := range db.animes {
		sorted = append(sorted, v)
	}

	if args.SortByTime {
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].LastModified.Before(sorted[j].LastModified)
		})
	} else {
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Name < sorted[j].Name
		})
	}

	if !args.All {
		sorted = Filter(sorted, func(a *Status, _ int) bool {
			if a.Name == "lotgh" {
				return true
			}

			return time.Since(a.LastModified) < 90*time.Hour*24
		})
	}

	table := chat.CreateTable(
		chat.TableHeader{"Title", chat.TableAlignRight},
		chat.TableHeader{"Episode", chat.TableAlignRight},
		chat.TableHeader{"Last Updated", chat.TableAlignLeft},
	)

	for _, a := range sorted {
		table.AddRow(
			strings.TrimSpace(a.Name),
			fmt.Sprintf("%d", a.CurrentEpisode),
			a.LastModified.Format("Mon, Jan 02"),
		)
	}

	return "```Markdown\n" + table.Render() + "```", nil
}

type IncrDecrArguments struct {
	Anime        string `arg:"positional"`
	Delta        int    `arg:"positional" default:"1"`
	NoSource     bool   `arg:"-q,--quiet"`
	NoUpdateTime bool   `arg:"-t,--time-stop"`
}

func (db *Database) Incr(msg *discordgo.MessageCreate, args *IncrDecrArguments) (string, error) {
	name, anime := db.search(args.Anime)
	if anime == nil {
		return UnknownAnime(args.Anime), nil
	}

	anime.CurrentEpisode += int64(args.Delta)
	anime.LastModified = time.Now()
	db.animes[name] = *anime

	chat.SendMessageToChannel(msg.ChannelID, anime.Short())

	if !args.NoSource {
		link, err := anime.GetSourceLink()
		if err != nil {
			return CannotSource(name, err), nil
		}

		return link, nil
	}

	return "", nil
}

func (db *Database) Decr(msg *discordgo.MessageCreate, args *IncrDecrArguments) (string, error) {
	name, anime := db.search(args.Anime)
	if anime == nil {
		return UnknownAnime(args.Anime), nil
	}

	anime.CurrentEpisode -= int64(args.Delta)
	anime.LastModified = time.Now()
	db.animes[name] = *anime

	chat.SendMessageToChannel(msg.ChannelID, anime.Short())

	if !args.NoSource {
		link, err := anime.GetSourceLink()
		if err != nil {
			return CannotSource(name, err), nil
		}

		return link, nil
	}

	return "", nil
}
