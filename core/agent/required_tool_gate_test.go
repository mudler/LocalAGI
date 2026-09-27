package agent

import (
	"context"
	"path/filepath"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/cogito/tests/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// gateTestAction is a regular (not user-defined) action the gate can require.
type gateTestAction struct {
	types.BaseAction
	name string
}

func (g *gateTestAction) Run(context.Context, *types.AgentSharedState, types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{Result: `{"ok": true}`}, nil
}

func (g *gateTestAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{Name: types.ActionDefinitionName(g.name), Description: "required check"}
}

var _ = Describe("required-tool gate", func() {
	Describe("requiredToolResultOK", func() {
		DescribeTable("decides whether a tool result counts as passed",
			func(result string, want bool) {
				Expect(requiredToolResultOK(result)).To(Equal(want))
			},
			Entry("clean ok true", `{"ok": true, "flags": []}`, true),
			Entry("clean ok false", `{"ok": false, "flags": [{"severity":"high"}]}`, false),
			Entry("compact ok true", `{"ok":true}`, true),
			Entry("surrounding whitespace", "\n  {\"ok\": true}\n", true),
			Entry("ok true embedded in text", `tool output: {"ok": true, "summary": {}} done`, true),
			Entry("second embedded object carries ok true", `first {"a": 1} then {"ok": true}`, true),
			Entry("garbage", `not json at all`, false),
			Entry("empty", ``, false),
			Entry("ok is a string, not a bool", `{"ok": "true"}`, false),
			Entry("ok missing", `{"status": "fine"}`, false),
			Entry("ok true only nested", `{"result": {"ok": true}, "ok": false}`, false),
			Entry("ok true only nested, wrapped in text", `out: {"result": {"ok": true}}`, false),
			Entry("ok true only inside a string value", `{"message": "\"ok\": true"}`, false),
			Entry("ok true text inside a string, wrapped", `log: {"message": "\"ok\": true"} end`, false),
			Entry("JSON array", `[{"ok": true}]`, false),
			Entry("unterminated object", `result {"ok": true`, false),
			Entry("bare substring without an object", `"ok": true`, false),
		)
	})

	Describe("requiredToolGate", func() {
		const max = 3

		It("allows the answer when the tool is not bound", func() {
			n := 0
			_, blocked := requiredToolGate(false, false, "nudge", &n, max)
			Expect(blocked).To(BeFalse())
			Expect(n).To(Equal(0))
		})

		It("allows the answer once the tool passed", func() {
			n := 0
			_, blocked := requiredToolGate(true, true, "nudge", &n, max)
			Expect(blocked).To(BeFalse())
			Expect(n).To(Equal(0))
		})

		It("blocks with an adjustment while the tool has not passed", func() {
			n := 0
			decision, blocked := requiredToolGate(true, false, "nudge", &n, max)
			Expect(blocked).To(BeTrue())
			// Approved=true makes cogito re-run tool selection instead of aborting the run.
			Expect(decision.Approved).To(BeTrue())
			Expect(decision.Adjustment).To(Equal("nudge"))
			Expect(n).To(Equal(1))
		})

		It("lets the answer through after the attempt cap", func() {
			n := 0
			for i := 0; i < max; i++ {
				_, blocked := requiredToolGate(true, false, "nudge", &n, max)
				Expect(blocked).To(BeTrue(), "attempt %d should still block", i+1)
			}
			_, blocked := requiredToolGate(true, false, "nudge", &n, max)
			Expect(blocked).To(BeFalse())
			Expect(n).To(Equal(max))
		})
	})

	Describe("textFinalizationNeedsRequiredTool", func() {
		const max = 3
		DescribeTable("decides whether a text answer must be deferred",
			func(available, passed bool, attempts int, role, content string, want bool) {
				Expect(textFinalizationNeedsRequiredTool(available, passed, attempts, max, role, content)).To(Equal(want))
			},
			Entry("tool unavailable", false, false, 0, "assistant", "answer", false),
			Entry("already passed", true, true, 0, "assistant", "answer", false),
			Entry("attempt cap reached", true, false, max, "assistant", "answer", false),
			Entry("last message is a tool result", true, false, 0, "tool", "", false),
			Entry("empty text answer", true, false, 0, "assistant", "   ", false),
			Entry("text answer without the tool", true, false, 0, "assistant", "here is my answer", true),
			Entry("still under the cap", true, false, max-1, "assistant", "answer", true),
		)
	})

	Describe("requiredFinishPromptFor", func() {
		It("names the tool by default", func() {
			Expect(requiredFinishPromptFor("check_policy", "")).To(ContainSubstring("check_policy"))
		})
		It("uses the override verbatim", func() {
			Expect(requiredFinishPromptFor("check_policy", "do the thing")).To(Equal("do the thing"))
		})
	})

	Describe("options", func() {
		It("is off by default", func() {
			o := defaultOptions()
			Expect(o.requiredFinishTool).To(BeEmpty())
		})

		It("stores the tool, prompt and attempt cap", func() {
			o := defaultOptions()
			for _, opt := range []Option{
				WithRequiredToolBeforeFinish("check_policy"),
				WithRequiredToolBeforeFinishPrompt("call it first"),
				WithRequiredToolBeforeFinishAttempts(5),
			} {
				Expect(opt(o)).To(Succeed())
			}
			Expect(o.requiredFinishTool).To(Equal("check_policy"))
			Expect(o.requiredFinishPrompt).To(Equal("call it first"))
			Expect(o.requiredFinishAttempts).To(Equal(5))
		})
	})

	Describe("text-finalization retry in consumeJob", func() {
		var (
			a   *Agent
			llm *mock.MockOpenAIClient
		)

		BeforeEach(func() {
			var err error
			a, err = New(
				WithModel("test-model"),
				WithLLMAPIURL("http://127.0.0.1:1"),
				WithSchedulerStorePath(filepath.Join(GinkgoT().TempDir(), "tasks.json")),
				WithActions(&gateTestAction{name: "check_policy"}),
				WithRequiredToolBeforeFinish("check_policy"),
			)
			Expect(err).ToNot(HaveOccurred())
			llm = mock.NewMockOpenAIClient()
			a.llm = llm
		})

		It("keeps the result recorded by a user-defined tool picked during a retry", func() {
			// First pass: the model picks no tool and answers with plain text, so the
			// required tool has not run and the gate sends a nudge.
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("ungated answer")
			// Retry: the model picks the caller's own tool. That hands control back to
			// the caller; the agent must not overwrite the job result afterwards.
			llm.AddCreateChatCompletionFunction("client_tool", `{"q": "x"}`)

			job := types.NewJob(
				types.WithText("hello"),
				types.WithUserTools([]types.ActionDefinition{{
					Name:        "client_tool",
					Description: "a tool the caller runs",
				}}),
			)

			a.consumeJob(job, UserRole)

			res, err := job.Result.WaitResult(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Error).ToNot(HaveOccurred())
			// replyWithToolCall leaves Response empty so the caller sees a tool call.
			Expect(res.Response).To(BeEmpty())
			Expect(res.State).ToNot(BeEmpty())
			Expect(res.State[len(res.State)-1].Action.Definition().Name.String()).To(Equal("client_tool"))
			last := res.Conversation[len(res.Conversation)-1]
			Expect(last.ToolCalls).To(HaveLen(1))
			Expect(last.ToolCalls[0].Function.Name).To(Equal("client_tool"))
		})

		It("finalizes with the gated answer once the retry runs the required tool", func() {
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("ungated answer")
			llm.AddCreateChatCompletionFunction("check_policy", `{}`)
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("gated answer")

			job := types.NewJob(types.WithText("hello"))
			a.consumeJob(job, UserRole)

			res, err := job.Result.WaitResult(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Error).ToNot(HaveOccurred())
			Expect(res.Response).To(Equal("gated answer"))
		})

		It("keeps the previous answer when a retry fails", func() {
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)
			llm.SetAskResponse("ungated answer")
			// The retry selects no tool and then fails to produce its answer.
			llm.AddCreateChatCompletionFunction("no_tool_to_call", `{}`)

			job := types.NewJob(types.WithText("hello"))
			a.consumeJob(job, UserRole)

			res, err := job.Result.WaitResult(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Error).ToNot(HaveOccurred())
			Expect(res.Response).To(Equal("ungated answer"))
		})
	})
})
