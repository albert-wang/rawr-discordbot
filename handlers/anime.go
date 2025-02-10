package handlers

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
	openai "github.com/sashabaranov/go-openai"
)

type animeStatus struct {
	Name            string
	CurrentEpisode  int64
	LastModified    time.Time
	CrunchyrollCode string // In value?query-param format
	Subgroup        string
}

func (a *animeStatus) FormattedTime() string {
	return a.LastModified.Format("Mon, January 02")
}

func clamp(v, l, h int64) int64 {
	if v < l {
		return l
	}

	if v > h {
		return h
	}

	return v
}

func fuzzySearch(lookup string, animes map[string]animeStatus) (string, *animeStatus) {
	candidates := []animeStatus{}
	key := ""
	for k, v := range animes {
		if strings.HasPrefix(k, lookup) {
			candidates = append(candidates, v)
			key = k
		}
	}

	if len(candidates) != 1 {
		return "", nil
	}

	return key, &candidates[0]
}

func AnimeStatus(m *discordgo.MessageCreate, args []string) error {
	if len(args) < 1 {
		chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime <del|mv|incr|decr|set|list> <name> [<value>]")
	}

	conn := Redis.Get()
	defer conn.Close()

	key := makeKey("animestatus")
	animes := map[string]animeStatus{}
	deserialize(conn, key, &animes)

	// Supports del, mv, incr, decr, set, list
	switch args[0] {
	case "del":
		{
			if len(args) != 2 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime del <name>")
				return nil
			}

			delete(animes, args[1])
			break
		}
	case "mv":
		{
			if len(args) != 3 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime mv <name> <new>")
				return nil
			}

			_, ok := animes[args[2]]
			v, ok2 := animes[args[1]]

			if ok || !ok2 {
				chat.SendPrivateMessageTo(m.Author.ID, "!anime mv cannot overwrite elements, or source element did not exist")
			}

			v.Name = args[2]
			animes[args[2]] = v
			delete(animes, args[1])
			break
		}

	case "set":
		{
			if len(args) != 3 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime set <name> <ep#>")
				return nil
			}

			episode, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}

			episode = clamp(episode, -10, 1000)
			v, ok := animes[args[1]]
			if !ok {
				animes[args[1]] = animeStatus{args[1], episode, time.Now(), "-", "-"}
			} else {
				v.CurrentEpisode = episode
				v.LastModified = time.Now()
				animes[args[1]] = v
			}

			v = animes[args[1]]
			chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s)", v.Name, v.CurrentEpisode, v.LastModified.Format("Mon, January 02")))
			break
		}
	case "sub":
		{
			if len(args) != 3 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime sub <name> <value>")
				return nil
			}

			k, anime := fuzzySearch(args[1], animes)
			if anime == nil {
				chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("I don't know anything about %s!", args[1]))
				break
			}

			sub := args[2]
			anime.Subgroup = sub
			anime.LastModified = time.Now()
			animes[k] = *anime

			chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %s (%s)", anime.Name, anime.Subgroup, anime.LastModified.Format("Mon, January 02")))
			break
		}
	case "cr":
		{
			if len(args) != 3 && len(args) != 2 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime cr <name> <value>")
				return nil
			}

			if len(args) == 2 {
				_, anime := fuzzySearch(args[1], animes)
				if anime == nil {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("I don't know anything about %s!", args[1]))
					break
				}

				if anime.CrunchyrollCode == "" {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("No crunchyroll data for %s exists", anime.Name))
					break
				}

				data, err := ParseCrunchyrollString(anime.CrunchyrollCode)
				if err != nil {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("The crunchydata is wrong? (%s)", anime.CrunchyrollCode))
					break
				}

				ep, err := GetBestGuessCrunchyrollLink(data, anime.CurrentEpisode)
				if err != nil {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("Couldn't get episode info, %s", err.Error()))
					break
				}

				chat.SendMessageToChannel(m.ChannelID, ep.Link)
				break
			}

			if len(args) == 3 {
				k, anime := fuzzySearch(args[1], animes)
				if anime == nil {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("I don't know anything about %s!", args[1]))
					break
				}

				crunchyrollCode := args[2]

				anime.CrunchyrollCode = crunchyrollCode
				anime.LastModified = time.Now()
				animes[k] = *anime

				chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s -> %s", anime.Name, anime.CrunchyrollCode))
			}

			break
		}
	case "decr", "incr":
		{
			if len(args) != 2 {
				chat.SendPrivateMessageTo(m.Author.ID, fmt.Sprintf("Usage: !anime %s <name>", args[0]))
				return nil
			}

			delta := int64(-1)
			if args[0] == "incr" {
				delta = 1
			}

			k, anime := fuzzySearch(args[1], animes)
			if anime == nil {
				chat.SendPrivateMessageTo(m.Author.ID, fmt.Sprintf("Usage: !anime %s <name> requires a valid name, or name was ambiguous", args[0]))
				return nil
			} else {
				anime.CurrentEpisode = anime.CurrentEpisode + delta
				anime.CurrentEpisode = clamp(anime.CurrentEpisode, -10, 1000)

				if args[0] == "incr" {
					anime.LastModified = time.Now()
				}

				animes[k] = *anime
				if anime.Subgroup != "" {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s) [%s]", anime.Name, anime.CurrentEpisode, anime.LastModified.Format("Mon, Jan 02"), anime.Subgroup))
				} else {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s)", anime.Name, anime.CurrentEpisode, anime.LastModified.Format("Mon, Jan 02")))
				}
			}
			break
		}
	case "suggest":
		{
			SuggestAnime(m, []string{})
			break
		}
	case "list":
		{
			sortByTime := false
			if len(args) == 2 {
				if args[1] == "updated" {
					sortByTime = true
				}
			}

			animes := []animeStatus{}
			for _, v := range animes {
				animes = append(animes, v)
			}

			if sortByTime {
				sort.Slice(animes, func(i, j int) bool {
					return animes[i].LastModified.Before(animes[j].LastModified)
				})
			} else {
				sort.Slice(animes, func(i, j int) bool {
					return animes[i].Name < animes[j].Name
				})
			}

			table := CreateTable(
				TableHeader{"Title", TableAlignRight},
				TableHeader{"Episode", TableAlignRight},
				TableHeader{"Subs?", TableAlignRight},
				TableHeader{"Last Updated", TableAlignLeft},
			)

			stringOr := func(v string, def string) string {
				trimmed := strings.TrimSpace(v)
				if len(trimmed) == 0 || trimmed == "-" {
					return def
				}

				return trimmed
			}

			for _, a := range animes {
				table.AddRow(
					strings.TrimSpace(a.Name),
					fmt.Sprintf("%d", a.CurrentEpisode),
					stringOr(a.Subgroup, " "),
					a.LastModified.Format("Mon, Jan 02"),
				)
			}

			chat.SendMessageToChannel(m.ChannelID, "```Markdown\n"+table.Render()+"```")
		}
	}

	serialize(conn, key, &animes)
	return nil
}

