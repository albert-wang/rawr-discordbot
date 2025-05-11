package chat

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func AttachmentIsImage(attachment *discordgo.MessageAttachment) bool {
	return strings.HasPrefix(attachment.ContentType, "image/")
}

func AttachmentExtension(attachment *discordgo.MessageAttachment) string {
	name := attachment.Filename
	return path.Ext(name)
}

func ForeachImageAttachment(attachments []*discordgo.MessageAttachment, cb func(attachment *discordgo.MessageAttachment, img []byte) error) {
	for _, attachment := range attachments {
		if !AttachmentIsImage(attachment) {
			continue
		}

		b, err := GetURLBytes(attachment.URL)
		if err != nil {
			log.Print(err)
			continue
		}

		err = cb(attachment, b)
		if err != nil {
			log.Print(err)
			continue
		}
	}
}

func ConvertAttachmentsToDataURL(attachments []*discordgo.MessageAttachment, maxWidth int, maxHeight int) []string {
	result := []string{}

	ForeachImageAttachment(attachments, func(attachment *discordgo.MessageAttachment, b []byte) error {
		out, err := ConvertImage(b, AttachmentExtension(attachment),
			"-resize",
			fmt.Sprintf("%dx%d>", maxWidth, maxHeight),
		)

		if err != nil {
			log.Print(err)
			return err
		}

		defer os.Remove(out)
		output, err := os.Open(out)
		if err != nil {
			log.Print(err)
			return err
		}

		defer output.Close()
		newBytes, err := io.ReadAll(output)
		if err != nil {
			log.Print(err)
			return err
		}

		bs := base64.StdEncoding.EncodeToString(newBytes)
		log.Print(attachment.ContentType)
		result = append(result, fmt.Sprintf("data:%s;base64,%s", attachment.ContentType, bs))

		return nil
	})

	return result
}
