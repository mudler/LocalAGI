package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v61/github"
	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type GithubIssuesLabeler struct {
	token, repository, owner, customActionName string
	availableLabels                            []string
	context                                    context.Context
	client                                     *github.Client
}

func NewGithubIssueLabeler(ctx context.Context, config map[string]string) *GithubIssuesLabeler {
	client := github.NewClient(nil).WithAuthToken(config["token"])

	// Get available labels
	availableLabels := []string{"bug", "enhancement"}

	if config["availableLabels"] != "" {
		availableLabels = strings.Split(config["availableLabels"], ",")
	}

	return &GithubIssuesLabeler{
		client:           client,
		token:            config["token"],
		customActionName: config["customActionName"],
		repository:       config["repository"],
		owner:            config["owner"],
		context:          ctx,
		availableLabels:  availableLabels,
	}
}

func (g *GithubIssuesLabeler) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Repository  string `json:"repository"`
		Owner       string `json:"owner"`
		Label       string `json:"label"`
		IssueNumber int    `json:"issue_number"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		return action.ActionResult{}, err
	}

	if g.repository != "" && g.owner != "" {
		result.Repository = g.repository
		result.Owner = g.owner
	}

	labels, _, err := g.client.Issues.AddLabelsToIssue(g.context, result.Owner, result.Repository, result.IssueNumber, []string{result.Label})
	//labelsNames := []string{}
	for _, l := range labels {
		xlog.Info("Label added", "label", l.Name)
		//labelsNames = append(labelsNames, l.GetName())
	}

	resultString := fmt.Sprintf("Added label '%s' to issue %d in repository %s/%s", result.Label, result.IssueNumber, result.Owner, result.Repository)
	if err != nil {
		resultString = fmt.Sprintf("Error adding label '%s' to issue %d in repository %s/%s: %v", result.Label, result.IssueNumber, result.Owner, result.Repository, err)
	}
	return action.ActionResult{Result: resultString}, err
}

func (g *GithubIssuesLabeler) Definition() action.ActionDefinition {
	actionName := "add_label_to_github_issue"
	if g.customActionName != "" {
		actionName = g.customActionName
	}
	if g.repository != "" && g.owner != "" {
		return action.ActionDefinition{
			Name:        action.ActionDefinitionName(actionName),
			Description: "Add a label to a Github issue. You might want to assign labels to issues to categorize them.",
			Properties: map[string]jsonschema.Definition{
				"issue_number": {
					Type:        jsonschema.Number,
					Description: "The number of the issue to add the label to.",
				},
				"label": {
					Type:        jsonschema.String,
					Description: "The label to add to the issue.",
					Enum:        g.availableLabels,
				},
			},
			Required: []string{"issue_number", "label"},
		}
	}
	return action.ActionDefinition{
		Name:        action.ActionDefinitionName(actionName),
		Description: "Add a label to a Github issue. You might want to assign labels to issues to categorize them.",
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
			"label": {
				Type:        jsonschema.String,
				Description: "The label to add to the issue.",
				Enum:        g.availableLabels,
			},
		},
		Required: []string{"issue_number", "repository", "owner", "label"},
	}
}
