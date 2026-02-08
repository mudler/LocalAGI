# Task Scheduler

A comprehensive task scheduling system for LocalAGI, enabling agents to execute tasks on cron schedules, intervals, or one-time execution.

## Features

- ✅ **Multiple Schedule Types**:
  - **Cron**: Use standard cron expressions with seconds (e.g., `0 0 0 * * *` for daily at midnight)
  - **Interval**: Execute tasks at fixed intervals in milliseconds (e.g., `300000` for every 5 minutes)
  - **Once**: Execute tasks at a specific time (ISO 8601 timestamp)

- ✅ **JSON Storage**: Simple file-based persistence with thread-safe operations
- ✅ **Task Management**: Create, update, pause, resume, delete tasks
- ✅ **Execution Logging**: Track task runs with duration, status, and results
- ✅ **Interface-Based Design**: Easy to extend with different storage backends

## Installation

The scheduler is part of the LocalAGI core package. Import it in your Go code:

```go
import "github.com/mudler/LocalAGI/core/scheduler"
```

## Quick Start

### 1. Create a Store

```go
store, err := scheduler.NewJSONStore("data/scheduled_tasks.json")
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

### 2. Implement AgentExecutor

```go
type MyAgentExecutor struct {
    // Your agent execution logic
}

func (e *MyAgentExecutor) Execute(ctx context.Context, agentName string, prompt string) (*scheduler.JobResult, error) {
    // Execute the agent with the given prompt
    // Return the result
    return &scheduler.JobResult{
        Response: "Task completed",
        Error:    nil,
    }, nil
}
```

### 3. Create and Start Scheduler

```go
executor := &MyAgentExecutor{}
sched := scheduler.NewScheduler(store, executor, time.Minute)
sched.Start()
defer sched.Stop()
```

### 4. Create Tasks

#### Cron Task (Daily at midnight)
```go
task, err := scheduler.NewTask(
    "my-agent",
    "Check for new emails and summarize",
    scheduler.ScheduleTypeCron,
    "0 0 0 * * *", // 6 fields: second minute hour day month day-of-week
)
if err != nil {
    log.Fatal(err)
}
sched.CreateTask(task)
```

#### Interval Task (Every 5 minutes)
```go
intervalTask, err := scheduler.NewTask(
    "monitor-agent",
    "Monitor system health",
    scheduler.ScheduleTypeInterval,
    "300000", // 300,000 milliseconds = 5 minutes
)
sched.CreateTask(intervalTask)
```

#### One-Time Task
```go
oneTimeTask, err := scheduler.NewTask(
    "reminder-agent",
    "Send meeting reminder",
    scheduler.ScheduleTypeOnce,
    "2026-12-25T09:00:00Z", // RFC3339 format
)
sched.CreateTask(oneTimeTask)
```

## Task Management

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
```

### Get Tasks by Agent
```go
tasks, err := sched.GetTasksByAgent("my-agent")
for _, task := range tasks {
    fmt.Printf("Task %s: %s\n", task.ID, task.Prompt)
}
```

### Get Task Execution History
```go
runs, err := sched.GetTaskRuns(taskID, 10) // Get last 10 runs
for _, run := range runs {
    fmt.Printf("Run at %s: %s (took %dms)\n", 
        run.RunAt, run.Status, run.DurationMs)
}
```

## Cron Expression Format

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
