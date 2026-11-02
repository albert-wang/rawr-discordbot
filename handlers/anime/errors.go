package anime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/albert-wang/rawr-discordbot/chat"
)

// UnknownAnime says like didn't match exactly one anime. If it matched
// several, they're listed so the right slug can be picked.
func (db *Database) UnknownAnime(like string) string {
	matches := []Status{}
	for k, v := range db.animes {
		if strings.HasPrefix(k, like) {
			matches = append(matches, v)
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("I don't know anything about %s", like)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].LastModified.After(matches[j].LastModified)
	})

	table := chat.CreateTable(
		chat.TableHeader{Title: "Title", Align: chat.TableAlignLeft},
		chat.TableHeader{Title: "Slug", Align: chat.TableAlignLeft},
		chat.TableHeader{Title: "Last Watched", Align: chat.TableAlignLeft},
	)

	for _, a := range matches {
		date := a.LastModified.Format("Mon, Jan 02")
		if a.LastModified.Year() < 2000 {
			date = ""
		}

		table.AddRow(strings.TrimSpace(a.DisplayName), strings.TrimSpace(a.Name), date)
	}

	return fmt.Sprintf("%s could be any of these:\n```ansi\n%s```", like, table.Render())
}

func NoSource(anime string) string {
	return fmt.Sprintf("No source for %s", anime)
}

func CannotSource(anime string, err error) string {
	return fmt.Sprintf("Couldn't get episode info for anime=%s err=%s", anime, err.Error())
}
