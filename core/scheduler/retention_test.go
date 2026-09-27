package scheduler_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/LocalAGI/core/scheduler"
)

var _ = Describe("Task run retention", func() {
	var tempFile string

	BeforeEach(func() {
		f, err := os.CreateTemp("", "retention_test_*.json")
		Expect(err).NotTo(HaveOccurred())
		tempFile = f.Name()
		f.Close()
		DeferCleanup(func() { os.Remove(tempFile) })
	})

	newStore := func(p scheduler.RetentionPolicy) *scheduler.JSONStore {
		store, err := scheduler.NewJSONStoreWithRetention(tempFile, p)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(store.Close)
		return store
	}

	newTask := func(store scheduler.TaskStore, agent string) *scheduler.Task {
		task, err := scheduler.NewTask(agent, "prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
		Expect(err).NotTo(HaveOccurred())
		Expect(store.Create(task)).To(Succeed())
		return task
	}

	Context("MaxRunsPerTask", func() {
		It("keeps only the newest runs for a task", func() {
			store := newStore(scheduler.RetentionPolicy{MaxRunsPerTask: 3})
			task := newTask(store, "agent")

			for i := 0; i < 10; i++ {
				run := scheduler.NewTaskRun(task.ID)
				run.Status = "success"
				run.Result = string(rune('a' + i))
				Expect(store.LogRun(run)).To(Succeed())
			}

			runs, err := store.GetRuns(task.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(runs).To(HaveLen(3))
			// GetRuns walks backwards, so the newest run comes first.
			Expect(runs[0].Result).To(Equal("j"))
			Expect(runs[2].Result).To(Equal("h"))
		})

		It("applies the cap per task rather than across the whole store", func() {
			store := newStore(scheduler.RetentionPolicy{MaxRunsPerTask: 2})
			first := newTask(store, "agent-one")
			second := newTask(store, "agent-two")

			for _, id := range []string{first.ID, second.ID} {
				for i := 0; i < 4; i++ {
					Expect(store.LogRun(scheduler.NewTaskRun(id))).To(Succeed())
				}
			}

			firstRuns, err := store.GetRuns(first.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstRuns).To(HaveLen(2))

			secondRuns, err := store.GetRuns(second.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(secondRuns).To(HaveLen(2))
		})

		It("keeps every run when the cap is zero", func() {
			store := newStore(scheduler.RetentionPolicy{MaxRunsPerTask: 0})
			task := newTask(store, "agent")

			for i := 0; i < 8; i++ {
				Expect(store.LogRun(scheduler.NewTaskRun(task.ID))).To(Succeed())
			}

			runs, err := store.GetRuns(task.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(runs).To(HaveLen(8))
		})
	})

	Context("MaxRunAge", func() {
		It("removes runs older than the window", func() {
			store := newStore(scheduler.RetentionPolicy{MaxRunAge: time.Hour})
			task := newTask(store, "agent")

			stale := scheduler.NewTaskRun(task.ID)
			stale.RunAt = time.Now().Add(-48 * time.Hour)
			stale.Result = "stale"
			Expect(store.LogRun(stale)).To(Succeed())

			fresh := scheduler.NewTaskRun(task.ID)
			fresh.Result = "fresh"
			Expect(store.LogRun(fresh)).To(Succeed())

			runs, err := store.GetRuns(task.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(runs).To(HaveLen(1))
			Expect(runs[0].Result).To(Equal("fresh"))
		})

		It("keeps every run when the window is zero", func() {
			store := newStore(scheduler.RetentionPolicy{MaxRunAge: 0})
			task := newTask(store, "agent")

			stale := scheduler.NewTaskRun(task.ID)
			stale.RunAt = time.Now().Add(-10000 * time.Hour)
			Expect(store.LogRun(stale)).To(Succeed())

			runs, err := store.GetRuns(task.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(runs).To(HaveLen(1))
		})
	})

	Context("orphaned runs", func() {
		It("drops runs whose task no longer exists", func() {
			store := newStore(scheduler.RetentionPolicy{MaxRunsPerTask: 5})
			task := newTask(store, "agent")

			Expect(store.LogRun(scheduler.NewTaskRun(task.ID))).To(Succeed())
			Expect(store.Delete(task.ID)).To(Succeed())

			// Any later write triggers a sweep that reaps the orphan.
			survivor := newTask(store, "agent")
			Expect(store.LogRun(scheduler.NewTaskRun(survivor.ID))).To(Succeed())

			orphaned, err := store.GetRuns(task.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(orphaned).To(BeEmpty())
		})
	})

	Context("backwards compatibility", func() {
		It("retains everything when constructed without a policy", func() {
			store, err := scheduler.NewJSONStore(tempFile)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(store.Close)

			task := newTask(store, "agent")
			for i := 0; i < 20; i++ {
				Expect(store.LogRun(scheduler.NewTaskRun(task.ID))).To(Succeed())
			}

			runs, err := store.GetRuns(task.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(runs).To(HaveLen(20))
		})
	})

	Context("existing history", func() {
		It("prunes a backlog loaded from disk on the first write", func() {
			seed, err := scheduler.NewJSONStore(tempFile)
			Expect(err).NotTo(HaveOccurred())
			task := newTask(seed, "agent")
			for i := 0; i < 50; i++ {
				Expect(seed.LogRun(scheduler.NewTaskRun(task.ID))).To(Succeed())
			}
			Expect(seed.Close()).To(Succeed())

			store := newStore(scheduler.RetentionPolicy{MaxRunsPerTask: 5})
			Expect(store.LogRun(scheduler.NewTaskRun(task.ID))).To(Succeed())

			runs, err := store.GetRuns(task.ID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(runs).To(HaveLen(5))
		})
	})
})
