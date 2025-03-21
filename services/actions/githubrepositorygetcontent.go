package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubRepositoryGetContent struct {
	token, repository, owner, customActionName string
	context                                    context.Context
	client                                     *github.Client
}

func NewGithubRepositoryGetContent(ctx context.Context, config map[string]string) *GithubRepositoryGetContent {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositoryGetContent{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
		context:          ctx,
	}
}

func (g *GithubRepositoryGetContent) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Path       string `json:"path"`
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	fileContent, directoryContent, _, err := g.client.Repositories.GetContents(g.context, result.Owner, result.Repository, result.Path, nil)
	if err != nil {
		resultString := fmt.Sprintf("Error getting content : %v", err)
		return action.ActionResult{Result: resultString}, err
	}

	if len(directoryContent) > 0 {
		resultString := fmt.Sprintf("Directory found: %s\n", result.Path)
		for _, f := range directoryContent {
			resultString += fmt.Sprintf("File: %s\n", f.GetName())
		}
		return action.ActionResult{Result: resultString}, err
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return action.ActionResult{}, err
	}

	return action.ActionResult{Result: fmt.Sprintf("File %s\nContent:%s\n", result.Path, content)}, err
}

func (g *GithubRepositoryGetContent) Definition() action.ActionDefinition {
	actionName := "get_github_repository_content"
	actionDescription := "Get content of a file or directory in a github repository"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	if g.repository != "" && g.owner != "" {
		return action.ActionDefinition{
			Name:        action.ActionDefinitionName(actionName),
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
	return action.ActionDefinition{
		Name:        action.ActionDefinitionName(actionName),
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
