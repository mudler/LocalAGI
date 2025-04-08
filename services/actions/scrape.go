package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/tmc/langchaingo/tools/scraper"
)

func NewScraper(config map[string]string) *ScraperAction {

	return &ScraperAction{}
}

type ScraperAction struct{}

func (a *ScraperAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		URL string `json:"url"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}
	scraper, err := scraper.New()
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}
	res, err := scraper.Call(ctx, result.URL)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}
	return types.ActionResult{Result: res}, nil
}

func (a *ScraperAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
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

func (a *ScraperAction) Plannable() bool {
	return true
}
