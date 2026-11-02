package handlers

import (
	"fmt"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/handlers/anime"
	"github.com/bwmarrin/discordgo"
)

var animeOption = &discordgo.ApplicationCommandOption{
	Type:         discordgo.ApplicationCommandOptionString,
	Name:         "anime",
	Description:  "Slug of the show, start typing to search",
	Required:     true,
	Autocomplete: true,
}

var deltaOption = &discordgo.ApplicationCommandOption{
	Type:        discordgo.ApplicationCommandOptionInteger,
	Name:        "delta",
	Description: "How many episodes, defaults to 1",
}

var quietOption = &discordgo.ApplicationCommandOption{
	Type:        discordgo.ApplicationCommandOptionBoolean,
	Name:        "quiet",
	Description: "Don't post a link to the episode",
}

// AnimeCommand is /anime. Each subcommand is one of the old !anime ones, and
// its options are passed to it in order as arguments. Boolean options become
// --flags, so their names have to match the flags in the argument structs.
var AnimeCommand = &discordgo.ApplicationCommand{
	Name:        "anime",
	Description: "Keep track of the anime we're watching",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "incr",
			Description: "Watched more episodes of a show",
			Options:     []*discordgo.ApplicationCommandOption{animeOption, deltaOption, quietOption},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "decr",
			Description: "Take episodes back off a show",
			Options:     []*discordgo.ApplicationCommandOption{animeOption, deltaOption, quietOption},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set",
			Description: "Set the episode a show is on",
			Options: []*discordgo.ApplicationCommandOption{
				animeOption,
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "episode",
					Description: "Last episode watched",
					Required:    true,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "List the shows we're watching",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "name",
					Description: "Sort by name instead of by when it was last watched",
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "all",
					Description: "Also show finished, dropped, and stale shows",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "del",
			Description: "Mark a show as done",
			Options:     []*discordgo.ApplicationCommandOption{animeOption},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "inspect",
			Description: "Dump everything stored for a show",
			Options:     []*discordgo.ApplicationCommandOption{animeOption},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "src",
			Description: "Get the current episode link, or change where links come from",
			Options: []*discordgo.ApplicationCommandOption{
				animeOption,
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "source",
					Description: "cr:<series id>?season=<n> or nyaa:<search terms>",
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "remove",
					Description: "Clear the source",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "sub",
			Description: "Set the sub group for a show",
			Options: []*discordgo.ApplicationCommandOption{
				animeOption,
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "subber",
					Description: "Name of the sub group",
					Required:    true,
				},
			},
		},
	},
}

func Anime(i *discordgo.InteractionCreate) error {
	sub := i.ApplicationCommandData().Options[0]

	args := []string{}
	for _, opt := range sub.Options {
		if opt.Type != discordgo.ApplicationCommandOptionBoolean {
			args = append(args, fmt.Sprint(opt.Value))
		} else if opt.BoolValue() {
			args = append(args, "--"+opt.Name)
		}
	}

	resp, err := anime.Handle(sub.Name, args)
	if err != nil {
		return err
	}

	// Later followups show up as replies quoting the first one, so only the
	// first chunk is a followup.
	for n, msg := range chat.SplitMessage(resp) {
		if n == 0 {
			chat.SendFollowup(i.Interaction, msg)
		} else {
			chat.SendMessageToChannel(i.ChannelID, msg)
		}
	}

	return nil
}
