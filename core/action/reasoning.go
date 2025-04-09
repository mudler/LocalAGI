package action

import (
	"context"

	"github.com/mudler/LocalAGI/core/types"
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

func (a *ReasoningAction) Run(context.Context, types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{}, nil
}

func (a *ReasoningAction) Plannable() bool {
	return false
}

func (a *ReasoningAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
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
