package chat

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// checkInvariants asserts properties that must hold for any input:
// at least one chunk for non-blank input, every chunk within the discord
// limit, no blank chunks, and no chunk cut mid-rune.
func checkInvariants(t *testing.T, msg string, chunks []string) {
	t.Helper()

	if strings.TrimSpace(msg) != "" && len(chunks) == 0 {
		t.Fatalf("non-empty message produced no chunks: %d bytes", len(msg))
	}

	for i, c := range chunks {
		if len(c) > MAX_MESSAGE_LENGTH {
			t.Errorf("chunk %d is %d bytes, over the %d limit", i, len(c), MAX_MESSAGE_LENGTH)
		}

		if strings.TrimSpace(c) == "" {
			t.Errorf("chunk %d is blank", i)
		}

		if !utf8.ValidString(c) {
			t.Errorf("chunk %d is not valid UTF-8", i)
		}
	}
}

// normalize collapses all whitespace so content can be compared across
// split points, where a space or newline may have been exchanged.
func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TestSplitShortMessages(t *testing.T) {
	for _, msg := range []string{
		"hi",
		strings.Repeat("a", 150),
		strings.Repeat("a", 400),
		strings.Repeat("a", 1500),
		strings.Repeat(strings.Repeat("a", 200)+"\n", 3),
	} {
		chunks := SplitMessage(msg)
		checkInvariants(t, msg, chunks)

		if len(chunks) != 1 {
			t.Fatalf("%d byte message should be a single chunk, got %d", len(msg), len(chunks))
		}

		if normalize(chunks[0]) != normalize(msg) {
			t.Errorf("content changed: %q -> %q", msg, chunks[0])
		}
	}
}

func TestSplitEmpty(t *testing.T) {
	if chunks := SplitMessage(""); len(chunks) != 0 {
		t.Errorf("empty message produced %d chunks", len(chunks))
	}
}

func TestSplitLongSingleLine(t *testing.T) {
	msg := strings.Repeat("a", 4000)
	chunks := SplitMessage(msg)
	checkInvariants(t, msg, chunks)

	if len(chunks) < 3 {
		t.Fatalf("4000 byte line should split into at least 3 chunks, got %d", len(chunks))
	}

	total := 0
	for _, c := range chunks {
		total += len(strings.TrimSpace(c))
	}

	if total != len(msg) {
		t.Errorf("content lost: %d bytes in, %d bytes out", len(msg), total)
	}
}

func TestSplitLongProseAtSpaces(t *testing.T) {
	word := "hello "
	msg := strings.TrimSpace(strings.Repeat(word, 700))
	chunks := SplitMessage(msg)
	checkInvariants(t, msg, chunks)

	if normalize(strings.Join(chunks, " ")) != normalize(msg) {
		t.Error("words were lost or altered when splitting prose")
	}

	for i, c := range chunks {
		for _, w := range strings.Fields(c) {
			if w != "hello" {
				t.Fatalf("chunk %d split mid-word: %q", i, w)
			}
		}
	}
}

func TestSplitCodeBlockSpanningChunks(t *testing.T) {
	msg := "some intro\n```go\n" + strings.Repeat("fmt.Println(\"a line of code\")\n", 150) + "```\ndone"
	chunks := SplitMessage(msg)
	checkInvariants(t, msg, chunks)

	if len(chunks) < 2 {
		t.Fatalf("expected the code block to span chunks, got %d", len(chunks))
	}

	for i, c := range chunks {
		if strings.Count(c, "```")%2 != 0 {
			t.Errorf("chunk %d has unbalanced code fences:\n%s", i, c)
		}
	}

	for _, c := range chunks[1:] {
		if !strings.HasPrefix(c, "```go\n") {
			t.Errorf("continuation chunk should reopen the fence with its language, got %q", c[:20])
		}
	}
}

func TestSplitMultibyteRunes(t *testing.T) {
	msg := strings.Repeat("あ", 1500)
	chunks := SplitMessage(msg)
	checkInvariants(t, msg, chunks)

	if strings.Join(chunks, "") != msg {
		t.Error("multibyte content was lost or corrupted")
	}
}

func TestSplitSweep(t *testing.T) {
	// Compare with all whitespace stripped: a split may exchange a space
	// for a newline or fall mid-word when the input has no spaces at all,
	// but no other content may be lost.
	stripped := func(s string) string {
		return strings.Join(strings.Fields(s), "")
	}

	for size := 1; size < 6000; size += 137 {
		plain := strings.Repeat("a", size)
		spaced := strings.TrimSpace(strings.Repeat("word123 ", size/8+1))[:size]
		lined := strings.ReplaceAll(spaced, " ", "\n")

		for _, msg := range []string{plain, spaced, lined} {
			chunks := SplitMessage(msg)
			checkInvariants(t, msg, chunks)

			if stripped(strings.Join(chunks, " ")) != stripped(msg) {
				t.Fatalf("content changed for %d byte message", len(msg))
			}
		}
	}
}
