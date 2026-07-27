package chat

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const MAX_MESSAGE_LENGTH = 1900

// Lines longer than this are hard-split, leaving enough slack under
// MAX_MESSAGE_LENGTH for a reopened code fence and trailing newline.
const maxSegmentLength = MAX_MESSAGE_LENGTH - 100

var codeBlockRegex = regexp.MustCompile("```([a-zA-Z0-9_]*)")

// SplitMessage splits a message into chunks that each fit in a single
// discord message. Chunks break at newlines when possible; a single line
// longer than the limit is split at a nearby space, or mid-word at a rune
// boundary. A code block spanning a chunk boundary is closed at the end of
// the chunk and reopened with the same language in the next one.
func SplitMessage(msg string) []string {
	result := []string{}
	current := ""
	inFence := false
	fenceLang := ""

	capacity := func() int {
		if inFence {
			// Leave room for the closing "\n```" appended on flush.
			return MAX_MESSAGE_LENGTH - 4
		}

		return MAX_MESSAGE_LENGTH
	}

	flush := func() {
		chunk := strings.TrimRight(current, "\n")
		if inFence {
			chunk += "\n```"
		}

		if strings.TrimSpace(chunk) != "" {
			result = append(result, chunk)
		}

		current = ""
		if inFence {
			current = "```" + fenceLang + "\n"
		}
	}

	for _, line := range strings.Split(msg, "\n") {
		segments := []string{line}
		if len(line) > maxSegmentLength {
			segments = splitLongLine(line, maxSegmentLength)
		}

		for _, segment := range segments {
			// The +1 accounts for the newline appended after the segment.
			if len(current)+len(segment)+1 > capacity() {
				flush()
			}

			current += segment + "\n"
		}

		if match := codeBlockRegex.FindStringSubmatch(line); match != nil {
			if inFence {
				inFence = false
			} else {
				inFence = true
				fenceLang = match[1]
			}
		}
	}

	flush()
	return result
}

// splitLongLine splits a single overlong line into pieces of at most max
// bytes, cutting at a space near the limit when one exists, and never in
// the middle of a UTF-8 rune.
func splitLongLine(line string, max int) []string {
	pieces := []string{}
	for len(line) > max {
		cut := max
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut--
		}

		if space := strings.LastIndexByte(line[:cut], ' '); space > cut-200 && space > 0 {
			pieces = append(pieces, line[:space])
			line = line[space+1:]
			continue
		}

		pieces = append(pieces, line[:cut])
		line = line[cut:]
	}

	if line != "" {
		pieces = append(pieces, line)
	}

	return pieces
}
