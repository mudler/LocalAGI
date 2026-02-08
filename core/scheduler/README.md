# Task Scheduler - Integrated with LocalAGI Reminder System

A comprehensive task scheduling system integrated into LocalAGI's existing reminder functionality, enabling agents to execute tasks on cron schedules, intervals, or one-time execution.

## Overview

The task scheduler is automatically initialized when an agent is created and manages all reminder functionality. It provides:

- **Persistent Storage**: Reminders survive agent restarts (stored in JSON)
- **Multiple Schedule Types**: Cron, interval, and once schedules
- **Execution History**: Full tracking of task runs with duration and status
- **Interface-Based Design**: Easy to extend with different storage backends

## Integration

The scheduler is transparently integrated into LocalAGI's reminder system:

1. **Automatic Initialization**: Created when agent starts
2. **Existing Actions**: `set_reminder`, `list_reminders`, `remove_reminder` use the scheduler
3. **Backward Compatible**: Falls back to in-memory storage if needed
4. **Persistent**: Tasks stored in `data/scheduled_tasks.json` by default

## How to Use (Agent Actions)

### Setting Reminders

Agents can set reminders using the `set_reminder` action:

```python
# Through agent conversation
User: "Remind me to check emails every 5 minutes"
Agent: *uses set_reminder action*
  {
    "message": "check emails",
    "cron_expr": "0 */5 * * * *",  # Every 5 minutes
    "is_recurring": true
  }
```

### Listing Reminders

```python
User: "What reminders do I have?"
Agent: *uses list_reminders action*
# Returns:
# 1. check emails (Next run: 2026-02-08T10:00:00Z, Status: recurring, ID: abc-123)
# 2. meeting reminder (Next run: 2026-02-08T14:00:00Z, Status: one-time, ID: def-456)
```

### Removing Reminders

```python
User: "Remove the first reminder"
Agent: *uses remove_reminder action with index: 1*
# Removes: "check emails" reminder
```

## Programmatic Usage (Go)

For direct programmatic access to the scheduler (advanced use):

```go
import (
    "github.com/mudler/LocalAGI/core/agent"
    "github.com/mudler/LocalAGI/core/scheduler"
)

// Create agent with custom scheduler path
agent, err := agent.New(
    agent.WithSchedulerStorePath("custom/path/tasks.json"),
    // ... other options
)

// Access scheduler through agent
// (scheduler is started automatically when agent.Run() is called)
```

### Manual Scheduler Usage

If you need direct control (not typical):

#### Create and Start Scheduler

```go
store, _ := scheduler.NewJSONStore("tasks.json")
executor := &MyExecutor{}
sched := scheduler.NewScheduler(store, executor, time.Minute)
sched.Start()
defer sched.Stop()
```

#### Create Tasks

### Pause a Task
```go
err := sched.PauseTask(taskID)
```

### Resume a Task
```go
err := sched.ResumeTask(taskID)
```

### Delete a Task
```go
err := sched.DeleteTask(taskID)
```

### Get Task Status
```go
task, err := sched.GetTask(taskID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Task status: %s\n", task.Status)
fmt.Printf("Next run: %s\n", task.NextRun)
```go
task, err := scheduler.NewTask(
    "agent-name",
    "Check for new emails",
    scheduler.ScheduleTypeCron,
    "0 0 0 * * *", // Daily at midnight (6 fields: second minute hour day month day-of-week)
)
sched.CreateTask(task)
```

**Interval Task** (Every 5 minutes):
```go
task, _ := scheduler.NewTask(
    "agent-name",
    "Monitor health",
    scheduler.ScheduleTypeInterval,
    "300000", // 300,000 milliseconds
)
sched.CreateTask(task)
```

**One-Time Task**:
```go
task, _ := scheduler.NewTask(
    "agent-name",
    "Send reminder",
    scheduler.ScheduleTypeOnce,
    "2026-12-25T09:00:00Z", // RFC3339 format
)
sched.CreateTask(task)
```

#### Task Management

```go
// Pause/Resume
sched.PauseTask(taskID)
sched.ResumeTask(taskID)

// Delete
sched.DeleteTask(taskID)

// Query
task, _ := sched.GetTask(taskID)
tasks, _ := sched.GetAllTasks()
runs, _ := sched.GetTaskRuns(taskID, 10) // Last 10 runs
```

## Schedule Types

The scheduler uses 6-field cron expressions with the following format:

```
┌───────────── second (0-59)
│ ┌───────────── minute (0-59)
│ │ ┌───────────── hour (0-23)
│ │ │ ┌───────────── day of month (1-31)
│ │ │ │ ┌───────────── month (1-12)
│ │ │ │ │ ┌───────────── day of week (0-6, Sunday=0)
│ │ │ │ │ │
* * * * * *
```

### Examples

- `0 0 0 * * *` - Every day at midnight
- `0 */15 * * * *` - Every 15 minutes
- `0 0 9 * * 1-5` - Every weekday at 9 AM
- `0 30 14 * * *` - Every day at 2:30 PM
- `0 0 0 1 * *` - First day of every month at midnight

## Architecture

### Interfaces

The scheduler is built around two main interfaces:

- **TaskStore**: Handles task persistence (default: JSON file storage)
- **AgentExecutor**: Executes agent tasks

This design makes it easy to implement alternative storage backends (SQLite, PostgreSQL, etc.) or different execution strategies.

### Thread Safety

All store operations are protected by read-write mutexes to ensure thread-safe concurrent access.

### Graceful Shutdown

The scheduler properly handles graceful shutdown:
- Stops accepting new tasks
- Waits for running tasks to complete
- Closes the store

## Testing

Run the comprehensive test suite:

```bash
go test ./core/scheduler/...
```

Or with Ginkgo:

```bash
go run github.com/onsi/ginkgo/v2/ginkgo -v ./core/scheduler/...
```

## Future Enhancements

The interface-based design allows for easy extensions:

- [ ] Database backends (SQLite, PostgreSQL)
- [ ] REST API endpoints
- [ ] Web UI integration
- [ ] Task dependencies and chaining
- [ ] Priority-based execution
- [ ] Retry policies with exponential backoff
- [ ] Notification webhooks
- [ ] Task groups and bulk operations

## Contributing

Contributions are welcome! Please ensure all tests pass before submitting a PR.

## License

Part of the LocalAGI project.
