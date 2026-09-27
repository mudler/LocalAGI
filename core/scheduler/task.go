package scheduler

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var dayPattern = regexp.MustCompile(`^(\d+)d(.*)$`)

// ParseDuration extends time.ParseDuration with support for days ("d").
// Examples: "1d" = 24h, "2d12h" = 60h, "30m", "2h30m".
func ParseDuration(s string) (time.Duration, error) {
	if m := dayPattern.FindStringSubmatch(s); m != nil {
		days, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		d := time.Duration(days) * 24 * time.Hour
		if m[2] != "" {
			rest, err := time.ParseDuration(m[2])
			if err != nil {
				return 0, fmt.Errorf("invalid duration: %w", err)
			}
			d += rest
		}
		return d, nil
	}
	return time.ParseDuration(s)
}

type TaskStatus string

const (
	TaskStatusActive TaskStatus = "active"
	TaskStatusPaused TaskStatus = "paused"
)

type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"
	ScheduleTypeInterval ScheduleType = "interval"
	ScheduleTypeOnce     ScheduleType = "once"
)

// Task represents a scheduled task
type Task struct {
	ID            string                 `json:"id"`
	AgentName     string                 `json:"agent_name"`
	Prompt        string                 `json:"prompt"`
	ScheduleType  ScheduleType           `json:"schedule_type"`
	ScheduleValue string                 `json:"schedule_value"`
	Status        TaskStatus             `json:"status"`
	NextRun       time.Time              `json:"next_run"`
	LastRun       *time.Time             `json:"last_run,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ContextMode   string                 `json:"context_mode"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// TaskRun represents a single execution of a task
type TaskRun struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"task_id"`
	RunAt      time.Time `json:"run_at"`
	DurationMs int64     `json:"duration_ms"`
	Status     string    `json:"status"` // "success", "error", "timeout"
	Result     string    `json:"result,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// NewTask creates a new task with the given parameters
func NewTask(agentName, prompt string, scheduleType ScheduleType, scheduleValue string) (*Task, error) {
	task := &Task{
		ID:            uuid.New().String(),
		AgentName:     agentName,
		Prompt:        prompt,
		ScheduleType:  scheduleType,
		ScheduleValue: scheduleValue,
		Status:        TaskStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ContextMode:   "agent",
		Metadata:      make(map[string]interface{}),
	}

	if err := task.CalculateNextRun(); err != nil {
		return nil, err
	}

	return task, nil
}

// CalculateNextRun calculates the next run time based on schedule type
func (t *Task) CalculateNextRun() error {
	now := time.Now()

	switch t.ScheduleType {
	case ScheduleTypeCron:
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(t.ScheduleValue)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		t.NextRun = schedule.Next(now)

	case ScheduleTypeInterval:
		intervalMs, err := strconv.ParseInt(t.ScheduleValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		if intervalMs <= 0 {
			return fmt.Errorf("invalid interval: %d", intervalMs)
		}
		if t.LastRun != nil {
			t.NextRun = t.LastRun.Add(time.Duration(intervalMs) * time.Millisecond)
		} else {
			t.NextRun = now.Add(time.Duration(intervalMs) * time.Millisecond)
		}

	case ScheduleTypeOnce:
		duration, err := ParseDuration(t.ScheduleValue)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		if duration < 0 {
			return fmt.Errorf("duration must be positive: %s", t.ScheduleValue)
		}
		t.NextRun = now.Add(duration)

	default:
		return fmt.Errorf("unknown schedule type: %s", t.ScheduleType)
	}

	return nil
}

// IsDue checks if the task should be executed now
func (t *Task) IsDue() bool {
	return t.Status == TaskStatusActive && time.Now().After(t.NextRun)
}

// NewTaskRun creates a new task run record
func NewTaskRun(taskID string) *TaskRun {
	return &TaskRun{
		ID:     uuid.New().String(),
		TaskID: taskID,
		RunAt:  time.Now(),
	}
}
