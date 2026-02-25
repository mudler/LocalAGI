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
	// Render the scheduler task template - if custom template is set, it will include {{.Task}}
	// If no custom scheduler template is set, fall back to default inner monologue template
	innerMonologue := fmt.Sprintf("You need to execute the following task, by using the tools available to you. When the task is completed, you need to send a message to the user with send_message tool to inform them that the task is completed: %s", prompt)

	if e.agent.options.schedulerTaskTemplate != "" {
		tmpl, err := templateBase("taskTemplate", e.agent.options.schedulerTaskTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to render scheduler task template: %w", err)
		}

		innerMonologue, err = templateExecute(tmpl, struct {
			Task string
		}{
			Task: prompt,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to render scheduler task template: %w", err)
		}
	}

	// Create a job for the reminder with the rendered inner monologue
	reminderJob := types.NewJob(
		types.WithText(innerMonologue),
		types.WithReasoningCallback(e.agent.options.reasoningCallback),
		types.WithResultCallback(e.agent.options.resultCallback),
		types.WithContext(ctx),
		types.WithMetadata(map[string]any{
			"message":     prompt,
			"is_reminder": true,
			"type":        "scheduled",
		}),
	)

	// Attach observable so UI can show reminder processing state
	if e.agent.observer != nil {
		obs := e.agent.observer.NewObservable()
		obs.Name = "reminder"
		obs.Icon = "bell"
		e.agent.observer.Update(*obs)
		reminderJob.Obs = obs
	}

	// Send the job to be processed
	e.agent.jobQueue <- reminderJob

	// Wait for the job to complete or context to be cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		result, err := reminderJob.Result.WaitResult(ctx)
		if err != nil {
			return nil, err
		}
		if result.Error != nil {
			return &scheduler.JobResult{
				Response: "",
				Error:    result.Error,
			}, result.Error
		}
		return &scheduler.JobResult{
			Response: result.Response,
			Error:    nil,
		}, nil
	}
}
