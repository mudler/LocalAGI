package conversations

import (
	"sync"
	"time"

	"github.com/mudler/xlog"
)

// Pruner sweeps a conversation directory on an interval.
//
// One Pruner covers the whole directory, which is shared by every agent in a
// pool. Running one per agent would mean N concurrent sweeps over the same tens
// of thousands of files.
type Pruner struct {
	dir      string
	policy   RetentionPolicy
	interval time.Duration

	mu     sync.Mutex
	stop   chan struct{}
	done   chan struct{}
	closed bool
}

// NewPruner returns a sweeper for dir. It does nothing until Start is called,
// and nothing at all if the policy is disabled.
func NewPruner(dir string, policy RetentionPolicy, interval time.Duration) *Pruner {
	return &Pruner{
		dir:      dir,
		policy:   policy,
		interval: interval,
	}
}

// Start begins sweeping in the background. The first sweep runs immediately
// rather than after a full interval, so a process that restarts often still
// makes progress against a backlog.
func (p *Pruner) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.dir == "" || !p.policy.Enabled() || p.interval <= 0 || p.stop != nil {
		return
	}

	p.stop = make(chan struct{})
	p.done = make(chan struct{})

	go p.loop(p.stop, p.done)
}

func (p *Pruner) loop(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	p.sweep()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			p.sweep()
		}
	}
}

func (p *Pruner) sweep() {
	removed, err := Prune(p.dir, p.policy, time.Now())
	if err != nil {
		xlog.Error("Failed to prune conversations", "dir", p.dir, "error", err)
		return
	}
	if removed > 0 {
		xlog.Info("Pruned conversations", "dir", p.dir, "removed", removed)
	}
}

// Running reports whether a sweeper is currently active.
func (p *Pruner) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.stop != nil && !p.closed
}

// Stop ends the sweep and waits for an in-flight one to finish. It is safe to
// call more than once, and safe to call without Start.
func (p *Pruner) Stop() {
	p.mu.Lock()
	if p.stop == nil || p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.stop)
	done := p.done
	p.mu.Unlock()

	<-done
}
