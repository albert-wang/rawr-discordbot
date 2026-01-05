package chat

import (
	"math"
	"regexp"
	"strings"
)

const MAX_MESSAGE_LENGTH = 1800

func SplitMessage(msg string) []string {

	lines := strings.Split(msg, "\n")

	targetLength := len(msg)
	if len(msg) > MAX_MESSAGE_LENGTH {
		messagesCount := math.Ceil(float64(len(msg)) / MAX_MESSAGE_LENGTH)
		targetLength = int(MAX_MESSAGE_LENGTH / messagesCount)
	}

	codeBlockRegex := regexp.MustCompile("```([a-zA-Z0-9_]*)")

	currentMessage := ""
	currentBlockElement := ""

	result := []string{}
	for _, line := range lines {
		if len(line)+len(currentMessage) > targetLength {
			if currentBlockElement != "" {
				currentMessage += "\n```"
			}

			result = append(result, currentMessage)
			currentMessage = ""

			if currentBlockElement != "" {
				currentMessage = "```" + currentBlockElement + "\n"
			}
		}

		codeBlockMatch := codeBlockRegex.FindStringSubmatch(line)
		if len(codeBlockMatch) > 0 {
			if currentBlockElement != "" {
				// Exit a code block
				currentBlockElement = ""
			} else {
				// Enter a code block
				if len(codeBlockMatch) != 0 {
					currentBlockElement = codeBlockMatch[1]
				} else {
					currentBlockElement = "code"
				}
			}
		}

		currentMessage += line + "\n"
	}

	if currentBlockElement != "" {
		currentMessage += "```"
	}

	result = append(result, currentMessage)
	return result
}
