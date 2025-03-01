package actions

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
	"jaytaylor.com/html2text"
)

func NewBrowse(config map[string]string) *BrowseAction {

	return &BrowseAction{}
}

type BrowseAction struct{}

func (a *BrowseAction) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		URL string `json:"url"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}
	// download page with http.Client
	client := &http.Client{}
	req, err := http.NewRequest("GET", result.URL, nil)
	if err != nil {
		return action.ActionResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return action.ActionResult{}, err
	}
	defer resp.Body.Close()
	pagebyte, err := io.ReadAll(resp.Body)
	if err != nil {
		return action.ActionResult{}, err
	}

	rendered, err := html2text.FromString(string(pagebyte), html2text.Options{PrettyTables: true})

	if err != nil {
		return action.ActionResult{}, err
	}

	return action.ActionResult{Result: fmt.Sprintf("The webpage '%s' content is:\n%s", result.URL, rendered)}, nil
}

func (a *BrowseAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "browse",
		Description: "Use this tool to visit an URL. It browse a website page and return the text content.",
		Properties: map[string]jsonschema.Definition{
			"url": {
				Type:        jsonschema.String,
				Description: "The website URL.",
			},
		},
		Required: []string{"url"},
	}
}
