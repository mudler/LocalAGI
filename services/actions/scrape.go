package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/tmc/langchaingo/tools/scraper"
)

func NewScraper(config map[string]string) *ScraperAction {

	return &ScraperAction{}
}

type ScraperAction struct{}

func (a *ScraperAction) Run(ctx context.Context, params action.ActionParams) (string, error) {
	result := struct {
		URL string `json:"url"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}
	scraper, err := scraper.New()
	if err != nil {
		fmt.Printf("error: %v", err)

		return "", err
	}
	return scraper.Call(ctx, result.URL)
}

func (a *ScraperAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "scrape",
		Description: "Scrapes a full website content and returns the entire site data.",
		Properties: map[string]jsonschema.Definition{
			"url": {
				Type:        jsonschema.String,
				Description: "The website URL.",
			},
		},
		Required: []string{"url"},
	}
}
