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

func convertMessageToContent(message *discordgo.Message, textPrefix string) ([]openai.ChatMessagePart, bool) {
	requiresVision := false
	// Convert text into text content
	result := []openai.ChatMessagePart{}

	if strings.TrimSpace(message.Content) != "" {
		content := ResolveMentionsToNicks(message.Content, message.GuildID, message.Mentions)

		result = append(result, openai.ChatMessagePart{
			Type: "text",
			Text: fmt.Sprintf(textPrefix, content),
		})
	}

	if len(message.Embeds) > 0 {
		requiresVision = true
		embeds := convertEmbedsToContent(message)
		result = append(result, embeds...)
	}

	if len(message.Attachments) > 0 {
		requiresVision = true
		attachments := convertAttachmentsToContent(message)
		result = append(result, attachments...)
	}

	return result, requiresVision
}

func convertAttachmentsToContent(message *discordgo.Message) []openai.ChatMessagePart {
	result := []openai.ChatMessagePart{}

	for _, e := range message.Attachments {
		b, err := DownloadAttachment(e.URL)
		out, err := ConvertImage(b, ".jpg",
			"-resize",
			"1536x1536>",
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
			openai.ChatMessagePart{
				Type: "image_url",
				ImageURL: &openai.ChatMessageImageURL{
					URL: fmt.Sprintf("data:image/jpeg;base64,%s", bs),
				},
			})
	}

	return result
}

func convertEmbedsToContent(message *discordgo.Message) []openai.ChatMessagePart {
	result := []openai.ChatMessagePart{}

	for _, e := range message.Embeds {
		result = append(result, openai.ChatMessagePart{
			Type: "text",
			Text: fmt.Sprintf("%s %s", e.Title, e.Description),
		})

		if e.Thumbnail != nil {
			b, err := DownloadAttachment(e.Thumbnail.URL)
			out, err := ConvertImage(b, ".jpg",
				"-resize",
				"1536x1536>",
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
				openai.ChatMessagePart{
					Type: "image_url",
					ImageURL: &openai.ChatMessageImageURL{
						URL: fmt.Sprintf("data:image/jpeg;base64,%s", bs),
					},
				})
		}
	}

	return result
}

func textContent(msg string) []openai.ChatMessagePart {
	return []openai.ChatMessagePart{{
		Type: "text",
		Text: msg,
	}}
}
