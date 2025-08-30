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

func RotateLastImages(m *discordgo.MessageCreate, args []string) error {
	key := fmt.Sprintf("%s:%s", m.GuildID, m.ChannelID)
	m, ok := messageMapping[key]
	if !ok {
		chat.SendMessageToChannel(m.ChannelID, "I don't see any images to rotate")
		return nil
	}

	complete := chat.ShowTyping(m.ChannelID)
	defer complete()

	amount := "90"
	if len(args) == 1 {
		amount = args[0]
	}

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

	chat.SendImagesToChannel(m.ChannelID, files)
	return nil
}
