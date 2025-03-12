package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubRepositoryCreateOrUpdateContent struct {
	token, repository, owner, customActionName string
	context                                    context.Context
	client                                     *github.Client
}

func NewGithubRepositoryCreateOrUpdateContent(ctx context.Context, config map[string]string) *GithubRepositoryCreateOrUpdateContent {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubRepositoryCreateOrUpdateContent{
		client:           client,
		token:            config["token"],
		repository:       config["repository"],
		owner:            config["owner"],
		customActionName: config["customActionName"],
		context:          ctx,
	}
}

func (g *GithubRepositoryCreateOrUpdateContent) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Path       string `json:"path"`
		Repository string `json:"repository"`
		Owner      string `json:"owner"`
		Content    string `json:"content"`
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

	fileContent, _, err := g.client.Repositories.CreateFile(g.context, result.Owner, result.Repository, result.Path, &github.RepositoryContentFileOptions{
		Content: []byte(result.Content),
	})
	if err != nil {
		resultString := fmt.Sprintf("Error creating content : %v", err)
		return action.ActionResult{Result: resultString}, err
	}

	return action.ActionResult{Result: fmt.Sprintf("File created/updated: %s\n", fileContent.GetURL())}, err
}

func (g *GithubRepositoryCreateOrUpdateContent) Definition() action.ActionDefinition {
	actionName := "github_repository_create_or_update_content"
	actionDescription := "Create or update a file in a GitHub repository"
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
				"content": {
					Type:        jsonschema.String,
					Description: "The content to create/update",
				},
			},
			Required: []string{"path", "content"},
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
			"content": {
				Type:        jsonschema.String,
				Description: "The content to create/update",
			},
		},
		Required: []string{"path", "repository", "owner", "content"},
	}
}