func Countdown(m *discordgo.MessageCreate, args []string) error {
	start := int64(3)
	var err error

	if len(args) == 1 {
		start, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			start = 3
		}
	}

	if start > 30 {
		start = 30
	}

	for i := int64(0); i < start; i++ {
		chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%d", start-i))
		time.Sleep(time.Second)
	}

	chat.SendMessageToChannel(m.ChannelID, "g")
	return nil
}

var (
	junbiCount, junbiMembers int64
)

func JunbiOK(m *discordgo.MessageCreate, args []string) error {
	junbiMembers = 3
	var err error

	if len(args) == 1 {
		junbiMembers, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			junbiMembers = 3
		}
	}

	if junbiCount == 0 {
		chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("Junbi OK?"))
		time.Sleep(300 * time.Millisecond)
		chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("Type !rdy to confirm!"))
		junbiCount++
		return nil
	}

	if junbiCount < junbiMembers {
		count := int64(junbiMembers - junbiCount)
		chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("Waiting on %d more!", count))
		junbiCount++
		return nil
	} else {
		Countdown(m, []string{"3"})
		junbiCount = 0
	}
	return nil
}

func SuggestAnime(m *discordgo.MessageCreate, args []string) error {

	complete := chat.ShowTyping(m.ChannelID)
	defer complete()

	conn := Redis.Get()
	defer conn.Close()

	key := makeKey("animestatus")
	res := map[string]animeStatus{}
	deserialize(conn, key, &res)

	tplText := `
	Given the following list of items, pick four titles to watch. Take into account how recently they have been watched, with ones
	that have not been watched recently having slightly higher priority. Don't suggest anything that was last watched over
	3 months ago.

	{{ range .Animes }}{{ .Name }}, {{ .LastModified.Format "Mon, January 02 2006" }}
{{ end }}
	`

	buff := bytes.NewBuffer(nil)
	tpl, err := template.New("anime").Funcs(template.FuncMap{
		"pad": func(amount int, spacer string, val string) string {
			if len(val) < amount {
				return strings.Repeat(spacer, amount-len(val)) + val
			}

			return val
		},
	}).Parse(tplText)

	if err != nil {
		chat.SendMessageToChannel(m.ChannelID, err.Error())
	}

	maximumTitle := 0
	for _, v := range res {
		if len(v.Name) > maximumTitle {
			maximumTitle = len(v.Name)
		}
	}

	err = tpl.Execute(buff, map[string]interface{}{
		"Animes": res,
		"Len":    maximumTitle,
	})

	messages := GenerateMessagesWithContext(m.GuildID, m.ChannelID, 32)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleUser,
		MultiContent: textContent(buff.String()),
	})

	UnboundedRespondToContent(m.GuildID, m.ChannelID, messages, true)
	return nil
}
