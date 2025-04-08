package services

import (
	"encoding/json"

	"github.com/mudler/LocalAgent/pkg/config"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services/prompts"

	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/state"
)

const (
	DynamicPromptCustom = "custom"
)

var AvailableBlockPrompts = []string{
	DynamicPromptCustom,
}

func DynamicPromptsConfigMeta() []config.FieldGroup {
	return []config.FieldGroup{
					 prompts.NewDynamicPromptConfigMeta(),
	}
}

func DynamicPrompts(a *state.AgentConfig) []agent.DynamicPrompt {
	promptblocks := []agent.DynamicPrompt{}

	for _, c := range a.DynamicPrompts {
		var config map[string]string
		if err := json.Unmarshal([]byte(c.Config), &config); err != nil {
			xlog.Info("Error unmarshalling connector config", err)
			continue
		}
		switch c.Type {
		case DynamicPromptCustom:
			prompt, err := prompts.NewDynamicPrompt(config, "")
			if err != nil {
				xlog.Error("Error creating custom prompt", "error", err)
				continue
			}
			promptblocks = append(promptblocks, prompt)
		}
	}
	return promptblocks
}
