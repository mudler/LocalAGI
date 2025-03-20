package services

import (
	"context"
	"encoding/json"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/pkg/xlog"

	"github.com/mudler/LocalAgent/core/agent"
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

func Actions(a *state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []agent.Action {
	return func(ctx context.Context, pool *state.AgentPool) []agent.Action {
		allActions := []agent.Action{}

		for _, a := range a.Actions {
			var config map[string]string
			if err := json.Unmarshal([]byte(a.Config), &config); err != nil {
				xlog.Error("Error unmarshalling action config", "error", err)
				continue
			}

			switch a.Name {
			case ActionCustom:
				customAction, err := action.NewCustom(config, "")
				if err != nil {
					xlog.Error("Error creating custom action", "error", err)
					continue
				}
				allActions = append(allActions, customAction)
			case ActionGenerateImage:
				allActions = append(allActions, actions.NewGenImage(config))
			case ActionSearch:
				allActions = append(allActions, actions.NewSearch(config))
			case ActionGithubIssueLabeler:
				allActions = append(allActions, actions.NewGithubIssueLabeler(ctx, config))
			case ActionGithubIssueOpener:
				allActions = append(allActions, actions.NewGithubIssueOpener(ctx, config))
			case ActionGithubIssueCloser:
				allActions = append(allActions, actions.NewGithubIssueCloser(ctx, config))
			case ActionGithubIssueSearcher:
				allActions = append(allActions, actions.NewGithubIssueSearch(ctx, config))
			case ActionGithubIssueReader:
				allActions = append(allActions, actions.NewGithubIssueReader(ctx, config))
			case ActionGithubIssueCommenter:
				allActions = append(allActions, actions.NewGithubIssueCommenter(ctx, config))
			case ActionGithubRepositoryGet:
				allActions = append(allActions, actions.NewGithubRepositoryGetContent(ctx, config))
			case ActionGithubRepositoryCreateOrUpdate:
				allActions = append(allActions, actions.NewGithubRepositoryCreateOrUpdateContent(ctx, config))
			case ActionGithubREADME:
				allActions = append(allActions, actions.NewGithubRepositoryREADME(ctx, config))
			case ActionScraper:
				allActions = append(allActions, actions.NewScraper(config))
			case ActionWikipedia:
				allActions = append(allActions, actions.NewWikipedia(config))
			case ActionBrowse:
				allActions = append(allActions, actions.NewBrowse(config))
			case ActionSendMail:
				allActions = append(allActions, actions.NewSendMail(config))
			case ActionTwitterPost:
				allActions = append(allActions, actions.NewPostTweet(config))
			case ActionCounter:
				allActions = append(allActions, actions.NewCounter(config))
			case ActionCallAgents:
				allActions = append(allActions, actions.NewCallAgent(config, pool))
			case ActionShellcommand:
				allActions = append(allActions, actions.NewShell(config))
			}
		}

		return allActions
	}
}
