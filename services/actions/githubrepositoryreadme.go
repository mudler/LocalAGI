package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubRepositoryREADME struct {
	token, customActionName string
	context                 context.Context
	client                  *github.Client
}

func NewGithubRepositoryREADME(ctx context.Context, config map[string]string) *GithubRepositoryREADME {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositoryREADME{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		context:          ctx,
	}
}

func (g *GithubRepositoryREADME) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return types.ActionResult{}, err
	}
	fileContent, _, err := g.client.Repositories.GetReadme(g.context, result.Owner, result.Repository, &github.RepositoryContentGetOptions{})
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
