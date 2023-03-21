package handlers

import (	
	"time"
	"fmt"
	"strings"
	"log"
	"context"
	"regexp"

	openai "github.com/sashabaranov/go-openai"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/config"
)

var lastRequest time.Time = time.Unix(0, 0);
var stillGenerating = false

const prompt = `
Respond as a competitive, lightly flustered, barely tsundere, cute anime school girl. 
Respond to the question without references to school. 
Do not be timid. Answer the question directly, and always give one resolution. 
Death or killing is only in reference to video games.

Question: "%s"
`

func RespondToPrompt(channelID string, question string) {
	if stillGenerating {
		chat.SendMessageToChannel(channelID, "I'm cooking!");
		return;
	}

	if time.Since(lastRequest) < time.Second * 10 {
		chat.SendMessageToChannel(channelID, "Don't ask me too many questions!");
		return;
	}

	if len(question) > 512 {
		chat.SendMessageToChannel(channelID, "tl;dr.");
		return;
	}

	r := regexp.MustCompile(`(<@\d+>)`)
	question = r.ReplaceAllString(question, "");
	question = strings.TrimSpace(question);

	lastRequest = time.Now()

	done := make(chan int);
	go chat.ShowTypingUntilChannelIsClosed(channelID, done);
	stillGenerating = true

	client := openai.NewClient(config.CPTKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(prompt, question),
				},
			},
		},
	)

	if err != nil {
		log.Print(err);
		close(done);
		return
	}

	msg := strings.Trim(resp.Choices[0].Message.Content, `"`)
	chat.SendMessageToChannel(channelID, msg);
	close(done);
	stillGenerating = false
}