package prompts

import (
	"context"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
)

type MemoryLayer interface {
	Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error)
}
type MemoryPrompt struct {
	config map[string]string
	memory MemoryLayer
}

func NewMemoryPrompt(config map[string]string, memory MemoryLayer) *MemoryPrompt {
	return &MemoryPrompt{
		config: config,
		memory: memory,
	}
}

func NewMemoryPromptConfigMeta() config.FieldGroup {
	return config.FieldGroup{
		Name:  "memory",
		Label: "Memory",
	}
}

func (a *MemoryPrompt) Render(c *agent.Agent) (types.PromptResult, error) {
	result, err := a.memory.Run(c.Context(), c.SharedState(), types.ActionParams{})
	if err != nil {
		return types.PromptResult{}, err
	}

	return types.PromptResult{
		Content: result.Result,
	}, nil
}

func (a *MemoryPrompt) Role() string {
	return "system"
}
