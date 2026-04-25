package chat

import (
	"math"
	"regexp"
	"strings"
)

const MAX_MESSAGE_LENGTH = 1900

func SplitMessage(msg string) []string {
	lines := strings.Split(msg, "\n")

	targetLength := len(msg)
	if len(msg) > MAX_MESSAGE_LENGTH {
		messagesCount := math.Ceil(float64(len(msg)) / MAX_MESSAGE_LENGTH)
		targetLength = len(msg) / int(messagesCount)
	}

	codeBlockRegex := regexp.MustCompile("```([a-zA-Z0-9_]*)")
	result := []string{}

	for attempts := 0; attempts < 5; attempts++ {
		currentMessage := ""
		currentBlockElement := ""
		linesInMessage := 0

		for _, line := range lines {
			if len(line)+len(currentMessage) > targetLength {
				if currentBlockElement != "" {
					currentMessage += "\n```"
				}

				result = append(result, currentMessage)
				currentMessage = ""
				linesInMessage = 0

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
			linesInMessage += 1
		}

		if currentBlockElement != "" {
			currentMessage += "```"
		}

		result = append(result, currentMessage)

		// This looks ugly. Try again with lower target length
		if linesInMessage < 5 {
			targetLength = (int)(float64(targetLength) * 0.9)
			if targetLength < 200 {
				return result
			}

			result = []string{}
		} else {
			return result
		}
	}

	return result
}
