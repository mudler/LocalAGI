package action

import (
	"context"

	"github.com/sashabaranov/go-openai/jsonschema"
)

// PlanActionName is the name of the plan action
// used by the LLM to schedule more actions
const PlanActionName = "plan"

func NewPlan(plannableActions []string) *PlanAction {
	return &PlanAction{
		plannables: plannableActions,
	}
}

type PlanAction struct {
	plannables []string
}

type PlanResult struct {
	Subtasks []PlanSubtask `json:"subtasks"`
	Goal     string        `json:"goal"`
}
type PlanSubtask struct {
	Action    string `json:"action"`
	Reasoning string `json:"reasoning"`
}

func (a *PlanAction) Run(context.Context, ActionParams) (ActionResult, error) {
	return ActionResult{}, nil
}

func (a *PlanAction) Plannable() bool {
	return false
}

func (a *PlanAction) Definition() ActionDefinition {
	return ActionDefinition{
		Name:        PlanActionName,
		Description: "The assistant for solving complex tasks that involves calling more functions in sequence, replies with the action.",
		Properties: map[string]jsonschema.Definition{
			"subtasks": {
				Type:        jsonschema.Array,
				Description: "The subtasks to be executed",
				Items: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"action": {
							Type:        jsonschema.String,
							Description: "The action to call",
							Enum:        a.plannables,
						},
						"reasoning": {
							Type:        jsonschema.String,
							Description: "The reasoning for calling this action",
						},
					},
				},
			},
			"goal": {
				Type:        jsonschema.String,
				Description: "The goal of this plan",
			},
		},
		Required: []string{"subtasks", "goal"},
	}
}
