package handlers

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
)

type animeStatus struct {
	Name           string
	CurrentEpisode int64
	LastModified   time.Time
	Day            string
	Subgroup       string
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

func AnimeStatus(m *discordgo.MessageCreate, args []string) error {
	if len(args) < 1 {
		chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime <del|mv|incr|decr|set|list> <name> [<value>]")
	}

	conn := Redis.Get()
	defer conn.Close()

	key := makeKey("animestatus")
	res := map[string]animeStatus{}
	deserialize(conn, key, &res)

	// Supports del, mv, incr, decr, set, list
	switch args[0] {
	case "del":
		{
			if len(args) != 2 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime del <name>")
				return nil
			}

			delete(res, args[1])
			break
		}
	case "mv":
		{
			if len(args) != 3 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime mv <name> <new>")
				return nil
			}

			_, ok := res[args[2]]
			v, ok2 := res[args[1]]

			if ok || !ok2 {
				chat.SendPrivateMessageTo(m.Author.ID, "!anime mv cannot overwrite elements, or source element did not exist")
			}

			v.Name = args[2]
			res[args[2]] = v
			delete(res, args[1])
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
			v, ok := res[args[1]]
			if !ok {
				res[args[1]] = animeStatus{args[1], episode, time.Now(), "-", "-"}
			} else {
				v.CurrentEpisode = episode
				v.LastModified = time.Now()
				res[args[1]] = v
			}

			v = res[args[1]]
			chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s)", v.Name, v.CurrentEpisode, v.LastModified.Format("Mon, January 02")))
			break
		}
	case "sub":
		{
			if len(args) != 3 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime sub <name> <value>")
				return nil
			}

			sub := args[2]

			v, ok := res[args[1]]
			if !ok {
				res[args[1]] = animeStatus{args[1], 0, time.Now(), "", sub}
			} else {
				v.Subgroup = sub
				v.LastModified = time.Now()
				res[args[1]] = v
			}

			v = res[args[1]]
			chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s)", v.Name, v.Subgroup, v.LastModified.Format("Mon, January 02")))
			break
		}
	case "day":
		{
			if len(args) != 3 {
				chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime day <name> <value>")
				return nil
			}

			valid := []string{"sun", "mon", "tue", "wen", "thr", "fri", "sat", "-"}
			found := false
			for _, v := range valid {
				if args[2] == v {
					found = true
					break
				}
			}

			if !found {
				chat.SendPrivateMessageTo(m.Author.ID, "Invalid day")
				return nil
			}

			day := args[2]

			v, ok := res[args[1]]
			if !ok {
				res[args[1]] = animeStatus{args[1], 0, time.Now(), day, "-"}
			} else {
				v.Day = day
				v.LastModified = time.Now()
				res[args[1]] = v
			}

			v = res[args[1]]
			chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s)", v.Name, v.CurrentEpisode, v.LastModified.Format("Mon, January 02")))
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

			candidates := []animeStatus{}
			key := ""
			for k, v := range res {
				if strings.HasPrefix(k, args[1]) {
					candidates = append(candidates, v)
					key = k
				}
			}

			if len(candidates) != 1 {
				chat.SendPrivateMessageTo(m.Author.ID, fmt.Sprintf("Usage: !anime %s <name> requires a valid name, or name was ambiguous", args[0]))
				return nil
			} else {
				v := candidates[0]

				v.CurrentEpisode = v.CurrentEpisode + delta
				v.CurrentEpisode = clamp(v.CurrentEpisode, -10, 1000)

				if args[0] == "incr" {
					v.LastModified = time.Now()
				}

				res[key] = v
				if v.Subgroup != "" {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s) [%s]", v.Name, v.CurrentEpisode, v.LastModified.Format("Mon, Jan 02"), v.Subgroup))
				} else {
					chat.SendMessageToChannel(m.ChannelID, fmt.Sprintf("%s - %d (%s)", v.Name, v.CurrentEpisode, v.LastModified.Format("Mon, Jan 02")))
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
			tplText := `Markdown
{{ pad .Len " " "Title" }} | Episode | Last Updated
{{ pad .Len "-" "-----" }}-+---------+-------------
{{ range .Animes }}{{ pad $.Len " " .Name }} | {{ with $x := printf "%d" .CurrentEpisode }}{{ pad 7 " " $x }}{{ end }} | {{ .LastModified.Format "Mon, Jan 02" }}
{{ end }}`

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

			if err != nil {
				log.Print(err)
			}

			chat.SendMessageToChannel(m.ChannelID, "```"+buff.String()+"```")
		}
	}

	serialize(conn, key, &res)
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
	conn := Redis.Get()
	defer conn.Close()

	key := makeKey("animestatus")
	res := map[string]animeStatus{}
	deserialize(conn, key, &res)

	tplText := `
	Given the following list of items, pick four titles to watch. Take into account how recently they have been watched, with ones
	that have not been watched recently having slightly higher priority. Respond as a competitive, lightly flustered, barely tsundere, cute anime school girl.

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

	log.Print(buff.String())
	UnboundedRespondToPrompt(m.ChannelID, buff.String(), []string{})
	return nil
}
