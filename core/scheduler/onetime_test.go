package scheduler_test

import (
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/LocalAGI/core/scheduler"
)

// recordingStore notes which mutating calls the scheduler makes, so a spec can
// assert on the sequence rather than on a log line.
type recordingStore struct {
	scheduler.TaskStore

	mu    sync.Mutex
	calls []string
}

func (r *recordingStore) record(call string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *recordingStore) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func (r *recordingStore) Delete(id string) error {
	r.record("delete:" + id)
	return r.TaskStore.Delete(id)
}

func (r *recordingStore) Update(task *scheduler.Task) error {
	r.record("update:" + task.ID)
	return r.TaskStore.Update(task)
}

var _ = Describe("One-time task execution", func() {
	var store *recordingStore
	var sched *scheduler.Scheduler

	BeforeEach(func() {
		f, err := os.CreateTemp("", "onetime_test_*.json")
		Expect(err).NotTo(HaveOccurred())
		name := f.Name()
		f.Close()
		DeferCleanup(func() { os.Remove(name) })

		base, err := scheduler.NewJSONStore(name)
		Expect(err).NotTo(HaveOccurred())

		store = &recordingStore{TaskStore: base}
		sched = scheduler.NewScheduler(store, &MockExecutor{}, 50*time.Millisecond)
		sched.Start()
		DeferCleanup(sched.Stop)
	})

	// The update used to run unconditionally after the delete, so every
	// one-time task logged "task not found" on the way out.
	It("does not update a one-time task after deleting it", func() {
		task, err := scheduler.NewTask("agent", "ping", scheduler.ScheduleTypeOnce, "0s")
		Expect(err).NotTo(HaveOccurred())
		task.NextRun = time.Now().Add(-time.Second)

		_, err = sched.CreateTask(task)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() []string {
			return store.Calls()
		}, "3s", "50ms").Should(ContainElement("delete:" + task.ID))

		Consistently(func() []string {
			return store.Calls()
		}, "300ms", "50ms").ShouldNot(ContainElement("update:" + task.ID))
	})

	It("still reschedules a recurring task after it runs", func() {
		task, err := scheduler.NewTask("agent", "ping", scheduler.ScheduleTypeCron, "* * * * *")
		Expect(err).NotTo(HaveOccurred())
		task.NextRun = time.Now().Add(-time.Second)

		_, err = sched.CreateTask(task)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() []string {
			return store.Calls()
		}, "3s", "50ms").Should(ContainElement("update:" + task.ID))

		Expect(store.Calls()).ToNot(ContainElement("delete:" + task.ID))
	})
})
