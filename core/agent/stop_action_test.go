package agent

import (
	"context"
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

		job := types.NewJob(types.WithText("read the state, then stop"))
		a.consumeJob(job, UserRole)

		res, err := job.Result.WaitResult(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Error).ToNot(HaveOccurred())
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
