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
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

var RESIZE_OPTIONS = []string{"-resize", "2048x2048>"}

type ConversionOptions struct {
	// Adds in text descriptions of attachments, embeds, and stickers.
	IncludeMedia bool
}

func MessageContent(message *discordgo.Message, opts ConversionOptions) []responses.ResponseInputContentUnionParam {
	result := []responses.ResponseInputContentUnionParam{}

	content := ""
	if strings.TrimSpace(message.Content) != "" {
		content = chat.ResolveMentionsToNicks(message.Content, message.GuildID, message.Mentions)
	}

	imageCount := 0
	if opts.IncludeMedia {
		for _, a := range message.Attachments {
			if chat.AttachmentIsImage(a) {
				imageCount++
			}
		}

		for _, e := range message.Embeds {
			if e.Thumbnail != nil {
				imageCount++
			}
		}
	}

	if opts.IncludeMedia == false {
		imageCount = 0
	}

	header := fmt.Sprintf(`<msg message_id="%s" author_name="%s" author_id="%s" image_count="%d">`,
		message.ID,
		message.Author.Username,
		message.Author.ID,
		imageCount,
	)

	if message.ReferencedMessage != nil {
		header = fmt.Sprintf(`<msg message_id="%s" author_name="%s" author_id="%s" image_count="%d" reference="%s">`,
			message.ID,
			message.Author.Username,
			message.Author.ID,
			imageCount,
			message.ReferencedMessage.ID,
		)
	}

	result = append(result,
		responses.ResponseInputContentParamOfInputText(fmt.Sprintf("%s\n%s\n</msg>", header, strings.TrimSpace(content))))

	if opts.IncludeMedia {
		chat.ForeachImageAttachment(message.Attachments, func(attachment *discordgo.MessageAttachment, img []byte) error {
			result = append(result, responses.ResponseInputContentParamOfInputText(
				fmt.Sprintf(`<attachment filename="%s" height="%d" width="%d" content_type="%s" />`,
					attachment.Filename,
					attachment.Height,
					attachment.Width,
					attachment.ContentType)))

			return nil
		})

		for _, e := range message.Embeds {
			if strings.TrimSpace(e.Title) == "" && strings.TrimSpace(e.Description) == "" {
				continue
			}

			result = append(result, responses.ResponseInputContentParamOfInputText(
				fmt.Sprintf(`<embed title="%s">%s</embed>`, e.Title, e.Description)))
		}

		for _, sticker := range message.StickerItems {
			result = append(result, responses.ResponseInputContentParamOfInputText(
				fmt.Sprintf(`<sticker>%s</sticker>`, sticker.Name)))
		}
	}

	return result
}

func AttachmentsContent(message *discordgo.Message) []responses.ResponseInputContentUnionParam {
	result := []responses.ResponseInputContentUnionParam{}

	chat.ForeachImageAttachment(message.Attachments, func(attachment *discordgo.MessageAttachment, img []byte) error {
		out, err := chat.ConvertImage(img, ".jpg", RESIZE_OPTIONS...)
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
		image := responses.ResponseInputContentParamOfInputImage("original")
		image.OfInputImage.ImageURL = param.NewOpt(fmt.Sprintf("data:image/jpeg;base64,%s", bs))

		result = append(result, image)
		return nil
	})

	return result
}

func EmbedsContent(message *discordgo.Message) []responses.ResponseInputContentUnionParam {
	result := []responses.ResponseInputContentUnionParam{}

	for _, e := range message.Embeds {
		if e.Thumbnail != nil {
			b, err := chat.GetURLBytes(e.Thumbnail.URL)
			if err != nil {
				log.Print(err)
				continue
			}

			out, err := chat.ConvertImage(b, ".jpg", RESIZE_OPTIONS...)
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
			image := responses.ResponseInputContentParamOfInputImage("original")
			image.OfInputImage.ImageURL = param.NewOpt(fmt.Sprintf("data:image/jpeg;base64,%s", bs))

			result = append(result, image)
		}
	}

	return result
}

func TextContent(msg string) []responses.ResponseInputContentUnionParam {
	return []responses.ResponseInputContentUnionParam{
		responses.ResponseInputContentParamOfInputText(msg),
	}
}

func TemplateContent(tplText string, args any) []responses.ResponseInputContentUnionParam {
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
		return []responses.ResponseInputContentUnionParam{}
	}

	err = tpl.Execute(buff, args)
	if err != nil {
		log.Print(err)
		return []responses.ResponseInputContentUnionParam{}
	}

	return TextContent(buff.String())
}
