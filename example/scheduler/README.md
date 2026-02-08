# Task Scheduler Example

This example demonstrates how to use the LocalAGI task scheduler to run agents on cron schedules, intervals, or one-time execution.

## Running the Example

```bash
cd example/scheduler
go run main.go
```

## What This Example Does

1. Creates a JSON-based task store
2. Starts the scheduler with a 5-second poll interval
3. Creates three different types of tasks:
   - **Cron Task**: Runs every minute
   - **Interval Task**: Runs every 30 seconds
   - **One-Time Task**: Runs once, 10 seconds after start
4. Lists all scheduled tasks
5. Runs until you press Ctrl+C
6. Shows execution history before exit

## Sample Output

```
Task Scheduler Started!
Creating example tasks...
✓ Created cron task (ID: abc-123) - runs every minute
✓ Created interval task (ID: def-456) - runs every 30 seconds
✓ Created one-time task (ID: ghi-789) - runs at 14:30:45

All scheduled tasks:
  - cron-agent (cron): Next run at 14:31:00
  - interval-agent (interval): Next run at 14:30:50
  - once-agent (once): Next run at 14:30:45

Scheduler is running. Press Ctrl+C to stop...
Watch the logs to see tasks being executed.

[14:30:45] Executing agent 'once-agent' with prompt: Send reminder
[14:30:50] Executing agent 'interval-agent' with prompt: Monitor system health
[14:31:00] Executing agent 'cron-agent' with prompt: Check for updates
```

## Files Created

- `example_tasks.json` - Persistent storage of scheduled tasks

## Next Steps

Try modifying the example to:
- Add more tasks with different schedules
- Implement a real agent executor
- Pause/resume tasks while running
- Query execution history
