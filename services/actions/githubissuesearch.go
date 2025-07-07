package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssueSearch struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubIssueSearch(config map[string]string) *GithubIssueSearch {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssueSearch{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
	}
}

func (g *GithubIssueSearch) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Query      string `json:"query"`
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	query := fmt.Sprintf("%s in:%s user:%s", result.Query, result.Repository, result.Owner)
	resultString := ""
	issues, _, err := g.client.Search.Issues(ctx, query, &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 5},
		Order:       "desc",
		//Sort:        "created",
	})
	if err != nil {
		resultString = fmt.Sprintf("Error listing issues: %v", err)
		return types.ActionResult{Result: resultString}, err
	}
	for _, i := range issues.Issues {
		xlog.Info("Issue found", "title", i.GetTitle())
		resultString += fmt.Sprintf("Issue found: %s\n", i.GetTitle())
		resultString += fmt.Sprintf("URL: %s\n", i.GetHTMLURL())
		//	resultString += fmt.Sprintf("Body: %s\n", i.GetBody())
	}

	return types.ActionResult{Result: resultString}, err
}

func (g *GithubIssueSearch) Definition() types.ActionDefinition {
	actionName := "search_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: "Search between github issues",
			Properties: map[string]jsonschema.Definition{
				"query": {
					Type:        jsonschema.String,
					Description: "The text to search for",
				},
			},
			Required: []string{"query"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: "Search between github issues",
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "The text to search for",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to search in",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository",
			},
		},
		Required: []string{"query", "repository", "owner"},
	}
}

func (a *GithubIssueSearch) Plannable() bool {
	return true
}

// GithubIssueSearchConfigMeta returns the metadata for GitHub Issue Search action configuration fields
func GithubIssueSearchConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "GitHub Token",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "GitHub API token with repository access",
		},
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "GitHub repository name",
		},
		{
			Name:     "owner",
			Label:    "Owner",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "GitHub repository owner",
		},
		{
			Name:     "customActionName",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			HelpText: "Custom name for this action",
		},
	}
}
