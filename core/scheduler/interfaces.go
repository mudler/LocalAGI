package scheduler

import (
	"context"
)

// TaskStore defines the interface for task persistence
type TaskStore interface {
	// Create adds a new task
	Create(task *Task) error

	// Get retrieves a task by ID
	Get(id string) (*Task, error)

	// GetAll retrieves all tasks
	GetAll() ([]*Task, error)

	// GetDue retrieves tasks that are due for execution
	GetDue() ([]*Task, error)

	// GetByAgent retrieves all tasks for a specific agent
	GetByAgent(agentName string) ([]*Task, error)

	// Update updates an existing task
	Update(task *Task) error

	// Delete removes a task
	Delete(id string) error

	// LogRun records a task execution
	LogRun(run *TaskRun) error

	// GetRuns retrieves execution history for a task
	GetRuns(taskID string, limit int) ([]*TaskRun, error)

	// Close releases resources
	Close() error
}

// AgentExecutor defines the interface for executing agent tasks
type AgentExecutor interface {
	Execute(ctx context.Context, agentName string, prompt string) (*JobResult, error)
}

// JobResult represents the result of an agent execution
type JobResult struct {
	Response string
	Error    error
}
