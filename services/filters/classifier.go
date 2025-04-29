package filters

import (
	"encoding/json"
	"fmt"

	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const FilterClassifier = "classifier"

type ClassifierFilter struct {
	name         string
	client       *openai.Client
	model        string
	description  string
	allowOnMatch bool
	isTrigger    bool
}

type ClassifierFilterConfig struct {
	Name         string `json:"name"`
	Model        string `json:"model,omitempty"`
	APIURL       string `json:"api_url,omitempty"`
	Description  string `json:"description"`
	AllowOnMatch bool   `json:"allow_on_match"`
	IsTrigger    bool   `json:"is_trigger"`
}

func NewClassifierFilter(configJSON string, a *state.AgentConfig) (*ClassifierFilter, error) {
	var cfg ClassifierFilterConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, err
	}
	var model string
	if cfg.Model != "" {
		model = cfg.Model
	} else {
		model = a.Model
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("Classifier with no name")
	}
	if cfg.Description == "" {
		return nil, fmt.Errorf("%s classifier has no description", cfg.Name)
	}
	apiUrl := a.APIURL
	if cfg.APIURL != "" {
		apiUrl = cfg.APIURL
	}
	client := llm.NewClient(a.APIKey, apiUrl, "1m")

	return &ClassifierFilter{
		name:         cfg.Name,
		model:        model,
		description:  cfg.Description,
		client:       client,
		allowOnMatch: cfg.AllowOnMatch,
		isTrigger:    cfg.IsTrigger,
	}, nil
}

const fmtT = `
  Does the below message fit the description "%s"

  %s
  `

func (f *ClassifierFilter) Name() string { return f.name }
func (f *ClassifierFilter) Apply(job *types.Job) (bool, error) {
	input := extractInputFromJob(job)
	guidance := fmt.Sprintf(fmtT, f.description, input)
	var result struct {
		Asserted bool `json:"answer"`
	}
	err := llm.GenerateTypedJSON(job.GetContext(), f.client, guidance, f.model, jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"answer": {
				Type:        jsonschema.Boolean,
				Description: "The answer to the first question",
			},
		},
		Required: []string{"answer"},
	}, &result)
	if err != nil {
		return false, err
	}

	if result.Asserted {
		return f.allowOnMatch, nil
	}
	return !f.allowOnMatch, nil
}

func (f *ClassifierFilter) IsTrigger() bool {
	return f.isTrigger
}

func ClassifierFilterConfigMeta() config.FieldGroup {
	return config.FieldGroup{
		Name:  FilterClassifier,
		Label: "Classifier Filter/Trigger",
		Fields: []config.Field{
			{Name: "name", Label: "Name", Type: "text", Required: true},
			{Name: "model", Label: "Model", Type: "text", Required: false,
				HelpText: "The LLM to use, usually a smaller one. Leave blank to use the same as the agent's"},
			{Name: "api_url", Label: "API URL", Type: "url", Required: false,
				HelpText: "The URL of the LLM service if different from the agent's"},
			{Name: "description", Label: "Description", Type: "text", Required: true,
				HelpText: "Describe the type of content to match against e.g. 'technical support request'"},
			{Name: "allow_on_match", Label: "Allow on Match", Type: "checkbox", Required: true},
			{Name: "is_trigger", Label: "Is Trigger", Type: "checkbox", Required: true},
		},
	}
}
