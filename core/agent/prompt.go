package agent

import "github.com/mudler/LocalAGI/core/types"

type DynamicPrompt interface {
	Render(a *Agent) (types.PromptResult, error)
	Role() string
}
