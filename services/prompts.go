package services

import (
	"encoding/json"

	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services/prompts"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/state"
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
