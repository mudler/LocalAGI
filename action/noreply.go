package action

import "context"

// StopActionName is the name of the action
// used by the LLM to stop any further action
const StopActionName = "stop"

func NewStop() *StopAction {
	return &StopAction{}
}

type StopAction struct{}

func (a *StopAction) Run(context.Context, ActionParams) (string, error) {
	return "no-op", nil
}

func (a *StopAction) Definition() ActionDefinition {
	return ActionDefinition{
		Name:        StopActionName,
		Description: "Use this tool to stop any further action and stop the conversation. You must use this when: the user wants to stop the conversation, it seems that the user does not need any additional answer, it looks like there is already a conclusion to the conversation or the topic diverged too much from the original conversation.",
	}
}
