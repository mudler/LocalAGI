package action

import (
	"context"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// NewGoal creates a new intention action
// The inention action is special as it tries to identify
// a tool to use and a reasoning over to use it
func NewGoal(s ...string) *GoalAction {
	return &GoalAction{tools: s}
}

type GoalAction struct {
	tools []string
}
type GoalResponse struct {
	Goal     string `json:"goal"`
	Achieved bool   `json:"achieved"`
}

func (a *GoalAction) Run(context.Context, types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{}, nil
}

func (a *GoalAction) Plannable() bool {
	return false
}

func (a *GoalAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "goal",
		Description: "Check if the goal is achieved",
		Properties: map[string]jsonschema.Definition{
			"goal": {
				Type:        jsonschema.String,
				Description: "The goal to check if it is achieved.",
			},
			"achieved": {
				Type:        jsonschema.Boolean,
				Description: "Whether the goal is achieved",
			},
		},
		Required: []string{"goal", "achieved"},
	}
}
