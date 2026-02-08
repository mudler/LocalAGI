package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONStore implements TaskStore using JSON file storage
type JSONStore struct {
	filePath string
	mu       sync.RWMutex
	data     *storeData
}

type storeData struct {
	Tasks    []*Task    `json:"tasks"`
	TaskRuns []*TaskRun `json:"task_runs"`
}

// NewJSONStore creates a new JSON-based task store
func NewJSONStore(filePath string) (*JSONStore, error) {
	store := &JSONStore{
		filePath: filePath,
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
	return s.save()
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

	return json.Unmarshal(file, s.data)
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
