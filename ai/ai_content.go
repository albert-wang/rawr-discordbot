package ai

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/bwmarrin/discordgo"
	openai "github.com/sashabaranov/go-openai"
)

type ConversionOptions struct {
	Format       string
	IncludeMedia bool
}

func MessageContent(message *discordgo.Message, opts ConversionOptions) []openai.ChatMessagePart {
	// Convert text into text content
	result := []openai.ChatMessagePart{}

	if strings.TrimSpace(message.Content) != "" {
		content := chat.ResolveMentionsToNicks(message.Content, message.GuildID, message.Mentions)

		result = append(result, openai.ChatMessagePart{
			Type: "text",
			Text: fmt.Sprintf(opts.Format, content),
		})
	}

	if opts.IncludeMedia {
		if len(message.Embeds) > 0 {
			embeds := EmbedsContent(message)
			result = append(result, embeds...)
		}

		if len(message.Attachments) > 0 {
			attachments := AttachmentsContent(message)
			result = append(result, attachments...)
		}
	}

	return result
}

func AttachmentsContent(message *discordgo.Message) []openai.ChatMessagePart {
	result := []openai.ChatMessagePart{}

	chat.ForeachImageAttachment(message.Attachments, func(attachment *discordgo.MessageAttachment, img []byte) error {
		out, err := chat.ConvertImage(img, ".jpg",
			"-resize",
			"1536x1536>",
		)

		if err != nil {
			log.Print(err)
			return err
		}

		defer os.Remove(out)

		bytes, err := os.ReadFile(out)
		if err != nil {
			log.Print(err)
			return err
		}

		bs := base64.StdEncoding.EncodeToString(bytes)
		result = append(result,
			openai.ChatMessagePart{
				Type: "image_url",
				ImageURL: &openai.ChatMessageImageURL{
					URL: fmt.Sprintf("data:image/jpeg;base64,%s", bs),
				},
			})

		return nil
	})

	return result
}

func EmbedsContent(message *discordgo.Message) []openai.ChatMessagePart {
	result := []openai.ChatMessagePart{}

	for _, e := range message.Embeds {
		result = append(result, openai.ChatMessagePart{
			Type: "text",
			Text: fmt.Sprintf("%s %s", e.Title, e.Description),
		})

		if e.Thumbnail != nil {
			b, err := chat.GetURLBytes(e.Thumbnail.URL)
			out, err := chat.ConvertImage(b, ".jpg",
				"-resize",
				"1536x1536>",
			)

			if err != nil {
				log.Print(err)
				continue
			}

			defer os.Remove(out)

			bytes, err := os.ReadFile(out)
			if err != nil {
				log.Print(err)
				continue
			}

			bs := base64.StdEncoding.EncodeToString(bytes)
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

func TextContent(msg string) []openai.ChatMessagePart {
	return []openai.ChatMessagePart{{
		Type: "text",
		Text: msg,
	}}
}

func TemplateContent(tplText string, args any) []openai.ChatMessagePart {
	buff := bytes.NewBuffer(nil)
	tpl, err := template.New("anime").Funcs(template.FuncMap{
		"pad": func(amount int, spacer string, val string) string {
			if len(val) < amount {
				return strings.Repeat(spacer, amount-len(val)) + val
			}

			return val
		},
	}).Parse(tplText)
	if err != nil {
		log.Print(err)
		return []openai.ChatMessagePart{}
	}

	err = tpl.Execute(buff, args)
	if err != nil {
		log.Print(err)
		return []openai.ChatMessagePart{}
	}

	return TextContent(buff.String())
}
