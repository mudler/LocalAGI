package agent

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAGI/core/scheduler"
	"github.com/mudler/LocalAGI/core/types"
)

// agentSchedulerExecutor implements scheduler.AgentExecutor for executing scheduled tasks through the agent
type agentSchedulerExecutor struct {
	agent *Agent
}

// Execute processes a scheduled task by creating a job for the agent
func (e *agentSchedulerExecutor) Execute(ctx context.Context, agentName string, prompt string) (*scheduler.JobResult, error) {
	// Create a job for the reminder
	reminderJob := types.NewJob(
		types.WithText(fmt.Sprintf("I have a reminder for you: %s", prompt)),
		types.WithReasoningCallback(e.agent.options.reasoningCallback),
		types.WithResultCallback(e.agent.options.resultCallback),
	)

	// Add metadata to indicate this is a reminder
	reminderJob.Metadata = map[string]interface{}{
		"message":     prompt,
		"is_reminder": true,
	}

	// Attach observable so UI can show reminder processing state
	if e.agent.observer != nil {
		reminderJob.WithObserver(e.agent.observer)
	}

	// Send the job to be processed
	e.agent.jobQueue <- reminderJob

	// Wait for the job to complete or context to be cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-reminderJob.Done:
		if reminderJob.Err != nil {
			return &scheduler.JobResult{
				Response: "",
				Error:    reminderJob.Err,
			}, reminderJob.Err
		}
		return &scheduler.JobResult{
			Response: reminderJob.Result,
			Error:    nil,
		}, nil
	}
}
