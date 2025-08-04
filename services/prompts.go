package services

import (
	"context"
	"encoding/json"

	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services/actions"
	"github.com/mudler/LocalAGI/services/prompts"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/state"
)

const (
	DynamicPromptCustom = "custom"
	DynamicPromptMemory = "memory"
)

var AvailableBlockPrompts = []string{
	DynamicPromptCustom,
	DynamicPromptMemory,
}

func DynamicPromptsConfigMeta() []config.FieldGroup {
	return []config.FieldGroup{
		prompts.NewDynamicPromptConfigMeta(),
		prompts.NewMemoryPromptConfigMeta(),
	}
}

func DynamicPrompts(dynamicConfig map[string]string) func(*state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []agent.DynamicPrompt {
	return func(a *state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []agent.DynamicPrompt {
		return func(ctx context.Context, pool *state.AgentPool) []agent.DynamicPrompt {

			memoryFilePath := memoryPath(a.Name, dynamicConfig)
			promptblocks := []agent.DynamicPrompt{}
			_, memory, err := actions.NewMemoryActions(memoryFilePath, dynamicConfig)
			if err != nil {
				xlog.Error("Error creating memory actions", "error", err)
				return promptblocks
			}

			for _, c := range a.DynamicPrompts {
				var config map[string]string
				if err := json.Unmarshal([]byte(c.Config), &config); err != nil {
					xlog.Info("Error unmarshalling connector config", err)
					continue
				}
				switch c.Type {
				case DynamicPromptCustom:
					prompt, err := prompts.NewDynamicCustomPrompt(config, "")
					if err != nil {
						xlog.Error("Error creating custom prompt", "error", err)
						continue
					}
					promptblocks = append(promptblocks, prompt)
				case DynamicPromptMemory:
					promptblocks = append(promptblocks,
						prompts.NewMemoryPrompt(config, memory),
					)
				}
			}
			return promptblocks
		}
	}
}
