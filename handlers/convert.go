package handlers

import (
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

	ch := make(chan int)
	go chat.ShowTypingUntilChannelIsClosed(m.ChannelID, ch)
	defer close(ch)

	amount := "90"
	if len(args) == 1 {
		amount = args[0]
	}

	files := []*discordgo.File{}

	for _, attach := range attachments {
		if !AttachmentIsImage(attach) {
			continue
		}

		bytes, err := DownloadAttachment(attach)
		if err != nil {
			log.Print(err)
			continue
		}

		out, err := ConvertImage(bytes, AttachmentExtension(attach), "-rotate", amount)
		if err != nil {
			log.Print(err)
			continue
		}

		defer os.Remove(out)

		output, err := os.Open(out)
		if err != nil {
			log.Print(err)
			continue
		}

		defer output.Close()
		files = append(files, &discordgo.File{
			Name:        attach.Filename,
			ContentType: attach.ContentType,
			Reader:      output,
		})
	}

	chat.SendImagesToChannel(m.ChannelID, files)
	return nil
}
