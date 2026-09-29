package agent

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/cogito/tests/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// When the required-tool gate gives up after its attempt cap, the answer leaves the agent
// ungated. Without a notice the user cannot tell a validated answer from one that skipped
// the required step. WithRequiredToolBypassNotice appends a visible notice in exactly that
// case, and only then.
var _ = Describe("required-tool bypass notice", func() {
	const notice = "NOT VALIDATED: this answer skipped the required check."

	Describe("appendBypassNotice", func() {
		It("appends the notice on its own paragraph", func() {
			Expect(appendBypassNotice("answer\n", notice)).To(Equal("answer\n\n" + notice))
		})
		It("leaves the answer untouched when no notice is configured", func() {
			Expect(appendBypassNotice("answer", "")).To(Equal("answer"))
		})
		It("does not append twice", func() {
			once := appendBypassNotice("answer", notice)
			Expect(appendBypassNotice(once, notice)).To(Equal(once))
		})
	})

	Describe("options", func() {
		It("stores the notice", func() {
			o := defaultOptions()
			Expect(WithRequiredToolBypassNotice(notice)(o)).To(Succeed())
			Expect(o.requiredFinishBypassNotice).To(Equal(notice))
		})
	})

	newAgent := func(extra ...Option) (*Agent, *mock.MockOpenAIClient) {
		opts := append([]Option{
			WithModel("test-model"),
			WithLLMAPIURL("http://127.0.0.1:1"),
			WithSchedulerStorePath(filepath.Join(GinkgoT().TempDir(), "tasks.json")),
			WithActions(&gateTestAction{name: "check_policy"}),
			WithRequiredToolBeforeFinish("check_policy"),
			WithRequiredToolBeforeFinishAttempts(1),
		}, extra...)
		a, err := New(opts...)
		Expect(err).ToNot(HaveOccurred())
		llm := mock.NewMockOpenAIClient()
		a.llm = llm
		return a, llm
	}

	run := func(a *Agent, job *types.Job) *types.JobResult {
		a.consumeJob(job, UserRole)
		res, err := job.Result.WaitResult(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Error).ToNot(HaveOccurred())
		return res
	}

	Context("text finalization", func() {
		It("marks an answer that was let through after the attempt cap", func() {
			a, llm := newAgent(WithRequiredToolBypassNotice(notice))
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("ungated answer")
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("still ungated")

			res := run(a, types.NewJob(types.WithText("hello")))
			Expect(res.Response).To(HavePrefix("still ungated"))
			Expect(res.Response).To(HaveSuffix(notice))
		})

		It("does not mark an answer whose required tool passed", func() {
			a, llm := newAgent(WithRequiredToolBypassNotice(notice))
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("ungated answer")
			llm.AddCreateChatCompletionFunction("check_policy", `{}`)
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("gated answer")

			res := run(a, types.NewJob(types.WithText("hello")))
			Expect(res.Response).To(Equal("gated answer"))
		})

		It("keeps today's behaviour when no notice is configured", func() {
			a, llm := newAgent()
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("ungated answer")
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("still ungated")

			res := run(a, types.NewJob(types.WithText("hello")))
			Expect(res.Response).To(Equal("still ungated"))
		})
	})

	Context("send_message", func() {
		It("marks a message sent after the attempt cap", func() {
			a, llm := newAgent(WithRequiredToolBypassNotice(notice))
			llm.AddCreateChatCompletionFunction("send_message", `{"message": "draft"}`)
			llm.AddCreateChatCompletionFunction("send_message", `{"message": "draft"}`)

			job := types.NewJob(types.WithText("scheduled work"), types.WithMetadata(map[string]any{"type": "scheduled"}))
			res := run(a, job)
			Expect(res.Conversation).ToNot(BeEmpty())
			sent := res.Conversation[0].Content
			Expect(strings.HasPrefix(sent, "draft")).To(BeTrue(), sent)
			Expect(sent).To(HaveSuffix(notice))
		})
	})

	Context("closing reply after stop (AGNTSIO#688)", func() {
		It("marks the closing reply when it is finalized ungated", func() {
			a, llm := newAgent(WithRequiredToolBypassNotice(notice), CanStopItself)
			llm.AddCreateChatCompletionFunction("stop", `{}`)
			llm.SetAskResponse("closing reply")
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("closing reply again")

			res := run(a, types.NewJob(types.WithText("draft an offer")))
			Expect(res.Response).To(HaveSuffix(notice))
		})
	})
})
