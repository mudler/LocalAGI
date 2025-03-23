package actions

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesReader struct {
	token, repository, owner, customActionName string
	context                                    context.Context
	client                                     *github.Client
}

func NewGithubIssueReader(ctx context.Context, config map[string]string) *GithubIssuesReader {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	return &GithubIssuesReader{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
		context:          ctx,
	}
}

func (g *GithubIssuesReader) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		Label       string `json:"label"`
		IssueNumber int    `json:"issue_number"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return types.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	issue, _, err := g.client.Issues.Get(g.context, result.Owner, result.Repository, result.IssueNumber)
	if err == nil && issue != nil {
		return types.ActionResult{
			Result: fmt.Sprintf(
				"Issue %d Repository: %s\nTitle: %s\nBody: %s",
				*issue.Number, *issue.Repository.FullName, *issue.Title, *issue.Body)}, nil
	}
	if err != nil {
		return types.ActionResult{Result: fmt.Sprintf("Error fetching issue: %s", err.Error())}, err
	}
	return types.ActionResult{Result: fmt.Sprintf("No issue found")}, err
}

func (g *GithubIssuesReader) Definition() types.ActionDefinition {
	actionName := "read_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	description := "Read a Github issue."
	if g.repository != "" && g.owner != "" {
		return types.ActionDefinition{
			Name:        types.ActionDefinitionName(actionName),
			Description: description,
			Properties: map[string]jsonschema.Definition{
				"issue_number": {
					Type:        jsonschema.Number,
					Description: "The number of the issue to read.",
				},
			},
			Required: []string{"issue_number"},
		}
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(actionName),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"issue_number": {
				Type:        jsonschema.Number,
				Description: "The number of the issue to add the label to.",
			},
			"repository": {
				Type:        jsonschema.String,
				Description: "The repository to add the label to.",
			},
			"owner": {
				Type:        jsonschema.String,
				Description: "The owner of the repository.",
			},
		},
		Required: []string{"issue_number", "repository", "owner"},
	}
}

func (a *GithubIssuesReader) Plannable() bool {
	return true
}
