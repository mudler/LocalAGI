package xstrings_test

import (
	xtrings "github.com/mudler/LocalAgent/pkg/xstrings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SplitParagraph", func() {
	It("should return the text as a single chunk if it's shorter than maxLen", func() {
		text := "Short text"
		maxLen := 20
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"Short text"}))
	})

	It("should split the text into chunks of maxLen without truncating words", func() {
		text := "This is a longer text that needs to be split into chunks."
		maxLen := 10
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"This is a", "longer", "text that", "needs to", "be split", "into", "chunks."}))
	})

	It("should handle texts with multiple spaces and newlines correctly", func() {
		text := "This  is\na\ntext  with\n\nmultiple spaces   and\nnewlines."
		maxLen := 10
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"This  is\na\n", "text  with\n", "multiple", "spaces  ", "and\n", "newlines."}))
	})

	It("should handle a text with a single word longer than maxLen", func() {
		text := "supercalifragilisticexpialidocious"
		maxLen := 10
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"supercalifragilisticexpialidocious"}))
	})

	It("should handle a text with empty lines", func() {
		text := "line1\n\nline2"
		maxLen := 10
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"line1\n\n", "line2"}))
	})

	It("should handle a text with leading and trailing spaces", func() {
		text := "   leading spaces and trailing spaces   "
		maxLen := 15
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"   leading", "spaces and", "trailing spaces"}))
	})

	It("should handle a text with only spaces", func() {
		text := "   "
		maxLen := 10
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"   "}))
	})

	It("should handle empty string", func() {
		text := ""
		maxLen := 10
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{""}))
	})

	It("should handle a text with only newlines", func() {
		text := "\n\n\n"
		maxLen := 10
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"\n\n\n"}))
	})

	It("should handle a text with special characters", func() {
		text := "This is a text with special characters !@#$%^&*()"
		maxLen := 20
		result := xtrings.SplitParagraph(text, maxLen)
		Expect(result).To(Equal([]string{"This is a text with", "special characters", "!@#$%^&*()"}))
	})
})
