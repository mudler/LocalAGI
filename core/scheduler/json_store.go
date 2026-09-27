package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// RetentionPolicy bounds how much execution history the store keeps.
// A zero value for either field disables that limit.
//
// Run history is append-only otherwise, and every save re-marshals the whole
// file — an unbounded task_runs list makes each task execution progressively
// more expensive.
type RetentionPolicy struct {
	// MaxRunsPerTask keeps only this many of the most recent runs per task.
	MaxRunsPerTask int
	// MaxRunAge removes runs older than this.
	MaxRunAge time.Duration
}

// Enabled reports whether the policy would remove anything at all.
func (p RetentionPolicy) Enabled() bool {
	return p.MaxRunsPerTask > 0 || p.MaxRunAge > 0
}

// JSONStore implements TaskStore using JSON file storage
type JSONStore struct {
	filePath  string
	mu        sync.RWMutex
	data      *storeData
	retention RetentionPolicy
}

type storeData struct {
	Tasks    []*Task    `json:"tasks"`
	TaskRuns []*TaskRun `json:"task_runs"`
}

// NewJSONStore creates a new JSON-based task store that retains all history.
func NewJSONStore(filePath string) (*JSONStore, error) {
	return NewJSONStoreWithRetention(filePath, RetentionPolicy{})
}

// NewJSONStoreWithRetention creates a store that bounds its run history.
// An existing backlog on disk is pruned at load, so a store that has already
// grown does not have to wait for the next task execution to shrink.
func NewJSONStoreWithRetention(filePath string, retention RetentionPolicy) (*JSONStore, error) {
	store := &JSONStore{
		filePath:  filePath,
		retention: retention,
		data: &storeData{
			Tasks:    make([]*Task, 0),
			TaskRuns: make([]*TaskRun, 0),
		},
	}

	if err := store.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load store: %w", err)
		}
		// File doesn't exist, create it
		if err := store.save(); err != nil {
			return nil, fmt.Errorf("failed to create store file: %w", err)
		}
		return store, nil
	}

	if store.pruneRuns() > 0 {
		if err := store.save(); err != nil {
			return nil, fmt.Errorf("failed to persist pruned store: %w", err)
		}
	}

	return store, nil
}

// Create adds a new task
func (s *JSONStore) Create(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate ID
	for _, t := range s.data.Tasks {
		if t.ID == task.ID {
			return fmt.Errorf("task with ID %s already exists", task.ID)
		}
	}

	s.data.Tasks = append(s.data.Tasks, task)
	return s.save()
}

// Get retrieves a task by ID
func (s *JSONStore) Get(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, task := range s.data.Tasks {
		if task.ID == id {
			return task, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", id)
}

// GetAll retrieves all tasks
func (s *JSONStore) GetAll() ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	tasks := make([]*Task, len(s.data.Tasks))
	copy(tasks, s.data.Tasks)
	return tasks, nil
}

// GetDue retrieves tasks that are due for execution
func (s *JSONStore) GetDue() ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	dueTasks := make([]*Task, 0)

	for _, task := range s.data.Tasks {
		if task.Status == TaskStatusActive && now.After(task.NextRun) {
			dueTasks = append(dueTasks, task)
		}
	}

	return dueTasks, nil
}

// GetByAgent retrieves all tasks for a specific agent
func (s *JSONStore) GetByAgent(agentName string) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentTasks := make([]*Task, 0)
	for _, task := range s.data.Tasks {
		if task.AgentName == agentName {
			agentTasks = append(agentTasks, task)
		}
	}

	return agentTasks, nil
}

// Update updates an existing task
func (s *JSONStore) Update(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.data.Tasks {
		if t.ID == task.ID {
			task.UpdatedAt = time.Now()
			s.data.Tasks[i] = task
			return s.save()
		}
	}

	return fmt.Errorf("task not found: %s", task.ID)
}

// Delete removes a task
func (s *JSONStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.data.Tasks {
		if task.ID == id {
			// Remove task from slice
			s.data.Tasks = append(s.data.Tasks[:i], s.data.Tasks[i+1:]...)
			return s.save()
		}
	}

	return fmt.Errorf("task not found: %s", id)
}

// LogRun records a task execution
func (s *JSONStore) LogRun(run *TaskRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.TaskRuns = append(s.data.TaskRuns, run)
	s.pruneRuns()
	return s.save()
}

// pruneRuns applies the retention policy to the run history and returns how
// many records it dropped. Callers must hold the write lock.
//
// It walks backwards so that "keep N" keeps the newest N, and drops runs whose
// task no longer exists — a one-time task is deleted right after its run is
// logged, and those orphans would otherwise accumulate forever.
func (s *JSONStore) pruneRuns() int {
	if !s.retention.Enabled() {
		return 0
	}

	known := make(map[string]struct{}, len(s.data.Tasks))
	for _, t := range s.data.Tasks {
		known[t.ID] = struct{}{}
	}

	var cutoff time.Time
	if s.retention.MaxRunAge > 0 {
		cutoff = time.Now().Add(-s.retention.MaxRunAge)
	}

	before := len(s.data.TaskRuns)
	kept := make([]*TaskRun, 0, before)
	perTask := map[string]int{}

	for i := before - 1; i >= 0; i-- {
		run := s.data.TaskRuns[i]

		if _, ok := known[run.TaskID]; !ok {
			continue
		}
		if !cutoff.IsZero() && run.RunAt.Before(cutoff) {
			continue
		}
		if s.retention.MaxRunsPerTask > 0 && perTask[run.TaskID] >= s.retention.MaxRunsPerTask {
			continue
		}

		perTask[run.TaskID]++
		kept = append(kept, run)
	}

	slices.Reverse(kept)
	s.data.TaskRuns = kept

	return before - len(kept)
}

// GetRuns retrieves execution history for a task
func (s *JSONStore) GetRuns(taskID string, limit int) ([]*TaskRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs := make([]*TaskRun, 0)
	for i := len(s.data.TaskRuns) - 1; i >= 0 && len(runs) < limit; i-- {
		if s.data.TaskRuns[i].TaskID == taskID {
			runs = append(runs, s.data.TaskRuns[i])
		}
	}

	return runs, nil
}

// Close releases resources (no-op for JSON store)
func (s *JSONStore) Close() error {
	return nil
}

// load reads data from the JSON file
func (s *JSONStore) load() error {
	file, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	// Handle empty file
	if len(file) == 0 {
		return nil
	}

	if err := json.Unmarshal(file, s.data); err != nil {
		return err
	}

	// Ensure slices are not nil after unmarshaling
	if s.data.Tasks == nil {
		s.data.Tasks = make([]*Task, 0)
	}
	if s.data.TaskRuns == nil {
		s.data.TaskRuns = make([]*TaskRun, 0)
	}

	return nil
}

// save writes data to the JSON file
func (s *JSONStore) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	basePath := filepath.Dir(s.filePath)
	os.MkdirAll(basePath, 0755)

	return os.WriteFile(s.filePath, data, 0644)
}
