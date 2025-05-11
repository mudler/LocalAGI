package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubRepositoryREADME struct {
	token, customActionName string
	client                  *github.Client
}

func NewGithubRepositoryREADME(config map[string]string) *GithubRepositoryREADME {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositoryREADME{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
	}
}

func (g *GithubRepositoryREADME) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}
	fileContent, _, err := g.client.Repositories.GetReadme(ctx, result.Owner, result.Repository, &github.RepositoryContentGetOptions{})
	if err != nil {
		resultString := fmt.Sprintf("Error getting content : %v", err)
		return types.ActionResult{Result: resultString}, err
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{Result: content}, err
}

func (g *GithubRepositoryREADME) Definition() types.ActionDefinition {
	actionName := "github_readme"
	actionDescription := "Get the README file of a GitHub repository to have a basic understanding of the project."
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: actionDescription,
		Properties: map[string]jsonschema.Definition{
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to search in",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository",
			},
		},
		Required: []string{"repository", "owner"},
	}
}

func (a *GithubRepositoryREADME) Plannable() bool {
	return true
}

// GithubRepositoryREADMEConfigMeta returns the metadata for GitHub Repository README action configuration fields
func GithubRepositoryREADMEConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "token",
			Label:    "GitHub Token",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "GitHub API token with repository access",
		},
		{
			Name:     "customActionName",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			HelpText: "Custom name for this action",
		},
	}
}
