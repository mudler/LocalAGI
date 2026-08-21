package connectors

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTelegramFormatRawMarkdownAndURLs(t *testing.T) {
	t.Parallel()

	chunks := telegramFormatResponse("# Title\n\n**bold**", []string{"https://example.com/a_(b)"}, 200)
	want := "# Title\n\n**bold**\n\nReferences:\n🔗 1. https://example.com/a_(b)\n"
	if len(chunks) != 1 || chunks[0] != want {
		t.Fatalf("telegramFormatResponse() = %#v, want [%q]", chunks, want)
	}
}

func TestTelegramFormatMarkdownV2(t *testing.T) {
	t.Parallel()

	input := "# Heading\n\n**bold** and *italic* with [link](https://example.com/a_(b)).\n\n```go\nfmt.Println(`ok`)\n```"
	got := telegramMarkdownV2(input)
	want := "*Heading*\n\n*bold* and _italic_ with [link](https://example.com/a_\\(b\\))\\.\n\n```go\nfmt.Println(\\`ok\\`)\n```"
	if got != want {
		t.Fatalf("telegramMarkdownV2() = %q, want %q", got, want)
	}
}

func TestTelegramFormatPlainText(t *testing.T) {
	t.Parallel()

	got := telegramPlainText("# Heading\n\n**bold** and [link](https://example.com)")
	want := "Heading\n\nbold and link (https://example.com)"
	if got != want {
		t.Fatalf("telegramPlainText() = %q, want %q", got, want)
	}
}

func TestTelegramSplitUTF8WithoutDataLoss(t *testing.T) {
	t.Parallel()

	input := strings.Repeat("🙂", 11)
	chunks := telegramSplitMarkdown(input, 4)
	if strings.Join(chunks, "") != input {
		t.Fatalf("joined chunks differ: %#v", chunks)
	}
	for _, chunk := range chunks {
		if !utf8.ValidString(chunk) || utf8.RuneCountInString(chunk) > 4 {
			t.Fatalf("invalid chunk %q", chunk)
		}
	}
}

func TestTelegramSplitBalancesFencedCodeBlocks(t *testing.T) {
	t.Parallel()

	input := "before\n```go\n" + strings.Repeat("line\n", 8) + "```\nafter"
	chunks := telegramSplitMarkdown(input, 28)
	if len(chunks) < 2 {
		t.Fatalf("got %d chunks, want multiple", len(chunks))
	}
	for _, chunk := range chunks {
		if strings.Count(chunk, "```")%2 != 0 {
			t.Fatalf("unbalanced code fence in %q", chunk)
		}
	}
	joined := strings.Join(chunks, "")
	joined = strings.ReplaceAll(joined, "```\n```go\n", "")
	if joined != input {
		t.Fatalf("split lost content:\n%q\nwant:\n%q", joined, input)
	}
}

func TestTelegramSplitTinyLimitMakesProgress(t *testing.T) {
	t.Parallel()

	input := "```go\n🙂🙂\n```"
	chunks := telegramSplitMarkdown(input, 5)
	if strings.Join(chunks, "") != input {
		t.Fatalf("joined chunks differ: %#v", chunks)
	}
}
