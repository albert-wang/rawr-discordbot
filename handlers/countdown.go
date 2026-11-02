package handlers

import (
	"fmt"
	"time"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
)

func Countdown(i *discordgo.InteractionCreate) error {
	start := int64(3)

	opt := i.ApplicationCommandData().GetOption("from")
	if opt != nil {
		start = opt.IntValue()
	}

	if start > 30 {
		start = 30
	}

	for n := int64(0); n < start; n++ {
		if n == 0 {
			chat.SendFollowup(i.Interaction, fmt.Sprintf("%d", start-n))
		} else {
			chat.SendMessageToChannel(i.ChannelID, fmt.Sprintf("%d", start-n))
		}
		time.Sleep(time.Second)
	}

	chat.SendMessageToChannel(i.ChannelID, "g")
	return nil
}
