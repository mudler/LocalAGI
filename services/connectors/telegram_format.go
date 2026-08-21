package connectors

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	telegramLinkPattern    = regexp.MustCompile(`(?m)\[([^\]]+)\]\(([^\n]+)\)`)
	telegramBoldPattern    = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	telegramItalicPattern  = regexp.MustCompile(`\*([^*\n]+)\*`)
	telegramHeadingPattern = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(.+)$`)
)

func telegramFormatResponse(response string, urls []string, limit int) []string {
	return telegramSplitMarkdown(formatResponseWithURLs(response, urls), limit)
}

// formatResponseWithURLs preserves ordinary Markdown for Telegram's rich API.
func formatResponseWithURLs(response string, urls []string) string {
	if len(urls) == 0 {
		return response
	}

	var result strings.Builder
	result.WriteString(response)
	result.WriteString("\n\nReferences:\n")
	for i, url := range urls {
		fmt.Fprintf(&result, "🔗 %d. %s\n", i+1, url)
	}
	return result.String()
}

func telegramMarkdownV2(markdown string) string {
	parts := strings.Split(markdown, "```")
	for i := range parts {
		if i%2 == 1 {
			parts[i] = escapeTelegramCode(parts[i])
			continue
		}
		parts[i] = telegramMarkdownV2Text(parts[i])
	}
	return strings.Join(parts, "```")
}

func telegramMarkdownV2Text(text string) string {
	tokens := []string{}
	protect := func(value string) string {
		tokens = append(tokens, value)
		return fmt.Sprintf("\x00%d\x00", len(tokens)-1)
	}
	inlineCodePattern := regexp.MustCompile("`([^`\\n]+)`")
	text = inlineCodePattern.ReplaceAllStringFunc(text, func(match string) string {
		return protect("`" + escapeTelegramCode(strings.TrimSuffix(strings.TrimPrefix(match, "`"), "`")) + "`")
	})

	text = telegramLinkPattern.ReplaceAllStringFunc(text, func(match string) string {
		groups := telegramLinkPattern.FindStringSubmatch(match)
		label := escapeTelegramMarkdownV2(groups[1])
		url := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(groups[2])
		return protect("[" + label + "](" + url + ")")
	})
	text = telegramHeadingPattern.ReplaceAllStringFunc(text, func(match string) string {
		groups := telegramHeadingPattern.FindStringSubmatch(match)
		return protect("*" + escapeTelegramMarkdownV2(groups[1]) + "*")
	})
	text = telegramBoldPattern.ReplaceAllStringFunc(text, func(match string) string {
		groups := telegramBoldPattern.FindStringSubmatch(match)
		return protect("*" + escapeTelegramMarkdownV2(groups[1]) + "*")
	})
	text = telegramItalicPattern.ReplaceAllStringFunc(text, func(match string) string {
		groups := telegramItalicPattern.FindStringSubmatch(match)
		return protect("_" + escapeTelegramMarkdownV2(groups[1]) + "_")
	})
	text = escapeTelegramMarkdownV2(text)
	for i, token := range tokens {
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00%d\x00", i), token)
	}
	return text
}

func escapeTelegramMarkdownV2(text string) string {
	var result strings.Builder
	for _, r := range text {
		if strings.ContainsRune(`_*[]()~`+"`"+`>#+-=|{}.!\\`, r) {
			result.WriteByte('\\')
		}
		result.WriteRune(r)
	}
	return result.String()
}

func escapeTelegramCode(text string) string {
	return strings.NewReplacer(`\`, `\\`, "`", "\\`").Replace(text)
}

func telegramPlainText(markdown string) string {
	text := telegramLinkPattern.ReplaceAllString(markdown, "$1 ($2)")
	text = telegramHeadingPattern.ReplaceAllString(text, "$1")
	text = telegramBoldPattern.ReplaceAllString(text, "$1")
	text = telegramItalicPattern.ReplaceAllString(text, "$1")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.ReplaceAll(text, "`", "")
	return text
}

func telegramSplitMarkdown(text string, limit int) []string {
	if text == "" || limit <= 0 {
		return []string{}
	}
	if utf8.RuneCountInString(text) <= limit {
		return []string{text}
	}

	chunks := []string{}
	remaining := text
	openFence := ""
	for remaining != "" {
		prefix := ""
		if openFence != "" {
			prefix = "```" + openFence + "\n"
		}
		if utf8.RuneCountInString(prefix) >= limit {
			piece, rest := splitTelegramText(remaining, limit)
			chunks = append(chunks, piece)
			remaining = rest
			openFence = ""
			continue
		}
		capacity := limit - utf8.RuneCountInString(prefix)
		piece, rest := splitTelegramText(remaining, capacity)
		fenceAfter := telegramFenceState(openFence, piece)
		if fenceAfter != "" && rest != "" {
			close := "```\n"
			if !strings.HasSuffix(piece, "\n") {
				close = "\n" + close
			}
			closeLen := utf8.RuneCountInString(close)
			if capacity <= closeLen {
				piece, rest = splitTelegramText(remaining, limit)
				chunks = append(chunks, piece)
				remaining = rest
				openFence = ""
				continue
			}
			piece, rest = splitTelegramText(remaining, capacity-closeLen)
			fenceAfter = telegramFenceState(openFence, piece)
			if fenceAfter != "" {
				piece += close
			}
		}
		chunks = append(chunks, prefix+piece)
		remaining = rest
		openFence = fenceAfter
	}
	return chunks
}

func splitTelegramText(text string, limit int) (string, string) {
	if limit <= 0 {
		return "", text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text, ""
	}
	cut := limit
	for i := limit; i > 0; i-- {
		if runes[i-1] == '\n' {
			cut = i
			break
		}
	}
	return string(runes[:cut]), string(runes[cut:])
}

func telegramFenceState(current, text string) string {
	state := current
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "```") {
			continue
		}
		if state == "" {
			state = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			continue
		}
		state = ""
	}
	return state
}
