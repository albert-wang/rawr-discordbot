package handlers

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
)

var messageMapping map[string]*discordgo.MessageCreate = map[string]*discordgo.MessageCreate{}

func RegisterAttachmentFromMessage(m *discordgo.MessageCreate) {
	key := fmt.Sprintf("%s:%s", m.GuildID, m.ChannelID)
	messageMapping[key] = m
}

func Rotate(i *discordgo.InteractionCreate) error {
	amount := "90"

	opt := i.ApplicationCommandData().GetOption("degrees")
	if opt != nil {
		amount = fmt.Sprintf("%d", opt.IntValue())
	}

	files := RotateLastImages(i.GuildID, i.ChannelID, amount)
	if len(files) == 0 {
		chat.SendFollowup(i.Interaction, "I don't see any images to rotate")
		return nil
	}

	chat.SendFollowup(i.Interaction, "", files...)
	return nil
}

// RotateLastImages rotates the images from the last message in the channel
// that had any, or returns nothing if there wasn't one.
func RotateLastImages(guildID string, channelID string, amount string) []*discordgo.File {
	key := fmt.Sprintf("%s:%s", guildID, channelID)
	m, ok := messageMapping[key]
	if !ok {
		return nil
	}

	complete := chat.ShowTyping(m.ChannelID)
	defer complete()

	files := []*discordgo.File{}

	chat.ForeachImageAttachment(m.Attachments, func(attach *discordgo.MessageAttachment, img []byte) error {
		out, err := chat.ConvertImage(img, chat.AttachmentExtension(attach), "-rotate", amount)
		if err != nil {
			log.Print(err)
			return err
		}

		defer os.Remove(out)

		convertedBytes, err := os.ReadFile(out)
		if err != nil {
			log.Print(err)
			return err
		}

		files = append(files, &discordgo.File{
			Name:        attach.Filename,
			ContentType: attach.ContentType,
			Reader:      bytes.NewBuffer(convertedBytes),
		})

		return nil
	})

	chat.ForeachImageEmbed(m.Embeds, func(embed *discordgo.MessageEmbed, format string, img []byte) error {
		out, err := chat.ConvertImage(img, fmt.Sprintf(".%s", format), "-rotate", amount)
		if err != nil {
			log.Print(err)
			return err
		}

		defer os.Remove(out)

		convertedBytes, err := os.ReadFile(out)
		if err != nil {
			log.Print(err)
			return err
		}

		files = append(files, &discordgo.File{
			Name:        fmt.Sprintf("image-%s.%s", embed.Timestamp, format),
			ContentType: fmt.Sprintf("image/%s", format),
			Reader:      bytes.NewBuffer(convertedBytes),
		})

		return nil
	})

	return files
}
