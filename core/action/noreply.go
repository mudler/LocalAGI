package action

import (
	"context"

	"github.com/mudler/LocalAGI/core/types"
)

// StopActionName is the name of the action
// used by the LLM to stop any further action
const StopActionName = "stop"

func NewStop() *StopAction {
	return &StopAction{}
}

type StopAction struct{}

func (a *StopAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	return types.ActionResult{}, nil
}

func (a *StopAction) Plannable() bool {
	return false
}

func (a *StopAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        StopActionName,
		Description: "Use this tool to stop any further action and stop the conversation. You must use this when it looks like there is a conclusion to the conversation or the topic diverged too much from the original conversation. For instance if the user offer his help and you already replied with a message, you can use this tool to stop the conversation.",
	}
}
