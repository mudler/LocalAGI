package filters

import (
	"encoding/json"
	"regexp"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
)

const FilterRegex = "regex"

type RegexFilter struct {
	name         string
	pattern      *regexp.Regexp
	allowOnMatch bool
	isTrigger    bool
}

type RegexFilterConfig struct {
	Name         string `json:"name"`
	Pattern      string `json:"pattern"`
	AllowOnMatch bool   `json:"allow_on_match"`
	IsTrigger    bool   `json:"is_trigger"`
}

func NewRegexFilter(configJSON string) (*RegexFilter, error) {
	var cfg RegexFilterConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, err
	}
	re, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		return nil, err
	}
	return &RegexFilter{
		name:         cfg.Name,
		pattern:      re,
		allowOnMatch: cfg.AllowOnMatch,
		isTrigger:    cfg.IsTrigger,
	}, nil
}

func (f *RegexFilter) Name() string { return f.name }
func (f *RegexFilter) Apply(job *types.Job) (bool, error) {
	input := extractInputFromJob(job)
	if f.pattern.MatchString(input) {
		return f.allowOnMatch, nil
	}
	return !f.allowOnMatch, nil
}

func (f *RegexFilter) IsTrigger() bool {
	return f.isTrigger
}

func RegexFilterConfigMeta() config.FieldGroup {
	return config.FieldGroup{
		Name:  FilterRegex,
		Label: "Regex Filter/Trigger",
		Fields: []config.Field{
			{Name: "name", Label: "Name", Type: "text", Required: true},
			{Name: "pattern", Label: "Pattern", Type: "text", Required: true},
			{Name: "allow_on_match", Label: "Allow on Match", Type: "checkbox", Required: true},
			{Name: "is_trigger", Label: "Is Trigger", Type: "checkbox", Required: true},
		},
	}
}

// extractInputFromJob attempts to extract a string input for filtering.
func extractInputFromJob(job *types.Job) string {
	if job.Metadata != nil {
		if v, ok := job.Metadata["input"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	// fallback: try to use conversation history if available
	if len(job.ConversationHistory) > 0 {
		// Use the last message content
		last := job.ConversationHistory[len(job.ConversationHistory)-1]
		return last.Content
	}
	return ""
}
