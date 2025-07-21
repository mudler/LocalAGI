package action

import (
	"context"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const StateActionName = "update_state"

func NewState() *StateAction {
	return &StateAction{}
}

type StateAction struct{}

func (a *StateAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{Result: "internal state has been updated"}, nil
}

func (a *StateAction) Plannable() bool {
	return false
}

func (a *StateAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        StateActionName,
		Description: "update the agent state (short memory) with the current state of the conversation.",
		Properties: map[string]jsonschema.Definition{
			"goal": {
				Type:        jsonschema.String,
				Description: "The current goal of the agent.",
			},
			"doing_next": {
				Type:        jsonschema.String,
				Description: "The next action the agent will do.",
			},
			"done_history": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
				Description: "A list of actions that the agent has done.",
			},
			"now_doing": {
				Type:        jsonschema.String,
				Description: "The current action the agent is doing.",
			},
			"memories": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
				Description: "A list of memories to keep between conversations.",
			},
		},
	}
}
