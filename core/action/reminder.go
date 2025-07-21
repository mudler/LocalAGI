package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/db"
	models "github.com/mudler/LocalAGI/dbmodels"
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
	ID string `json:"id"`
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

	// Validate that one-time reminders don't use recurring patterns
	if !result.IsRecurring && strings.Contains(result.CronExpr, "*/") {
		return types.ActionResult{}, fmt.Errorf("one-time reminders should use absolute time values, not recurring patterns like '*/X'. Calculate the exact target time instead")
	}

	// Calculate next run time
	now := time.Now()
	schedule, _ := parser.Parse(result.CronExpr) // We can ignore the error since we validated above
	nextRun := schedule.Next(now)

	// Create database reminder record
	dbReminder := models.Reminder{
		UserID:      sharedState.UserID,
		AgentID:     sharedState.AgentID,
		Message:     result.Message,
		CronExpr:    result.CronExpr,
		LastRun:     &now,
		NextRun:     nextRun,
		IsRecurring: result.IsRecurring,
		Active:      true,
	}

	// Save reminder to database
	if err := db.DB.Create(&dbReminder).Error; err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to save reminder to database: %w", err)
	}

	return types.ActionResult{
		Result: fmt.Sprintf("Reminder set successfully with ID: %s", dbReminder.ID.String()),
		Metadata: map[string]interface{}{
			"reminder_id": dbReminder.ID.String(),
			"reminder":    result,
		},
	}, nil
}

func (a *ListRemindersAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var dbReminders []models.Reminder
	err := db.DB.Where("UserID = ? AND AgentID = ? AND Active = ?", sharedState.UserID, sharedState.AgentID, true).
		Order("NextRun ASC").Find(&dbReminders).Error
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to load reminders from database: %w", err)
	}

	if len(dbReminders) == 0 {
		return types.ActionResult{
			Result: "No active reminders found",
		}, nil
	}

	var result strings.Builder
	result.WriteString("Current reminders:\n")
	for i, reminder := range dbReminders {
		status := "one-time"
		if reminder.IsRecurring {
			status = "recurring"
		}
		result.WriteString(fmt.Sprintf("%d. %s (ID: %s, Next run: %s, Status: %s)\n",
			i+1,
			reminder.Message,
			reminder.ID.String(),
			reminder.NextRun.Format(time.RFC3339),
			status))
	}

	// Convert DB reminders to response format for metadata
	reminderResponses := make([]types.ReminderActionResponse, len(dbReminders))
	for i, dbReminder := range dbReminders {
		lastRun := time.Time{}
		if dbReminder.LastRun != nil {
			lastRun = *dbReminder.LastRun
		}
		reminderResponses[i] = types.ReminderActionResponse{
			Message:     dbReminder.Message,
			CronExpr:    dbReminder.CronExpr,
			LastRun:     lastRun,
			NextRun:     dbReminder.NextRun,
			IsRecurring: dbReminder.IsRecurring,
		}
	}

	return types.ActionResult{
		Result: result.String(),
		Metadata: map[string]interface{}{
			"reminders": reminderResponses,
		},
	}, nil
}

func (a *RemoveReminderAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var removeParams RemoveReminderParams
	err := params.Unmarshal(&removeParams)
	if err != nil {
		return types.ActionResult{}, err
	}

	if removeParams.ID == "" {
		return types.ActionResult{}, fmt.Errorf("reminder ID cannot be empty")
	}

	// Find the reminder in database
	var dbReminder models.Reminder
	err = db.DB.Where("ID = ? AND UserID = ? AND AgentID = ? AND Active = ?",
		removeParams.ID, sharedState.UserID, sharedState.AgentID, true).First(&dbReminder).Error
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("reminder not found or already removed")
	}

	// Mark as inactive instead of deleting
	dbReminder.Active = false
	if err := db.DB.Save(&dbReminder).Error; err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to remove reminder: %w", err)
	}

	return types.ActionResult{
		Result: fmt.Sprintf("Removed reminder: %s", dbReminder.Message),
		Metadata: map[string]interface{}{
			"removed_reminder_id": dbReminder.ID.String(),
			"removed_reminder": types.ReminderActionResponse{
				Message:     dbReminder.Message,
				CronExpr:    dbReminder.CronExpr,
				LastRun:     *dbReminder.LastRun,
				NextRun:     dbReminder.NextRun,
				IsRecurring: dbReminder.IsRecurring,
			},
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
		Description: "Set a reminder for the agent to wake up and perform a task based on a cron schedule. Uses 6-field format: 'second minute hour day month weekday'. For one-time reminders (like 'in 5 minutes'), calculate exact future time. For recurring reminders (like 'every 5 minutes'), use pattern syntax.",
		Properties: map[string]jsonschema.Definition{
			"message": {
				Type:        jsonschema.String,
				Description: "The message or task to be reminded about",
			},
			"cron_expr": {
				Type:        jsonschema.String,
				Description: "Cron expression for scheduling using 6-field format: 'second minute hour day month weekday'. FOR ONE-TIME REMINDERS: Calculate the exact future time and use absolute values (e.g., if current time is 14:30:15 and user wants 'in 5 minutes', use '15 35 14 * * *'). FOR RECURRING REMINDERS: Use patterns with asterisks (e.g., '0 */5 * * * *' for every 5 minutes). NEVER use patterns like '*/X' for one-time reminders - they create recurring schedules.",
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
		Description: "List all currently active reminders with their next scheduled run times and unique IDs",
		Properties:  map[string]jsonschema.Definition{},
		Required:    []string{},
	}
}

func (a *RemoveReminderAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        RemoveReminderName,
		Description: "Remove a reminder by its unique ID (use list_reminders to see the IDs)",
		Properties: map[string]jsonschema.Definition{
			"id": {
				Type:        jsonschema.String,
				Description: "The unique ID of the reminder to remove",
			},
		},
		Required: []string{"id"},
	}
}
