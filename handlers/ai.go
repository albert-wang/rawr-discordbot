package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	openai "github.com/sashabaranov/go-openai"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/config"
)

var lastRequest time.Time = time.Unix(0, 0)
var stillGenerating = false

func RespondToPrompt(m *discordgo.MessageCreate) {
	if stillGenerating {
		chat.SendMessageToChannel(m.ChannelID, "Let me cook!")
		return
	}

	if time.Since(lastRequest) < time.Second*10 {
		chat.SendMessageToChannel(m.ChannelID, "Don't ask me too many questions!")
		return
	}

	complete := chat.ShowTyping(m.ChannelID)
	defer complete()

	messages := GenerateMessagesWithContext(m.GuildID, m.ChannelID, 32)

	content, _ := convertMessageToContent(m.Message, "%s")
	messages = append(messages, openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleUser,
		MultiContent: content,
	})

	UnboundedRespondToContent(m.GuildID, m.ChannelID, messages, true)
}

func invokeFunction(guild string, channel string, name string, args string) ([]openai.ChatMessagePart, bool) {
	log.Print("Invoking function ", name, " with args ", args)

	if name == "get_previous_n_messages_from_user" {
		arg := GetPreviousNMessagesFromUserArgs{}
		err := json.Unmarshal([]byte(args), &arg)
		if err != nil {
			log.Print(err)
			return []openai.ChatMessagePart{}, false
		}

		return GetPreviousNMessagesFromUser(guild, channel, arg)
	}

	if name == "get_last_image" {
		arg := GetLastImageArgs{}
		err := json.Unmarshal([]byte(args), &arg)
		if err != nil {
			log.Print(err)
			return []openai.ChatMessagePart{}, false
		}

		return GetLastImage(guild, channel, arg)
	}

	return []openai.ChatMessagePart{}, false
}

func MakeOpenAPIRequest(guild string, channel string, model AIModel, supportsFunctions bool, client *openai.Client, messages *[]openai.ChatCompletionMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:     model.Name,
		MaxTokens: 780,
		Messages:  *messages,
	}

	if supportsFunctions {
		req.Functions = SupportedFunctions()
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		req,
	)

	if err != nil {
		log.Print(err)

		dbgreq, _ := json.MarshalIndent(req, "", "  ")
		log.Print(string(dbgreq))

		dbg, _ := json.MarshalIndent(resp, "", "  ")
		log.Print(string(dbg))

		return "", err
	}

	if len(resp.Choices) == 0 {
		return "Empty response :(", nil
	}

	choice := resp.Choices[0]

	if choice.Message.FunctionCall != nil {
		fn, _ := json.Marshal(choice.Message.FunctionCall)
		*messages = append(*messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleAssistant,
			MultiContent: []openai.ChatMessagePart{{
				Type: "text",
				Text: string(fn),
			}},
		})

		additionalContext, _ := invokeFunction(guild, channel, choice.Message.FunctionCall.Name, choice.Message.FunctionCall.Arguments)
		if len(additionalContext) > 0 {
			*messages = append(*messages, openai.ChatCompletionMessage{
				Role:         openai.ChatMessageRoleUser,
				Name:         choice.Message.FunctionCall.Name,
				MultiContent: additionalContext,
			})
		}

		visionModel := GetVisionModel()
		return MakeOpenAPIRequest(guild, channel, visionModel, false, client, messages)
	}

	dbgreq, _ := json.MarshalIndent(req, "", "  ")
	log.Print(string(dbgreq))

	dbg, _ := json.MarshalIndent(resp, "", "  ")
	log.Print(string(dbg))

	msg := strings.Trim(choice.Message.Content, `"`)
	msg = strings.TrimSpace(msg)
	msg = strings.Trim(msg, `"`)

	msg = strings.TrimPrefix(msg, "NVG-Tan >")

	msg = strings.TrimSpace(msg)
	msg = strings.Trim(msg, `"`)

	return msg, nil
}

func UnboundedRespondToContent(guildID string, channelID string, messages []openai.ChatCompletionMessage, needsVision bool) {
	lastRequest = time.Now()

	stillGenerating = true
	client := openai.NewClient(config.CPTKey)

	for _, model := range models {
		if needsVision {
			if !model.Vision {
				continue
			}
		}

		log.Printf("Using model: %s", model.Name)

		msg, err := MakeOpenAPIRequest(guildID, channelID, model, model.Function, client, &messages)
		if err != nil {
			log.Print(err)
			continue
		}

		chat.SendMessageToChannel(channelID, msg)
		stillGenerating = false
		break
	}
}
