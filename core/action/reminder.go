package action

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	// Calculate next run time
	now := time.Now()
	schedule, _ := parser.Parse(result.CronExpr) // We can ignore the error since we validated above
	nextRun := schedule.Next(now)

	// Set the reminder details
	result.LastRun = now
	result.NextRun = nextRun
	// IsRecurring is set by the user through the action parameters

	// Store the reminder in the shared state
	if sharedState.Reminders == nil {
		sharedState.Reminders = make([]types.ReminderActionResponse, 0)
	}
	sharedState.Reminders = append(sharedState.Reminders, result)

	return types.ActionResult{
		Result: "Reminder set successfully",
		Metadata: map[string]interface{}{
			"reminder": result,
		},
	}, nil
}

func (a *ListRemindersAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	if sharedState.Reminders == nil || len(sharedState.Reminders) == 0 {
		return types.ActionResult{
			Result: "No reminders set",
		}, nil
	}

	var result strings.Builder
	result.WriteString("Current reminders:\n")
	for i, reminder := range sharedState.Reminders {
		status := "one-time"
		if reminder.IsRecurring {
			status = "recurring"
		}
		result.WriteString(fmt.Sprintf("%d. %s (Next run: %s, Status: %s)\n",
			i+1,
			reminder.Message,
			reminder.NextRun.Format(time.RFC3339),
			status))
	}

	return types.ActionResult{
		Result: result.String(),
		Metadata: map[string]interface{}{
			"reminders": sharedState.Reminders,
		},
	}, nil
}

func (a *RemoveReminderAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var removeParams RemoveReminderParams
	err := params.Unmarshal(&removeParams)
	if err != nil {
		return types.ActionResult{}, err
	}

	if sharedState.Reminders == nil || len(sharedState.Reminders) == 0 {
		return types.ActionResult{
			Result: "No reminders to remove",
		}, nil
	}

	// Convert from 1-based index to 0-based
	index := removeParams.Index - 1
	if index < 0 || index >= len(sharedState.Reminders) {
		return types.ActionResult{}, fmt.Errorf("invalid reminder index: %d", removeParams.Index)
	}

	// Remove the reminder
	removed := sharedState.Reminders[index]
	sharedState.Reminders = append(sharedState.Reminders[:index], sharedState.Reminders[index+1:]...)

	return types.ActionResult{
		Result: fmt.Sprintf("Removed reminder: %s", removed.Message),
		Metadata: map[string]interface{}{
			"removed_reminder": removed,
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
		Description: "Set a reminder for the agent to wake up and perform a task based on a cron schedule. Uses 6-field format: 'second minute hour day month weekday'. Examples: '0 0 0 * * *' (daily at midnight), '0 0 */2 * * *' (every 2 hours), '0 0 0 * * 1' (every Monday at midnight), '0 */5 * * * *' (every 5 minutes)",
		Properties: map[string]jsonschema.Definition{
			"message": {
				Type:        jsonschema.String,
				Description: "The message or task to be reminded about",
			},
			"cron_expr": {
				Type:        jsonschema.String,
				Description: "Cron expression for scheduling using 6-field format: 'second minute hour day month weekday'. Examples: '0 0 0 * * *' (daily at midnight), '0 */10 * * * *' (every 10 minutes), '0 30 14 * * *' (daily at 2:30 PM). For delays like 'in 2 minutes', calculate the target time and use absolute values.",
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
