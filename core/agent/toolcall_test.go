package agent

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sashabaranov/go-openai"
)

func toolCall(name string) []openai.ToolCall {
	return []openai.ToolCall{{
		Function: openai.FunctionCall{Name: name},
	}}
}

var _ = Describe("lastToolCallName", func() {
	It("returns the tool that produced the closing tool result", func() {
		name, ok := lastToolCallName(Messages{
			{Role: "user", Content: "hi"},
			{Role: "assistant", ToolCalls: toolCall("stop")},
			{Role: "tool", Content: "done"},
		})

		Expect(ok).To(BeTrue())
		Expect(name).To(Equal("stop"))
	})

	It("reports no tool call when the conversation is empty", func() {
		name, ok := lastToolCallName(Messages{})

		Expect(ok).To(BeFalse())
		Expect(name).To(BeEmpty())
	})

	It("reports no tool call when the conversation does not end in a tool result", func() {
		name, ok := lastToolCallName(Messages{
			{Role: "assistant", ToolCalls: toolCall("stop")},
			{Role: "assistant", Content: "all done"},
		})

		Expect(ok).To(BeFalse())
		Expect(name).To(BeEmpty())
	})

	// A tool result with nothing before it leaves no message to read the call
	// from; indexing len-2 would reach behind the slice.
	It("reports no tool call when the tool result is the only message", func() {
		name, ok := lastToolCallName(Messages{
			{Role: "tool", Content: "orphaned"},
		})

		Expect(ok).To(BeFalse())
		Expect(name).To(BeEmpty())
	})

	// The panic this guard exists for: the model returned no tool selection —
	// after a context-window overflow, for instance — so the message preceding
	// the tool result carries an empty ToolCalls slice.
	It("reports no tool call when the preceding message carries none", func() {
		name, ok := lastToolCallName(Messages{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "", ToolCalls: nil},
			{Role: "tool", Content: "done"},
		})

		Expect(ok).To(BeFalse())
		Expect(name).To(BeEmpty())
	})

	It("reads the first tool call when the preceding message carries several", func() {
		name, ok := lastToolCallName(Messages{
			{Role: "assistant", ToolCalls: []openai.ToolCall{
				{Function: openai.FunctionCall{Name: "first"}},
				{Function: openai.FunctionCall{Name: "second"}},
			}},
			{Role: "tool", Content: "done"},
		})

		Expect(ok).To(BeTrue())
		Expect(name).To(Equal("first"))
	})
})
