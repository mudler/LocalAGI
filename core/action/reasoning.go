package action

import (
	"context"

	"github.com/sashabaranov/go-openai/jsonschema"
)

// NewReasoning creates a new reasoning action
// The reasoning action is special as it tries to force the LLM
// to think about what to do next
func NewReasoning() *ReasoningAction {
	return &ReasoningAction{}
}

type ReasoningAction struct{}

type ReasoningResponse struct {
	Reasoning string `json:"reasoning"`
}

func (a *ReasoningAction) Run(context.Context, ActionParams) (string, error) {
	return "no-op", nil
}

func (a *ReasoningAction) Definition() ActionDefinition {
	return ActionDefinition{
		Name:        "pick_action",
		Description: "try to understand what's the best thing to do and pick an action with a reasoning",
		Properties: map[string]jsonschema.Definition{
			"reasoning": {
				Type:        jsonschema.String,
				Description: "A detailed reasoning on what would you do in this situation.",
			},
		},
		Required: []string{"reasoning"},
	}
}
