package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubRepositoryGetContent struct {
	token, repository, owner, customActionName string
	client                                     *github.Client
}

func NewGithubRepositoryGetContent(config map[string]string) *GithubRepositoryGetContent {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositoryGetContent{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
	}
}

func (g *GithubRepositoryGetContent) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Path       string `json:"path"`
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

	fileContent, directoryContent, _, err := g.client.Repositories.GetContents(ctx, result.Owner, result.Repository, result.Path, nil)
	if err != nil {
		resultString := fmt.Sprintf("Error getting content : %v", err)
		return types.ActionResult{Result: resultString}, err
	}

	if len(directoryContent) > 0 {
		resultString := fmt.Sprintf("Directory found: %s\n", result.Path)
		for _, f := range directoryContent {
			resultString += fmt.Sprintf("File: %s\n", f.GetName())
		}
		return types.ActionResult{Result: resultString}, err
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return types.ActionResult{}, err
	}

	return types.ActionResult{Result: fmt.Sprintf("File %s\nContent:%s\n", result.Path, content)}, err
}

func (g *GithubRepositoryGetContent) Definition() types.ActionDefinition {
	actionName := "get_github_repository_content"
	actionDescription := "Get content of a file or directory in a github repository"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: actionDescription,
			Properties: map[string]jsonschema.Definition{
				"path": {
					Type:        jsonschema.String,
					Description: "The path to the file or directory",
				},
			},
			Required: []string{"path"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: actionDescription,
		Properties: map[string]jsonschema.Definition{
			"path": {
				Type:        jsonschema.String,
				Description: "The path to the file or directory",
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
		Required: []string{"path", "repository", "owner"},
	}
}

func (a *GithubRepositoryGetContent) Plannable() bool {
	return true
}
