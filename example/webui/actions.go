package main

import (
	"context"
	"encoding/json"

	"github.com/mudler/local-agent-framework/action"
	"github.com/mudler/local-agent-framework/xlog"

	. "github.com/mudler/local-agent-framework/agent"
	"github.com/mudler/local-agent-framework/external"
)

const (
	// Actions
	ActionSearch              = "search"
	ActionCustom              = "custom"
	ActionGithubIssueLabeler  = "github-issue-labeler"
	ActionGithubIssueOpener   = "github-issue-opener"
	ActionGithubIssueCloser   = "github-issue-closer"
	ActionGithubIssueSearcher = "github-issue-searcher"
	ActionScraper             = "scraper"
	ActionWikipedia           = "wikipedia"
	ActionBrowse              = "browse"
	ActionSendMail            = "send_mail"
)

var AvailableActions = []string{
	ActionSearch,
	ActionCustom,
	ActionGithubIssueLabeler,
	ActionGithubIssueOpener,
	ActionGithubIssueCloser,
	ActionGithubIssueSearcher,
	ActionScraper,
	ActionBrowse,
	ActionWikipedia,
	ActionSendMail,
}

func (a *AgentConfig) availableActions(ctx context.Context) []Action {
	actions := []Action{}

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
			actions = append(actions, customAction)
		case ActionSearch:
			actions = append(actions, external.NewSearch(config))
		case ActionGithubIssueLabeler:
			actions = append(actions, external.NewGithubIssueLabeler(ctx, config))
		case ActionGithubIssueOpener:
			actions = append(actions, external.NewGithubIssueOpener(ctx, config))
		case ActionGithubIssueCloser:
			actions = append(actions, external.NewGithubIssueCloser(ctx, config))
		case ActionGithubIssueSearcher:
			actions = append(actions, external.NewGithubIssueSearch(ctx, config))
		case ActionScraper:
			actions = append(actions, external.NewScraper(config))
		case ActionWikipedia:
			actions = append(actions, external.NewWikipedia(config))
		case ActionBrowse:
			actions = append(actions, external.NewBrowse(config))
		case ActionSendMail:
			actions = append(actions, external.NewSendMail(config))
		}
	}

	return actions
}
