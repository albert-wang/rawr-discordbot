package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	openai "github.com/sashabaranov/go-openai"
)

func convertMessageToContent(message *discordgo.Message, textPrefix string) ([]openai.ChatContent, bool) {
	requiresVision := false
	// Convert text into text content
	result := []openai.ChatContent{}

	if strings.TrimSpace(message.Content) != "" {
		content := ResolveMentionsToNicks(message.Content, message.GuildID, message.Mentions)

		result = append(result, openai.ChatContent{
			Type: "text",
			Text: fmt.Sprintf(textPrefix, content),
		})
	}

	for _, e := range message.Embeds {
		requiresVision = true

		result = append(result, openai.ChatContent{
			Type: "text",
			Text: fmt.Sprintf("%s %s", e.Title, e.Description),
		})

		if e.Thumbnail != nil {
			b, err := DownloadAttachment(e.Thumbnail.URL)
			out, err := ConvertImage(b, ".jpg",
				"-resize",
				"512x512>",
			)

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
			newBytes, err := io.ReadAll(output)
			if err != nil {
				log.Print(err)
				continue
			}

			bs := base64.StdEncoding.EncodeToString(newBytes)
			result = append(result,
				openai.ChatContent{
					Type:     "image_url",
					ImageURL: fmt.Sprintf("data:image/jpeg;base64,%s", bs),
				})
		}
	}

	for _, e := range message.Attachments {
		requiresVision = true

		b, err := DownloadAttachment(e.URL)
		out, err := ConvertImage(b, ".jpg",
			"-resize",
			"512x512>",
		)

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
		newBytes, err := io.ReadAll(output)
		if err != nil {
			log.Print(err)
			continue
		}

		bs := base64.StdEncoding.EncodeToString(newBytes)
		result = append(result,
			openai.ChatContent{
				Type:     "image_url",
				ImageURL: fmt.Sprintf("data:image/jpeg;base64,%s", bs),
			})
	}

	return result, requiresVision
}

func textContent(msg string) []openai.ChatContent {
	return []openai.ChatContent{{
		Type: "text",
		Text: msg,
	}}
}
