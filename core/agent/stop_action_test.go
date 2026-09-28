package agent

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/cogito/tests/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// stopTestAction is a plain regular action the model can call before stopping.
type stopTestAction struct {
	types.BaseAction
}

func (stopTestAction) Run(context.Context, *types.AgentSharedState, types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{Result: "state read"}, nil
}

func (stopTestAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{Name: "read_state", Description: "read the state"}
}

// A model that decides to stop ends the run on purpose. The tool callback
// rejects the stop call (Approved=false), which cogito reports as
// ErrToolCallCallbackInterrupted; that must not surface as a job failure.
var _ = Describe("stop action in consumeJob", func() {
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
			WithActions(&stopTestAction{}),
			CanStopItself,
		)
		Expect(err).ToNot(HaveOccurred())
		llm = mock.NewMockOpenAIClient()
		a.llm = llm
	})

	It("finishes without an error when the model stops after a tool", func() {
		llm.AddCreateChatCompletionFunction("read_state", `{}`)
		llm.AddCreateChatCompletionFunction("stop", `{}`)
		llm.SetAskResponse("done")

		job := types.NewJob(types.WithText("read the state, then stop"))
		a.consumeJob(job, UserRole)

		res, err := job.Result.WaitResult(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Error).ToNot(HaveOccurred())
	})

	// AGNTSIO#688: in a user chat job the model chose stop instead of answering and the chat
	// received nothing. A stop there must still end with a reply the user can read.
	It("delivers a reply when the model stops in a user chat job", func() {
		llm.AddCreateChatCompletionFunction("read_state", `{}`)
		llm.AddCreateChatCompletionFunction("stop", `{}`)
		llm.SetAskResponse("I found no suitable template, so I cannot draft this offer.")

		job := types.NewJob(types.WithText("draft an offer"))
		a.consumeJob(job, UserRole)

		res, err := job.Result.WaitResult(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Error).ToNot(HaveOccurred())
		Expect(res.Response).To(Equal("I found no suitable template, so I cannot draft this offer."))
	})

	It("still ends silently when an autonomous (system) run stops", func() {
		// Counter-check: no user is waiting, so no extra reply is requested.
		llm.AddCreateChatCompletionFunction("read_state", `{}`)
		llm.AddCreateChatCompletionFunction("stop", `{}`)

		job := types.NewJob(types.WithText("periodic check"))
		a.consumeJob(job, SystemRole)

		res, err := job.Result.WaitResult(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Error).ToNot(HaveOccurred())
		Expect(res.Response).To(BeEmpty())
	})

	It("reports an error when the reply after a stop cannot be produced", func() {
		llm.AddCreateChatCompletionFunction("stop", `{}`)
		llm.SetAskError(errors.New("backend down"))

		job := types.NewJob(types.WithText("hello"))
		a.consumeJob(job, UserRole)

		res, err := job.Result.WaitResult(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Error).To(HaveOccurred())
	})

	It("still reports a genuine failure", func() {
		// Counter-check: nothing queued in the mock, so the LLM call fails.
		job := types.NewJob(types.WithText("hello"))
		a.consumeJob(job, UserRole)

		res, err := job.Result.WaitResult(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Error).To(HaveOccurred())
	})
})
