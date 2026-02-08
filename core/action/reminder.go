package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/scheduler"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/robfig/cron/v3"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	ReminderActionName = "set_reminder"
	ListRemindersName  = "list_reminders"
	RemoveReminderName = "remove_reminder"
)

func NewReminder() *ReminderAction {
	return &ReminderAction{}
}

func NewListReminders() *ListRemindersAction {
	return &ListRemindersAction{}
}

func NewRemoveReminder() *RemoveReminderAction {
	return &RemoveReminderAction{}
}

type ReminderAction struct{}
type ListRemindersAction struct{}
type RemoveReminderAction struct{}

type RemoveReminderParams struct {
	Index int `json:"index"`
}

func (a *ReminderAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := types.ReminderActionResponse{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, err
	}

	// Validate the cron expression
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err = parser.Parse(result.CronExpr)
	if err != nil {
		return types.ActionResult{}, err
	}

	// Create a scheduler task
	scheduleType := scheduler.ScheduleTypeCron
	if !result.IsRecurring {
		scheduleType = scheduler.ScheduleTypeOnce
	}

	task, err := scheduler.NewTask(
		"reminder", // agent name (not used for reminders, but required)
		result.Message,
		scheduleType,
		result.CronExpr,
	)
	if err != nil {
		return types.ActionResult{}, err
	}

	// Store task ID in metadata for tracking
	task.Metadata["reminder_type"] = "user_created"

	err = sharedState.Scheduler.CreateTask(task)
	if err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{
		Result: fmt.Sprintf("Reminder set successfully (ID: %s)", task.ID),
		Metadata: map[string]interface{}{
			"task_id":   task.ID,
			"message":   result.Message,
			"next_run":  task.NextRun,
			"recurring": result.IsRecurring,
		},
	}, nil

}

func (a *ListRemindersAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	tasksInterface, err := sharedState.Scheduler.GetAllTasks()
	if err != nil {
		return types.ActionResult{}, err
	}

	if len(tasksInterface) == 0 {
		return types.ActionResult{
			Result: "No reminders set",
		}, nil
	}

	var result strings.Builder
	result.WriteString("Current reminders:\n")

	for i, taskInterface := range tasksInterface {
		task, ok := taskInterface.(*scheduler.Task)
		if !ok {
			continue
		}

		status := "one-time"
		if task.ScheduleType == scheduler.ScheduleTypeCron || task.ScheduleType == scheduler.ScheduleTypeInterval {
			status = "recurring"
		}

		result.WriteString(fmt.Sprintf("%d. %s (Next run: %s, Status: %s, ID: %s)\n",
			i+1,
			task.Prompt,
			task.NextRun.Format(time.RFC3339),
			status,
			task.ID))
	}

	return types.ActionResult{
		Result: result.String(),
		Metadata: map[string]interface{}{
			"tasks": tasksInterface,
		},
	}, nil
}

func (a *RemoveReminderAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var removeParams RemoveReminderParams
	err := params.Unmarshal(&removeParams)
	if err != nil {
		return types.ActionResult{}, err
	}

	tasksInterface, err := sharedState.Scheduler.GetAllTasks()
	if err != nil {
		return types.ActionResult{}, err
	}

	if len(tasksInterface) == 0 {
		return types.ActionResult{
			Result: "No reminders to remove",
		}, nil
	}

	// Convert from 1-based index to 0-based
	index := removeParams.Index - 1
	if index < 0 || index >= len(tasksInterface) {
		return types.ActionResult{}, fmt.Errorf("invalid reminder index: %d", removeParams.Index)
	}

	task, ok := tasksInterface[index].(*scheduler.Task)
	if !ok {
		return types.ActionResult{}, fmt.Errorf("invalid task type")
	}

	err = sharedState.Scheduler.DeleteTask(task.ID)
	if err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{
		Result: fmt.Sprintf("Removed reminder: %s", task.Prompt),
		Metadata: map[string]interface{}{
			"removed_task_id": task.ID,
		},
	}, nil
}

func (a *ReminderAction) Plannable() bool {
	return true
}

func (a *ListRemindersAction) Plannable() bool {
	return true
}

func (a *RemoveReminderAction) Plannable() bool {
	return true
}

func (a *ReminderAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        ReminderActionName,
		Description: "Set a reminder for the agent to wake up and perform a task based on a cron schedule. Examples: '0 0 * * *' (daily at midnight), '0 */2 * * *' (every 2 hours), '0 0 * * 1' (every Monday at midnight)",
		Properties: map[string]jsonschema.Definition{
			"message": {
				Type:        jsonschema.String,
				Description: "The message or task to be reminded about",
			},
			"cron_expr": {
				Type:        jsonschema.String,
				Description: "Cron expression for scheduling (e.g. '0 0 * * *' for daily at midnight). Format: 'second minute hour day month weekday'",
			},
			"is_recurring": {
				Type:        jsonschema.Boolean,
				Description: "Whether this reminder should repeat according to the cron schedule (true) or trigger only once (false)",
			},
		},
		Required: []string{"message", "cron_expr", "is_recurring"},
	}
}

func (a *ListRemindersAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        ListRemindersName,
		Description: "List all currently set reminders with their next scheduled run times",
		Properties:  map[string]jsonschema.Definition{},
		Required:    []string{},
	}
}

func (a *RemoveReminderAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        RemoveReminderName,
		Description: "Remove a reminder by its index number (use list_reminders to see the index)",
		Properties: map[string]jsonschema.Definition{
			"index": {
				Type:        jsonschema.Integer,
				Description: "The index number of the reminder to remove (1-based)",
			},
		},
		Required: []string{"index"},
	}
}
