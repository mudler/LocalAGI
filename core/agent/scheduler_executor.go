package agent

import (
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/xlog"
)

// SchedulerExecutor handles recurring task execution for the agent
type SchedulerExecutor struct {
	agent *Agent
}

// NewSchedulerExecutor creates a new scheduler executor for the given agent
func NewSchedulerExecutor(agent *Agent) *SchedulerExecutor {
	return &SchedulerExecutor{
		agent: agent,
	}
}

// Execute runs a scheduled task using the inner monologue template.
// The task description is injected into the template via {{.Task}}.
func (s *SchedulerExecutor) Execute(task string) *types.JobResult {
	xlog.Debug("Scheduler executing task", "task", task)

	// Render the inner monologue template with the task
	// If no custom template is set, it will use the default
	templatedText, err := RenderInnerMonologueTemplate(s.agent.options.innerMonologueTemplate, task)
	if err != nil {
		xlog.Error("Failed to render inner monologue template", "error", err)
		// Fall back to default template
		templatedText = innerMonologueTemplate + "\n\nTask: " + task
	}

	// Create a new job with the templated text
	job := types.NewJob(
		types.WithText(templatedText),
		types.WithReasoningCallback(s.agent.options.reasoningCallback),
		types.WithResultCallback(s.agent.options.resultCallback),
	)

	// Execute the job
	s.agent.Execute(job)

	return job.Result
}
