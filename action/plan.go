package action

import (
	"github.com/sashabaranov/go-openai/jsonschema"
)

// PlanActionName is the name of the plan action
// used by the LLM to schedule more actions
const PlanActionName = "plan"

func NewPlan() *PlanAction {
	return &PlanAction{}
}

type PlanAction struct{}

type PlanResult struct {
	Subtasks []PlanSubtask `json:"subtasks"`
}
type PlanSubtask struct {
	Action    string `json:"action"`
	Reasoning string `json:"reasoning"`
}

func (a *PlanAction) Run(ActionParams) (string, error) {
	return "no-op", nil
}

func (a *PlanAction) Definition() ActionDefinition {
	return ActionDefinition{
		Name:        PlanActionName,
		Description: "The assistant for solving complex tasks that involves calling more functions in sequence, replies with the action.",
		Properties: map[string]jsonschema.Definition{
			"subtasks": {
				Type:        jsonschema.Array,
				Description: "The message to reply with",
				Properties: map[string]jsonschema.Definition{
					"action": {
						Type:        jsonschema.String,
						Description: "The action to call",
					},
					"reasoning": {
						Type:        jsonschema.String,
						Description: "The reasoning for calling this action",
					},
				},
			},
		},
		Required: []string{"subtasks"},
	}
}
