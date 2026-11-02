package anime

import (
	"log"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/bwmarrin/discordgo"
)

// Handle runs an anime command and returns what to reply with.
func Handle(command string, args []string) (string, error) {
	db := LoadDatabase()

	resp, err := db.Handle(command, args)
	if err != nil {
		log.Println(err)
		return "", err
	}

	db.Save()
	return resp, nil
}

// Autocomplete suggests up to 25 shows whose slug contains typed, most
// recently watched first.
func Autocomplete(typed string) []*discordgo.ApplicationCommandOptionChoice {
	db := LoadDatabase()

	matches := []Status{}
	for k, v := range db.animes {
		if strings.Contains(strings.ToLower(k), strings.ToLower(typed)) {
			matches = append(matches, v)
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].LastModified.After(matches[j].LastModified)
	})

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for _, v := range matches {
		if len(choices) == 25 {
			break
		}

		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  v.Short(),
			Value: v.Name,
		})
	}

	return choices
}

func genericHandle[T any](db *Database, args []string, cb func(db *Database, args *T) (string, error)) (string, error) {
	cfg := arg.Config{
		Program:           "nvg-tan",
		IgnoreEnv:         true,
		StrictSubcommands: true,
	}

	typedArgs := new(T)
	parser, err := arg.NewParser(cfg, typedArgs)
	if err != nil {
		log.Printf("args=%+v\n", args)
		log.Printf("result=%+v\n", typedArgs)

		log.Print(err)
		return "", err
	}

	err = parser.Parse(args)
	if err != nil {
		log.Print(err)
		return "", err
	}

	return cb(db, typedArgs)
}
