package agent

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/mudler/LocalAGI/core/sse"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/xlog"
)

type Observer interface {
	NewObservable() *types.Observable
	Update(types.Observable)
	History() []types.Observable
	ClearHistory()
}

// historyRingSize is the number of observables kept in the ring buffer. When full,
// the oldest entry is overwritten. The UI builds a tree from parent_id; if a parent
// is evicted before its children, those children will appear as roots or be omitted.
const historyRingSize = 500

type SSEObserver struct {
	agent   string
	maxID   int32
	manager sse.Manager

	mutex       sync.Mutex
	history     []types.Observable
	historyLast int
}

func NewSSEObserver(agent string, manager sse.Manager) *SSEObserver {
	return &SSEObserver{
		agent:   agent,
		maxID:   1,
		manager: manager,
		history: make([]types.Observable, historyRingSize),
	}
}

func (s *SSEObserver) NewObservable() *types.Observable {
	id := atomic.AddInt32(&s.maxID, 1)

	return &types.Observable{
		ID:    id - 1,
		Agent: s.agent,
	}
}

func (s *SSEObserver) Update(obs types.Observable) {
	data, err := json.Marshal(obs)
	if err != nil {
		xlog.Error("Error marshaling observable", "error", err)
		return
	}
	msg := sse.NewMessage(string(data)).WithEvent("observable_update")
	s.manager.Send(msg)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, o := range s.history {
		if o.ID == obs.ID {
			s.history[i] = obs
			return
		}
	}

	s.history[s.historyLast] = obs
	s.historyLast += 1
	if s.historyLast >= len(s.history) {
		s.historyLast = 0
	}
}

func (s *SSEObserver) History() []types.Observable {
	h := make([]types.Observable, 0, 20)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, obs := range s.history {
		if obs.ID == 0 {
			continue
		}

		h = append(h, obs)
	}

	return h
}

func (s *SSEObserver) ClearHistory() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.history = make([]types.Observable, historyRingSize)
	s.historyLast = 0
}
