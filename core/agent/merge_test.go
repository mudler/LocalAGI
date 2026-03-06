package agent

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mergeLeadingSystemMessages", func() {
	It("merges multiple leading system messages into one", func() {
		conv := Messages{
			{Role: SystemRole, Content: "You are a helper."},
			{Role: SystemRole, Content: "Given the user input you have the following in memory:\n- fact1"},
			{Role: "user", Content: "hello"},
		}
		out := conv.mergeLeadingSystemMessages()
		Expect(out).To(HaveLen(2))
		Expect(out[0].Role).To(Equal(SystemRole))
		Expect(out[0].Content).To(Equal("You are a helper.\n\nGiven the user input you have the following in memory:\n- fact1"))
		Expect(out[1].Role).To(Equal("user"))
		Expect(out[1].Content).To(Equal("hello"))
	})

	It("prepends prefix blocks in order (self-eval then HUD)", func() {
		conv := Messages{
			{Role: SystemRole, Content: "Main system prompt."},
			{Role: "user", Content: "hi"},
		}
		out := conv.mergeLeadingSystemMessages("Self-eval block.", "HUD block.")
		Expect(out).To(HaveLen(2))
		Expect(out[0].Role).To(Equal(SystemRole))
		Expect(out[0].Content).To(Equal("Self-eval block.\n\nHUD block.\n\nMain system prompt."))
		Expect(out[1].Role).To(Equal("user"))
	})

	It("skips empty prefix blocks", func() {
		conv := Messages{
			{Role: SystemRole, Content: "Only this."},
			{Role: "user", Content: "hi"},
		}
		out := conv.mergeLeadingSystemMessages("", "HUD.", "")
		Expect(out[0].Content).To(Equal("HUD.\n\nOnly this."))
	})

	It("leaves mid-conversation system messages unchanged", func() {
		conv := Messages{
			{Role: SystemRole, Content: "Leading system."},
			{Role: "user", Content: "message with images"},
			{Role: SystemRole, Content: "Image explainer (would be rectified elsewhere)."},
			{Role: "assistant", Content: "reply"},
		}
		out := conv.mergeLeadingSystemMessages()
		Expect(out).To(HaveLen(4))
		Expect(out[0].Role).To(Equal(SystemRole))
		Expect(out[0].Content).To(Equal("Leading system."))
		Expect(out[1].Role).To(Equal("user"))
		Expect(out[2].Role).To(Equal(SystemRole))
		Expect(out[2].Content).To(Equal("Image explainer (would be rectified elsewhere)."))
		Expect(out[3].Role).To(Equal("assistant"))
	})

	It("returns conv unchanged when there are no leading system messages and no prefix blocks", func() {
		conv := Messages{
			{Role: "user", Content: "hi"},
		}
		out := conv.mergeLeadingSystemMessages()
		Expect(out).To(Equal(conv))
	})

	It("returns only prefix blocks as single system message when conv has no leading system messages", func() {
		conv := Messages{
			{Role: "user", Content: "hi"},
		}
		out := conv.mergeLeadingSystemMessages("Self-eval.", "HUD.")
		Expect(out).To(HaveLen(2))
		Expect(out[0].Role).To(Equal(SystemRole))
		Expect(out[0].Content).To(Equal("Self-eval.\n\nHUD."))
		Expect(out[1].Role).To(Equal("user"))
	})

	It("produces exactly one leading system message with config + RAG + HUD content", func() {
		conv := Messages{
			{Role: SystemRole, Content: "RAG: memory context"},
			{Role: SystemRole, Content: "Config system prompt."},
			{Role: "user", Content: "hello"},
		}
		out := conv.mergeLeadingSystemMessages("Self-eval.", "HUD.")
		Expect(out).To(HaveLen(2))
		Expect(out[0].Role).To(Equal(SystemRole))
		Expect(out[0].Content).To(ContainSubstring("Self-eval."))
		Expect(out[0].Content).To(ContainSubstring("HUD."))
		Expect(out[0].Content).To(ContainSubstring("RAG: memory context"))
		Expect(out[0].Content).To(ContainSubstring("Config system prompt."))
		systemCount := 0
		for _, m := range out {
			if m.Role == SystemRole {
				systemCount++
			}
		}
		Expect(systemCount).To(Equal(1))
	})
})
