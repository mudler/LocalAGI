package action

// StopActionName is the name of the action
// used by the LLM to stop any further action
const StopActionName = "stop"

func NewStop() *StopAction {
	return &StopAction{}
}

type StopAction struct{}

func (a *StopAction) Run(ActionParams) (string, error) {
	return "no-op", nil
}

func (a *StopAction) Definition() ActionDefinition {
	return ActionDefinition{
		Name:        StopActionName,
		Description: "Use this tool to stop any further action and stop the conversation.",
	}
}
