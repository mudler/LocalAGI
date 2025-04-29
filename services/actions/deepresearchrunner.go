package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	api "github.com/mudler/LocalAGI/pkg/localoperator"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	MetadataDeepResearchResult = "deep_research_result"
)

type DeepResearchRunner struct {
	baseURL, customActionName string
	client                    *api.Client
}

func NewDeepResearchRunner(config map[string]string, defaultURL string) *DeepResearchRunner {
	if config["baseURL"] == "" {
		config["baseURL"] = defaultURL
	}

	client := api.NewClient(config["baseURL"])

	return &DeepResearchRunner{
		client:           client,
		baseURL:          config["baseURL"],
		customActionName: config["customActionName"],
	}
}

func (d *DeepResearchRunner) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := api.DeepResearchRequest{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	req := api.DeepResearchRequest{
		Topic:               result.Topic,
		MaxCycles:           result.MaxCycles,
		MaxNoActionAttempts: result.MaxNoActionAttempts,
		MaxResults:          result.MaxResults,
	}

	researchResult, err := d.client.RunDeepResearch(req)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to run deep research: %w", err)
	}

	// Format the research result into a readable string
	var resultStr string

	resultStr += "Deep research result\n"
	resultStr += fmt.Sprintf("Topic: %s\n", researchResult.Topic)
	resultStr += fmt.Sprintf("Summary: %s\n", researchResult.Summary)
	resultStr += fmt.Sprintf("Research Cycles: %d\n", researchResult.ResearchCycles)
	resultStr += fmt.Sprintf("Completion Time: %s\n\n", researchResult.CompletionTime)

	if len(researchResult.Sources) > 0 {
		resultStr += "Sources:\n"
		for _, source := range researchResult.Sources {
			resultStr += fmt.Sprintf("- %s (%s)\n  %s\n", source.Title, source.URL, source.Description)
		}
	}

	return types.ActionResult{
		Result:   fmt.Sprintf("Deep research completed successfully.\n%s", resultStr),
		Metadata: map[string]interface{}{MetadataDeepResearchResult: researchResult},
	}, nil
}

func (d *DeepResearchRunner) Definition() types.ActionDefinition {
	actionName := "run_deep_research"
	if d.customActionName != "" {
		actionName = d.customActionName
	}
	description := "Run a deep research on a specific topic, gathering information from multiple sources and providing a comprehensive summary"
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"topic": {
				Type:        jsonschema.String,
				Description: "The topic to research",
			},
			"max_cycles": {
				Type:        jsonschema.Number,
				Description: "Maximum number of research cycles to perform (optional)",
			},
			"max_no_action_attempts": {
				Type:        jsonschema.Number,
				Description: "Maximum number of attempts without taking an action (optional)",
			},
			"max_results": {
				Type:        jsonschema.Number,
				Description: "Maximum number of results to collect (optional)",
			},
		},
		Required: []string{"topic"},
	}
}

func (d *DeepResearchRunner) Plannable() bool {
	return true
}

// DeepResearchRunnerConfigMeta returns the metadata for Deep Research Runner action configuration fields
func DeepResearchRunnerConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "baseURL",
			Label:    "Base URL",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Base URL of the LocalOperator API",
		},
		{
			Name:     "customActionName",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			HelpText: "Custom name for this action",
		},
	}
}
