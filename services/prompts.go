package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/services/actions"
	"github.com/mudler/LocalAGI/services/prompts"
	"github.com/mudler/xlog"

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

type dynamicPrompt struct {
	agent.DynamicPrompt
	Name string
}

func dynamicPrompts(customDirectory string, existingConfigs map[string]map[string]string) (allPrompts []dynamicPrompt) {
	files, err := os.ReadDir(customDirectory)
	if err != nil {
		xlog.Error("Error reading custom actions directory", "error", err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".go" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(customDirectory, file.Name()))
		if err != nil {
			xlog.Error("Error reading custom action file", "error", err, "file", file.Name())
			continue
		}
		dynamicPromptName := strings.TrimSuffix(file.Name(), ".go")

		dynamicPromptConfig := map[string]string{
			"name": dynamicPromptName,
			"code": string(content),
		}

		if c, exists := existingConfigs[dynamicPromptName]; exists {
			dynamicPromptConfig["configuration"] = c["configuration"]
		}

		a, err := prompts.NewDynamicCustomPrompt(dynamicPromptConfig, "")
		if err != nil {
			xlog.Error("Error creating custom dynamic prompt", "error", err, "file", file.Name())
			continue
		}

		if !a.CanRender() {
			continue
		}

		allPrompts = append(allPrompts, dynamicPrompt{
			DynamicPrompt: a,
			Name:          dynamicPromptName,
		})
	}
	return
}

func DynamicPromptsConfigMeta(customDirectory string) []config.FieldGroup {
	defaultDynamicPrompts := []config.FieldGroup{
		prompts.NewDynamicPromptConfigMeta(),
		prompts.NewMemoryPromptConfigMeta(),
	}

	if customDirectory != "" {
		prompts := dynamicPrompts(customDirectory, map[string]map[string]string{})
		for _, p := range prompts {
			defaultDynamicPrompts = append(defaultDynamicPrompts, config.FieldGroup{
				Name:  p.Name,
				Label: p.Name,
				Fields: []config.Field{
					{
						Name:     "configuration",
						Label:    "Configuration",
						Type:     config.FieldTypeTextarea,
						HelpText: "Configuration for the custom prompt",
					},
				},
			})
		}
	}

	return defaultDynamicPrompts
}

func DynamicPrompts(dynamicConfig map[string]string) func(*state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []agent.DynamicPrompt {
	return func(a *state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []agent.DynamicPrompt {
		return func(ctx context.Context, pool *state.AgentPool) []agent.DynamicPrompt {
			customDirectory := dynamicConfig[CustomActionsDir]

			existingDynamicPromptsConfigs := map[string]map[string]string{}
			for _, c := range a.DynamicPrompts {
				var config map[string]string
				if err := json.Unmarshal([]byte(c.Config), &config); err != nil {
					xlog.Info("Error unmarshalling connector config", err)
					continue
				}

				existingDynamicPromptsConfigs[c.Type] = config
			}

			dynamicPromptsFound := dynamicPrompts(customDirectory, existingDynamicPromptsConfigs)

			memoryIdxPath := memoryIndexPath(a.Name, dynamicConfig)
			promptblocks := []agent.DynamicPrompt{}

			for _, c := range a.DynamicPrompts {
				config := existingDynamicPromptsConfigs[c.Type]

				switch c.Type {
				case DynamicPromptCustom:
					prompt, err := prompts.NewDynamicCustomPrompt(config, "")
					if err != nil {
						xlog.Error("Error creating custom prompt", "error", err)
						continue
					}
					promptblocks = append(promptblocks, prompt)
				case DynamicPromptMemory:
					_, memory, _, _ := actions.NewMemoryActions(memoryIdxPath, dynamicConfig)

					promptblocks = append(promptblocks,
						prompts.NewMemoryPrompt(config, memory),
					)
				default:
					// Check if we have configured a custom dynamic prompt coming from a directory
					for _, p := range dynamicPromptsFound {
						if p.Name == c.Type {
							promptblocks = append(promptblocks, p.DynamicPrompt)
						}
					}
				}
			}
			return promptblocks
		}
	}
}
