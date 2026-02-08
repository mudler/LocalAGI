package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mudler/xlog"
)

// Scheduler manages scheduled tasks
type Scheduler struct {
	store        TaskStore
	executor     AgentExecutor
	pollInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	runningTasks map[string]context.CancelFunc
}

// NewScheduler creates a new scheduler with the given store and executor
func NewScheduler(store TaskStore, executor AgentExecutor, pollInterval time.Duration) *Scheduler {

	return &Scheduler{
		store:        store,
		executor:     executor,
		pollInterval: pollInterval,
		runningTasks: make(map[string]context.CancelFunc),
	}
}

// Start begins the scheduler's polling loop
func (s *Scheduler) Start() {
	if s.ctx != nil {
		xlog.Warn("Scheduler already started")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	s.wg.Add(1)
	go s.run()
	xlog.Info("Task scheduler started", "poll_interval", s.pollInterval)
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.store.Close()
	xlog.Info("Task scheduler stopped")
	s.cancel = nil
	s.ctx = nil
}

// run is the main polling loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processDueTasks()
		}
	}
}

// processDueTasks checks for and executes due tasks
func (s *Scheduler) processDueTasks() {
	tasks, err := s.store.GetDue()
	if err != nil {
		xlog.Error("Failed to get due tasks", "error", err)
		return
	}

	if len(tasks) > 0 {
		xlog.Debug("Processing due tasks", "count", len(tasks))
	}

	for _, task := range tasks {
		// Check if task is already running
		s.mu.RLock()
		_, running := s.runningTasks[task.ID]
		s.mu.RUnlock()

		if running {
			xlog.Warn("Task already running, skipping", "task_id", task.ID)
			continue
		}

		// Execute task in goroutine
		s.wg.Add(1)
		go s.executeTask(task)
	}
}

// executeTask runs a single task
func (s *Scheduler) executeTask(task *Task) {
	defer s.wg.Done()

	taskCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// Register running task
	s.mu.Lock()
	s.runningTasks[task.ID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningTasks, task.ID)
		s.mu.Unlock()
	}()

	xlog.Info("Executing task", "task_id", task.ID, "agent", task.AgentName, "prompt", task.Prompt)

	startTime := time.Now()
	run := NewTaskRun(task.ID)

	// Execute the task
	result, err := s.executor.Execute(taskCtx, task.AgentName, task.Prompt)

	run.DurationMs = time.Since(startTime).Milliseconds()

	if err != nil {
		run.Status = "error"
		run.Error = err.Error()
		xlog.Error("Task execution failed", "task_id", task.ID, "error", err)
	} else {
		run.Status = "success"
		if result != nil {
			run.Result = result.Response
		}
		xlog.Info("Task executed successfully", "task_id", task.ID, "duration_ms", run.DurationMs)
	}

	// Log the run
	if err := s.store.LogRun(run); err != nil {
		xlog.Error("Failed to log task run", "task_id", task.ID, "error", err)
	}

	// Update task for next run
	now := time.Now()
	task.LastRun = &now

	// For one-time tasks, mark as deleted
	if task.ScheduleType == ScheduleTypeOnce {
		if err := s.store.Delete(task.ID); err != nil {
			xlog.Error("Failed to delete task", "task_id", task.ID, "error", err)
		}
	} else {
		// Calculate next run
		if err := task.CalculateNextRun(); err != nil {
			xlog.Error("Failed to calculate next run", "task_id", task.ID, "error", err)
			task.Status = TaskStatusPaused
		}
	}

	if err := s.store.Update(task); err != nil {
		xlog.Error("Failed to update task", "task_id", task.ID, "error", err)
	}
}

// CRUD operations

// CreateTask adds a new task
func (s *Scheduler) CreateTask(task *Task) error {
	return s.store.Create(task)
}

// GetTask retrieves a task by ID
func (s *Scheduler) GetTask(id string) (*Task, error) {
	return s.store.Get(id)
}

// GetAllTasks retrieves all tasks
func (s *Scheduler) GetAllTasks() ([]*Task, error) {
	return s.store.GetAll()
}

// GetTasksByAgent retrieves all tasks for a specific agent
func (s *Scheduler) GetTasksByAgent(agentName string) ([]*Task, error) {
	return s.store.GetByAgent(agentName)
}

// UpdateTask updates an existing task
func (s *Scheduler) UpdateTask(task *Task) error {
	return s.store.Update(task)
}

// DeleteTask removes a task
func (s *Scheduler) DeleteTask(id string) error {
	return s.store.Delete(id)
}

// GetTaskRuns retrieves execution history for a task
func (s *Scheduler) GetTaskRuns(taskID string, limit int) ([]*TaskRun, error) {
	return s.store.GetRuns(taskID, limit)
}

// PauseTask pauses a task
func (s *Scheduler) PauseTask(id string) error {
	task, err := s.store.Get(id)
	if err != nil {
		return err
	}

	task.Status = TaskStatusPaused
	return s.store.Update(task)
}

// ResumeTask resumes a paused task
func (s *Scheduler) ResumeTask(id string) error {
	task, err := s.store.Get(id)
	if err != nil {
		return err
	}

	task.Status = TaskStatusActive
	if err := task.CalculateNextRun(); err != nil {
		return err
	}

	return s.store.Update(task)
}

// CancelRunningTask cancels a currently running task
func (s *Scheduler) CancelRunningTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancel, exists := s.runningTasks[id]
	if !exists {
		return fmt.Errorf("task not running: %s", id)
	}

	cancel()
	return nil
}
