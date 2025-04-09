package action

import (
	"context"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// NewIntention creates a new intention action
// The inention action is special as it tries to identify
// a tool to use and a reasoning over to use it
func NewIntention(s ...string) *IntentAction {
	return &IntentAction{tools: s}
}

type IntentAction struct {
	tools []string
}
type IntentResponse struct {
	Tool      string `json:"tool"`
	Reasoning string `json:"reasoning"`
}

func (a *IntentAction) Run(context.Context, types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{}, nil
}

func (a *IntentAction) Plannable() bool {
	return false
}

func (a *IntentAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "pick_tool",
		Description: "Pick a tool",
		Properties: map[string]jsonschema.Definition{
			"reasoning": {
				Type:        jsonschema.String,
				Description: "A detailed reasoning on why you want to call this tool.",
			},
			"tool": {
				Type:        jsonschema.String,
				Description: "The tool you want to use",
				Enum:        a.tools,
			},
		},
		Required: []string{"tool", "reasoning"},
	}
}
