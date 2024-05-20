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

	question := ResolveMentionsToNicks(m.Content, m.GuildID, m.Mentions)
	if len(question) > 1024 {
		chat.SendMessageToChannel(m.ChannelID, "tl;dr.")
		return
	}

	content, needsVision := convertMessageToContent(m.Message, GetPrompt())
	UnboundedRespondToContent(m.GuildID, m.ChannelID, content, needsVision)
}

func invokeFunction(guild string, channel string, name string, args string) ([]openai.ChatContent, bool) {
	log.Print("Invoking function ", name, " with args ", args)

	if name == "get_previous_n_messages_from_user" {
		arg := GetPreviousNMessagesFromUserArgs{}
		err := json.Unmarshal([]byte(args), &arg)
		if err != nil {
			log.Print(err)
			return []openai.ChatContent{}, false
		}

		return GetPreviousNMessagesFromUser(guild, channel, arg)
	}

	if name == "get_last_image" {
		arg := GetLastImageArgs{}
		err := json.Unmarshal([]byte(args), &arg)
		if err != nil {
			log.Print(err)
			return []openai.ChatContent{}, false
		}

		return GetLastImage(guild, channel, arg)
	}

	return []openai.ChatContent{}, false
}

func MakeOpenAPIRequest(guild string, channel string, model AIModel, supportsFunctions bool, client *openai.Client, messages *[]openai.ChatCompletionMessageInput) (string, error) {
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

	b, _ := json.MarshalIndent(choice, "", "  ")
	log.Print(string(b))

	if choice.Message.FunctionCall != nil {
		fn, _ := json.Marshal(choice.Message.FunctionCall)
		*messages = append(*messages, openai.ChatCompletionMessageInput{
			Role: openai.ChatMessageRoleAssistant,
			Content: []openai.ChatContent{{
				Type: "text",
				Text: string(fn),
			}},
		})

		additionalContext, _ := invokeFunction(guild, channel, choice.Message.FunctionCall.Name, choice.Message.FunctionCall.Arguments)
		if len(additionalContext) > 0 {
			*messages = append(*messages, openai.ChatCompletionMessageInput{
				Role:    openai.ChatMessageRoleUser,
				Name:    choice.Message.FunctionCall.Name,
				Content: additionalContext,
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

	return msg, nil
}

func UnboundedRespondToContent(guildID string, channelID string, content []openai.ChatContent, needsVision bool) {
	lastRequest = time.Now()

	complete := chat.ShowTyping(channelID)
	defer complete()

	stillGenerating = true

	client := openai.NewClient(config.CPTKey)

	for _, model := range models {
		if needsVision {
			if !model.Vision {
				continue
			}
		}

		actualContent := content
		log.Printf("Using model: %s", model.Name)

		message := []openai.ChatCompletionMessageInput{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: actualContent,
			},
		}

		msg, err := MakeOpenAPIRequest(guildID, channelID, model, model.Function, client, &message)
		if err != nil {
			log.Print(err)
			continue
		}

		chat.SendMessageToChannel(channelID, msg)
		stillGenerating = false
		break
	}
}
