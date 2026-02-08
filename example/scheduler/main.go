package main

import (
"context"
"fmt"
"log"
"os"
"os/signal"
"syscall"
"time"

"github.com/mudler/LocalAGI/core/scheduler"
)

// ExampleExecutor demonstrates a simple agent executor implementation
type ExampleExecutor struct{}

func (e *ExampleExecutor) Execute(ctx context.Context, agentName string, prompt string) (*scheduler.JobResult, error) {
fmt.Printf("[%s] Executing agent '%s' with prompt: %s\n", time.Now().Format("15:04:05"), agentName, prompt)

// Simulate some work
time.Sleep(100 * time.Millisecond)

return &scheduler.JobResult{
Response: fmt.Sprintf("Completed task for %s", agentName),
Error:    nil,
}, nil
}

func main() {
// Create store
store, err := scheduler.NewJSONStore("example_tasks.json")
if err != nil {
log.Fatalf("Failed to create store: %v", err)
}
defer store.Close()

// Create executor
executor := &ExampleExecutor{}

// Create scheduler with 5-second poll interval
sched := scheduler.NewScheduler(store, executor, 5*time.Second)
sched.Start()
defer sched.Stop()

fmt.Println("Task Scheduler Started!")
fmt.Println("Creating example tasks...")

// Create a cron task - runs every minute
cronTask, err := scheduler.NewTask(
"cron-agent",
"Check for updates",
scheduler.ScheduleTypeCron,
"0 * * * * *", // Every minute at 0 seconds
)
if err != nil {
log.Fatalf("Failed to create cron task: %v", err)
}
if err := sched.CreateTask(cronTask); err != nil {
log.Fatalf("Failed to add cron task: %v", err)
}
fmt.Printf("✓ Created cron task (ID: %s) - runs every minute\n", cronTask.ID)

// Create an interval task - runs every 30 seconds
intervalTask, err := scheduler.NewTask(
"interval-agent",
"Monitor system health",
scheduler.ScheduleTypeInterval,
"30000", // 30 seconds
)
if err != nil {
log.Fatalf("Failed to create interval task: %v", err)
}
if err := sched.CreateTask(intervalTask); err != nil {
log.Fatalf("Failed to add interval task: %v", err)
}
fmt.Printf("✓ Created interval task (ID: %s) - runs every 30 seconds\n", intervalTask.ID)

// Create a one-time task - runs 10 seconds from now
futureTime := time.Now().Add(10 * time.Second)
onceTask, err := scheduler.NewTask(
"once-agent",
"Send reminder",
scheduler.ScheduleTypeOnce,
futureTime.Format(time.RFC3339),
)
if err != nil {
log.Fatalf("Failed to create once task: %v", err)
}
if err := sched.CreateTask(onceTask); err != nil {
log.Fatalf("Failed to add once task: %v", err)
}
fmt.Printf("✓ Created one-time task (ID: %s) - runs at %s\n", onceTask.ID, futureTime.Format("15:04:05"))

// List all tasks
fmt.Println("\nAll scheduled tasks:")
tasks, _ := sched.GetAllTasks()
for _, task := range tasks {
fmt.Printf("  - %s (%s): Next run at %s\n", 
task.AgentName, task.ScheduleType, task.NextRun.Format("15:04:05"))
}

fmt.Println("\nScheduler is running. Press Ctrl+C to stop...")
fmt.Println("Watch the logs to see tasks being executed.")

// Wait for interrupt signal
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

fmt.Println("\n\nShutting down...")

// Show execution history before exit
fmt.Println("\nTask execution history:")
for _, task := range tasks {
runs, _ := sched.GetTaskRuns(task.ID, 5)
if len(runs) > 0 {
fmt.Printf("\n%s (%s):\n", task.AgentName, task.ID)
for _, run := range runs {
fmt.Printf("  - %s: %s (took %dms)\n", 
run.RunAt.Format("15:04:05"), run.Status, run.DurationMs)
}
}
}

fmt.Println("\nGoodbye!")
}
