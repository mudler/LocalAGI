package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/xlog"

	"github.com/mudler/LocalAgent/services/actions"
)

const (
	// Actions
	ActionSearch                         = "search"
	ActionCustom                         = "custom"
	ActionGithubIssueLabeler             = "github-issue-labeler"
	ActionGithubIssueOpener              = "github-issue-opener"
	ActionGithubIssueCloser              = "github-issue-closer"
	ActionGithubIssueSearcher            = "github-issue-searcher"
	ActionGithubRepositoryGet            = "github-repository-get-content"
	ActionGithubRepositoryCreateOrUpdate = "github-repository-create-or-update-content"
	ActionGithubIssueReader              = "github-issue-reader"
	ActionGithubIssueCommenter           = "github-issue-commenter"
	ActionGithubREADME                   = "github-readme"
	ActionScraper                        = "scraper"
	ActionWikipedia                      = "wikipedia"
	ActionBrowse                         = "browse"
	ActionTwitterPost                    = "twitter-post"
	ActionSendMail                       = "send_mail"
	ActionGenerateImage                  = "generate_image"
	ActionCounter                        = "counter"
	ActionCallAgents                     = "call_agents"
	ActionShellcommand                   = "shell-command"
)

var AvailableActions = []string{
	ActionSearch,
	ActionCustom,
	ActionGithubIssueLabeler,
	ActionGithubIssueOpener,
	ActionGithubIssueCloser,
	ActionGithubIssueSearcher,
	ActionGithubRepositoryGet,
	ActionGithubRepositoryCreateOrUpdate,
	ActionGithubIssueReader,
	ActionGithubIssueCommenter,
	ActionGithubREADME,
	ActionScraper,
	ActionBrowse,
	ActionWikipedia,
	ActionSendMail,
	ActionGenerateImage,
	ActionTwitterPost,
	ActionCounter,
	ActionCallAgents,
	ActionShellcommand,
}

func Actions(a *state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []types.Action {
	return func(ctx context.Context, pool *state.AgentPool) []types.Action {
		allActions := []types.Action{}

		for _, a := range a.Actions {
			var config map[string]string
			if err := json.Unmarshal([]byte(a.Config), &config); err != nil {
				xlog.Error("Error unmarshalling action config", "error", err)
				continue
			}

			a, err := Action(a.Name, config, pool)
			if err != nil {
				continue
			}
			allActions = append(allActions, a)
		}

		return allActions
	}
}

func Action(name string, config map[string]string, pool *state.AgentPool) (types.Action, error) {
	var a types.Action
	var err error

	switch name {
	case ActionCustom:
		a, err = action.NewCustom(config, "")
	case ActionGenerateImage:
		a = actions.NewGenImage(config)
	case ActionSearch:
		a = actions.NewSearch(config)
	case ActionGithubIssueLabeler:
		a = actions.NewGithubIssueLabeler(context.Background(), config)
	case ActionGithubIssueOpener:
		a = actions.NewGithubIssueOpener(context.Background(), config)
	case ActionGithubIssueCloser:
		a = actions.NewGithubIssueCloser(context.Background(), config)
	case ActionGithubIssueSearcher:
		a = actions.NewGithubIssueSearch(context.Background(), config)
	case ActionGithubIssueReader:
		a = actions.NewGithubIssueReader(context.Background(), config)
	case ActionGithubIssueCommenter:
		a = actions.NewGithubIssueCommenter(context.Background(), config)
	case ActionGithubRepositoryGet:
		a = actions.NewGithubRepositoryGetContent(context.Background(), config)
	case ActionGithubRepositoryCreateOrUpdate:
		a = actions.NewGithubRepositoryCreateOrUpdateContent(context.Background(), config)
	case ActionGithubREADME:
		a = actions.NewGithubRepositoryREADME(context.Background(), config)
	case ActionScraper:
		a = actions.NewScraper(config)
	case ActionWikipedia:
		a = actions.NewWikipedia(config)
	case ActionBrowse:
		a = actions.NewBrowse(config)
	case ActionSendMail:
		a = actions.NewSendMail(config)
	case ActionTwitterPost:
		a = actions.NewPostTweet(config)
	case ActionCounter:
		a = actions.NewCounter(config)
	case ActionCallAgents:
		a = actions.NewCallAgent(config, pool)
	case ActionShellcommand:
		a = actions.NewShell(config)
	default:
		xlog.Error("Action not found", "name", name)
		return nil, fmt.Errorf("Action not found")
	}

	if err != nil {
		return nil, err
	}

	return a, nil
}
