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
	var innerMonologue string
	var err error

	if e.agent.options.schedulerTaskTemplate != "" {
		innerMonologue, err = RenderInnerMonologueTemplate(e.agent.options.schedulerTaskTemplate, prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to render scheduler task template: %w", err)
		}
	} else {
		// Use default inner monologue template with task injected
		innerMonologue, err = RenderInnerMonologueTemplate("", prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to render inner monologue template: %w", err)
		}
	}

	// Create a job for the reminder with the rendered inner monologue
	reminderJob := types.NewJob(
		types.WithText(fmt.Sprintf("%s\n\nTask: %s", innerMonologue, prompt)),
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
