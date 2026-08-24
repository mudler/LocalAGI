package action_test

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/scheduler"
	"github.com/mudler/LocalAGI/core/types"
)

// noopExecutor satisfies the scheduler's executor interface for specs that
// never let the scheduler poll.
type noopExecutor struct{}

func (noopExecutor) Execute(_ context.Context, _ string, _ string) (*scheduler.JobResult, error) {
	return &scheduler.JobResult{Response: "ok"}, nil
}

var _ = Describe("Reminder actions", func() {
	var sharedState *types.AgentSharedState
	var store scheduler.TaskStore

	newState := func(policy scheduler.CreationPolicy) {
		f, err := os.CreateTemp("", "reminder_action_test_*.json")
		Expect(err).NotTo(HaveOccurred())
		name := f.Name()
		f.Close()
		DeferCleanup(func() { os.Remove(name) })

		store, err = scheduler.NewJSONStore(name)
		Expect(err).NotTo(HaveOccurred())

		sharedState = types.NewAgentSharedState(0)
		sharedState.AgentName = "agent"
		sharedState.Scheduler = scheduler.NewSchedulerWithPolicy(store, noopExecutor{}, time.Hour, policy)
	}

	setRecurring := func(message, cron string) (types.ActionResult, error) {
		return action.NewRecurringReminder().Run(context.Background(), sharedState, types.ActionParams{
			"message":   message,
			"cron_expr": cron,
		})
	}

	Context("creating a duplicate recurring reminder", func() {
		BeforeEach(func() { newState(scheduler.CreationPolicy{Dedupe: true}) })

		It("reports the existing task rather than failing", func() {
			first, err := setRecurring("check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			second, err := setRecurring("Check open issues.", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			Expect(second.Metadata["task_id"]).To(Equal(first.Metadata["task_id"]))
			Expect(second.Result).To(ContainSubstring("already scheduled"))

			tasks, err := store.GetByAgent("agent")
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
		})
	})

	Context("hitting the per-agent ceiling", func() {
		BeforeEach(func() { newState(scheduler.CreationPolicy{MaxTasksPerAgent: 1}) })

		It("returns an error naming the actions that free room", func() {
			_, err := setRecurring("first", "0 1 * * *")
			Expect(err).NotTo(HaveOccurred())

			_, err = setRecurring("second", "0 2 * * *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list_tasks"))
			Expect(err.Error()).To(ContainSubstring("remove_task"))
		})
	})
})
