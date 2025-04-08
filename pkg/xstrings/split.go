package xstrings

import (
	"strings"
)

// SplitTextByLength splits text into chunks of specified maxLength,
// preserving complete words and special characters like newlines.
// It returns a slice of strings, each with length <= maxLength.
func SplitParagraph(text string, maxLength int) []string {
	// Handle edge cases
	if maxLength <= 0 || len(text) == 0 {
		return []string{text}
	}

	var chunks []string
	remainingText := text

	for len(remainingText) > 0 {
		// If remaining text fits in a chunk, add it and we're done
		if len(remainingText) <= maxLength {
			chunks = append(chunks, remainingText)
			break
		}

		// Try to find a good split point near the max length
		splitIndex := maxLength

		// Look backward from the max length to find a space or newline
		for splitIndex > 0 && !isWhitespace(rune(remainingText[splitIndex])) {
			splitIndex--
		}

		// If we couldn't find a good split point (no whitespace),
		// look forward for the next whitespace
		if splitIndex == 0 {
			splitIndex = maxLength
			// If we can't find whitespace forward, we'll have to split a word
			for splitIndex < len(remainingText) && !isWhitespace(rune(remainingText[splitIndex])) {
				splitIndex++
			}

			// If we still couldn't find whitespace, take the whole string
			if splitIndex == len(remainingText) {
				chunks = append(chunks, remainingText)
				break
			}
		}

		// Add the chunk up to the split point
		chunk := remainingText[:splitIndex]

		// Preserve trailing newlines with the current chunk
		if splitIndex < len(remainingText) && remainingText[splitIndex] == '\n' {
			chunk += string(remainingText[splitIndex])
			splitIndex++
		}

		chunks = append(chunks, chunk)

		// Remove leading whitespace from the next chunk
		remainingText = remainingText[splitIndex:]
		remainingText = strings.TrimLeftFunc(remainingText, isWhitespace)
	}

	return chunks
}

// Helper function to determine if a character is whitespace
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
