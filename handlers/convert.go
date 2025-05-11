package handlers

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
)

var attachmentMapping map[string][]*discordgo.MessageAttachment = map[string][]*discordgo.MessageAttachment{}

func RegisterAttachmentFromMessage(m *discordgo.MessageCreate) {
	key := fmt.Sprintf("%s:%s", m.GuildID, m.ChannelID)
	attachmentMapping[key] = m.Attachments
}

func RotateLastImages(m *discordgo.MessageCreate, args []string) error {
	key := fmt.Sprintf("%s:%s", m.GuildID, m.ChannelID)
	attachments, ok := attachmentMapping[key]
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

	chat.ForeachImageAttachment(attachments, func(attach *discordgo.MessageAttachment, img []byte) error {
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

	chat.SendImagesToChannel(m.ChannelID, files)
	return nil
}
