package services

import (
	"encoding/json"

	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services/prompts"

	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/state"
)

const (
	// Connectors
	DynamicPromptCustom = "custom"
)

var AvailableBlockPrompts = []string{
	DynamicPromptCustom,
}

func PromptBlocks(a *state.AgentConfig) []agent.PromptBlock {
	promptblocks := []agent.PromptBlock{}

	for _, c := range a.PromptBlocks {
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
