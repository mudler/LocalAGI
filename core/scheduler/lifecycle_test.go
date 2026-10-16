package scheduler_test

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mudler/LocalAGI/core/scheduler"
)

func newLifecycleScheduler(t *testing.T) *scheduler.Scheduler {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "sched_*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	store, err := scheduler.NewJSONStore(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return scheduler.NewScheduler(store, &MockExecutor{}, time.Millisecond)
}

// Agent.Run starts the scheduler on one goroutine while Agent.Stop can stop it
// from another. Start and Stop must be safe to call concurrently.
func TestStartStopConcurrent(t *testing.T) {
	for i := 0; i < 200; i++ {
		s := newLifecycleScheduler(t)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); s.Start() }()
		go func() { defer wg.Done(); s.Stop() }()
		wg.Wait()
		s.Stop()
	}
}

// Stop called twice must not panic or block.
func TestStopTwice(t *testing.T) {
	s := newLifecycleScheduler(t)
	s.Start()
	s.Stop()
	s.Stop()
}

// A scheduler that was stopped must not start again: its store is closed.
func TestStartAfterStopIsNoop(t *testing.T) {
	s := newLifecycleScheduler(t)
	s.Stop()
	s.Start()
	s.Stop()
}
