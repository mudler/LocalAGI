package scheduler_test

import (
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/LocalAGI/core/scheduler"
)

var _ = Describe("Task creation policy", func() {
	var store scheduler.TaskStore

	BeforeEach(func() {
		f, err := os.CreateTemp("", "creation_test_*.json")
		Expect(err).NotTo(HaveOccurred())
		name := f.Name()
		f.Close()
		DeferCleanup(func() { os.Remove(name) })

		store, err = scheduler.NewJSONStore(name)
		Expect(err).NotTo(HaveOccurred())
	})

	newSched := func(p scheduler.CreationPolicy) *scheduler.Scheduler {
		return scheduler.NewSchedulerWithPolicy(store, &MockExecutor{}, time.Hour, p)
	}

	create := func(sched *scheduler.Scheduler, agent, prompt, cron string) (*scheduler.Task, error) {
		task, err := scheduler.NewTask(agent, prompt, scheduler.ScheduleTypeCron, cron)
		Expect(err).NotTo(HaveOccurred())
		return sched.CreateTask(task)
	}

	countTasks := func(agent string) int {
		tasks, err := store.GetByAgent(agent)
		Expect(err).NotTo(HaveOccurred())
		return len(tasks)
	}

	Context("dedupe", func() {
		It("returns the existing task instead of creating a second copy", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: true})

			first, err := create(sched, "agent", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			second, err := create(sched, "agent", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			Expect(second.ID).To(Equal(first.ID))
			Expect(countTasks("agent")).To(Equal(1))
		})

		It("collapses case, punctuation and whitespace differences", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: true})

			first, err := create(sched, "agent", "Check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			for _, variant := range []string{
				"check open issues",
				"Check  open   issues",
				"  check open issues  ",
				"Check open issues.",
				"CHECK OPEN ISSUES!",
			} {
				again, err := create(sched, "agent", variant, "0 9 * * *")
				Expect(err).NotTo(HaveOccurred(), variant)
				Expect(again.ID).To(Equal(first.ID), variant)
			}

			Expect(countTasks("agent")).To(Equal(1))
		})

		It("treats a different schedule as a different task", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: true})

			first, err := create(sched, "agent", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			second, err := create(sched, "agent", "check open issues", "0 */2 * * *")
			Expect(err).NotTo(HaveOccurred())

			Expect(second.ID).NotTo(Equal(first.ID))
			Expect(countTasks("agent")).To(Equal(2))
		})

		It("scopes duplicates to one agent", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: true})

			first, err := create(sched, "agent-one", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			second, err := create(sched, "agent-two", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			Expect(second.ID).NotTo(Equal(first.ID))
			Expect(countTasks("agent-one")).To(Equal(1))
			Expect(countTasks("agent-two")).To(Equal(1))
		})

		It("treats a different schedule type as a different task", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: true})

			cron, err := scheduler.NewTask("agent", "ping", scheduler.ScheduleTypeCron, "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())
			created, err := sched.CreateTask(cron)
			Expect(err).NotTo(HaveOccurred())

			once, err := scheduler.NewTask("agent", "ping", scheduler.ScheduleTypeOnce, "1h")
			Expect(err).NotTo(HaveOccurred())
			second, err := sched.CreateTask(once)
			Expect(err).NotTo(HaveOccurred())

			Expect(second.ID).NotTo(Equal(created.ID))
			Expect(countTasks("agent")).To(Equal(2))
		})

		// This pins a known limit rather than a wish. The observed damage came
		// from an agent rewording one reminder seven ways; only the ceiling
		// bounds that, and nobody should later assume dedupe catches it.
		It("does not merge reworded prompts", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: true})

			first, err := create(sched, "agent", "results will be reported by next 2 hours", "0 */2 * * *")
			Expect(err).NotTo(HaveOccurred())

			second, err := create(sched, "agent", "results will be reported in the next 2 hours", "0 */2 * * *")
			Expect(err).NotTo(HaveOccurred())

			Expect(second.ID).NotTo(Equal(first.ID))
			Expect(countTasks("agent")).To(Equal(2))
		})

		It("creates every copy when dedupe is off", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: false})

			first, err := create(sched, "agent", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			second, err := create(sched, "agent", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			Expect(second.ID).NotTo(Equal(first.ID))
			Expect(countTasks("agent")).To(Equal(2))
		})
	})

	Context("per-agent ceiling", func() {
		It("refuses a new task past the limit and names the recovery actions", func() {
			sched := newSched(scheduler.CreationPolicy{MaxTasksPerAgent: 2})

			_, err := create(sched, "agent", "one", "0 1 * * *")
			Expect(err).NotTo(HaveOccurred())
			_, err = create(sched, "agent", "two", "0 2 * * *")
			Expect(err).NotTo(HaveOccurred())

			_, err = create(sched, "agent", "three", "0 3 * * *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(scheduler.ListTasksActionName))
			Expect(err.Error()).To(ContainSubstring(scheduler.RemoveTaskActionName))
			Expect(countTasks("agent")).To(Equal(2))
		})

		It("counts the ceiling per agent", func() {
			sched := newSched(scheduler.CreationPolicy{MaxTasksPerAgent: 1})

			_, err := create(sched, "agent-one", "one", "0 1 * * *")
			Expect(err).NotTo(HaveOccurred())

			_, err = create(sched, "agent-two", "one", "0 1 * * *")
			Expect(err).NotTo(HaveOccurred())

			_, err = create(sched, "agent-one", "two", "0 2 * * *")
			Expect(err).To(HaveOccurred())
		})

		// A duplicate adds nothing, so it must not be refused for lack of room.
		It("lets a duplicate through at the ceiling", func() {
			sched := newSched(scheduler.CreationPolicy{Dedupe: true, MaxTasksPerAgent: 1})

			first, err := create(sched, "agent", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())

			again, err := create(sched, "agent", "check open issues", "0 9 * * *")
			Expect(err).NotTo(HaveOccurred())
			Expect(again.ID).To(Equal(first.ID))
		})

		It("imposes no limit when the ceiling is zero", func() {
			sched := newSched(scheduler.CreationPolicy{MaxTasksPerAgent: 0})

			for _, cron := range []string{"0 1 * * *", "0 2 * * *", "0 3 * * *", "0 4 * * *"} {
				_, err := create(sched, "agent", "task "+cron, cron)
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(countTasks("agent")).To(Equal(4))
		})
	})

	Context("normalization helper", func() {
		It("reduces a prompt to its comparable form", func() {
			Expect(scheduler.NormalizePrompt("  Check   Open Issues!  ")).To(Equal("check open issues"))
			Expect(scheduler.NormalizePrompt("check open issues")).To(Equal("check open issues"))
			Expect(strings.TrimSpace(scheduler.NormalizePrompt(""))).To(BeEmpty())
		})
	})
})
