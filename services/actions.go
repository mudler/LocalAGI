package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"

	"github.com/mudler/LocalAGI/services/actions"
)

const (
	// Actions
	ActionSearch                         = "search"
	ActionCustom                         = "custom"
	ActionBrowserAgentRunner             = "browser-agent-runner"
	ActionDeepResearchRunner             = "deep-research-runner"
	ActionGithubIssueLabeler             = "github-issue-labeler"
	ActionGithubIssueOpener              = "github-issue-opener"
	ActionGithubIssueCloser              = "github-issue-closer"
	ActionGithubIssueSearcher            = "github-issue-searcher"
	ActionGithubRepositoryGet            = "github-repository-get-content"
	ActionGithubRepositoryCreateOrUpdate = "github-repository-create-or-update-content"
	ActionGithubIssueReader              = "github-issue-reader"
	ActionGithubIssueCommenter           = "github-issue-commenter"
	ActionGithubPRReader                 = "github-pr-reader"
	ActionGithubPRCommenter              = "github-pr-commenter"
	ActionGithubPRReviewer               = "github-pr-reviewer"
	ActionGithubPRCreator                = "github-pr-creator"
	ActionGithubGetAllContent            = "github-get-all-repository-content"
	ActionGithubREADME                   = "github-readme"
	ActionScraper                        = "scraper"
	ActionWikipedia                      = "wikipedia"
	ActionBrowse                         = "browse"
	ActionTwitterPost                    = "twitter-post"
	ActionSendMail                       = "send-mail"
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
	ActionGithubGetAllContent,
	ActionBrowserAgentRunner,
	ActionDeepResearchRunner,
	ActionGithubRepositoryCreateOrUpdate,
	ActionGithubIssueReader,
	ActionGithubIssueCommenter,
	ActionGithubPRReader,
	ActionGithubPRCommenter,
	ActionGithubPRReviewer,
	ActionGithubPRCreator,
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

func Actions(actionsConfigs map[string]string) func(a *state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []types.Action {
	return func(a *state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []types.Action {
		return func(ctx context.Context, pool *state.AgentPool) []types.Action {
			allActions := []types.Action{}

			agentName := a.Name

			for _, a := range a.Actions {
				var config map[string]string
				if err := json.Unmarshal([]byte(a.Config), &config); err != nil {
					xlog.Error("Error unmarshalling action config", "error", err)
					continue
				}

				a, err := Action(a.Name, agentName, config, pool, actionsConfigs)
				if err != nil {
					continue
				}
				allActions = append(allActions, a)
			}

			return allActions
		}
	}

}

func Action(name, agentName string, config map[string]string, pool *state.AgentPool, actionsConfigs map[string]string) (types.Action, error) {
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
		a = actions.NewGithubIssueLabeler(config)
	case ActionGithubIssueOpener:
		a = actions.NewGithubIssueOpener(config)
	case ActionGithubIssueCloser:
		a = actions.NewGithubIssueCloser(config)
	case ActionGithubIssueSearcher:
		a = actions.NewGithubIssueSearch(config)
	case ActionBrowserAgentRunner:
		a = actions.NewBrowserAgentRunner(config, actionsConfigs["browser-agent-runner-base-url"])
	case ActionDeepResearchRunner:
		a = actions.NewDeepResearchRunner(config, actionsConfigs["deep-research-runner-base-url"])
	case ActionGithubIssueReader:
		a = actions.NewGithubIssueReader(config)
	case ActionGithubPRReader:
		a = actions.NewGithubPRReader(config)
	case ActionGithubPRCommenter:
		a = actions.NewGithubPRCommenter(config)
	case ActionGithubPRReviewer:
		a = actions.NewGithubPRReviewer(config)
	case ActionGithubPRCreator:
		a = actions.NewGithubPRCreator(config)
	case ActionGithubGetAllContent:
		a = actions.NewGithubRepositoryGetAllContent(config)
	case ActionGithubIssueCommenter:
		a = actions.NewGithubIssueCommenter(config)
	case ActionGithubRepositoryGet:
		a = actions.NewGithubRepositoryGetContent(config)
	case ActionGithubRepositoryCreateOrUpdate:
		a = actions.NewGithubRepositoryCreateOrUpdateContent(config)
	case ActionGithubREADME:
		a = actions.NewGithubRepositoryREADME(config)
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
		a = actions.NewCallAgent(config, agentName, pool.InternalAPI())
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

func ActionsConfigMeta() []config.FieldGroup {
	return []config.FieldGroup{
		{
			Name:   "search",
			Label:  "Search",
			Fields: actions.SearchConfigMeta(),
		},
		{
			Name:   "browser-agent-runner",
			Label:  "Browser Agent Runner",
			Fields: actions.BrowserAgentRunnerConfigMeta(),
		},
		{
			Name:   "deep-research-runner",
			Label:  "Deep Research Runner",
			Fields: actions.DeepResearchRunnerConfigMeta(),
		},
		{
			Name:   "generate_image",
			Label:  "Generate Image",
			Fields: actions.GenImageConfigMeta(),
		},
		{
			Name:   "github-issue-labeler",
			Label:  "GitHub Issue Labeler",
			Fields: actions.GithubIssueLabelerConfigMeta(),
		},
		{
			Name:   "github-issue-opener",
			Label:  "GitHub Issue Opener",
			Fields: actions.GithubIssueOpenerConfigMeta(),
		},
		{
			Name:   "github-issue-closer",
			Label:  "GitHub Issue Closer",
			Fields: actions.GithubIssueCloserConfigMeta(),
		},
		{
			Name:   "github-issue-commenter",
			Label:  "GitHub Issue Commenter",
			Fields: actions.GithubIssueCommenterConfigMeta(),
		},
		{
			Name:   "github-issue-reader",
			Label:  "GitHub Issue Reader",
			Fields: actions.GithubIssueReaderConfigMeta(),
		},
		{
			Name:   "github-issue-searcher",
			Label:  "GitHub Issue Search",
			Fields: actions.GithubIssueSearchConfigMeta(),
		},
		{
			Name:   "github-repository-get-content",
			Label:  "GitHub Repository Get Content",
			Fields: actions.GithubRepositoryGetContentConfigMeta(),
		},
		{
			Name:   "github-get-all-repository-content",
			Label:  "GitHub Get All Repository Content",
			Fields: actions.GithubRepositoryGetAllContentConfigMeta(),
		},
		{
			Name:   "github-repository-create-or-update-content",
			Label:  "GitHub Repository Create/Update Content",
			Fields: actions.GithubRepositoryCreateOrUpdateContentConfigMeta(),
		},
		{
			Name:   "github-readme",
			Label:  "GitHub Repository README",
			Fields: actions.GithubRepositoryREADMEConfigMeta(),
		},
		{
			Name:   "github-pr-reader",
			Label:  "GitHub PR Reader",
			Fields: actions.GithubPRReaderConfigMeta(),
		},
		{
			Name:   "github-pr-commenter",
			Label:  "GitHub PR Commenter",
			Fields: actions.GithubPRCommenterConfigMeta(),
		},
		{
			Name:   "github-pr-reviewer",
			Label:  "GitHub PR Reviewer",
			Fields: actions.GithubPRReviewerConfigMeta(),
		},
		{
			Name:   "github-pr-creator",
			Label:  "GitHub PR Creator",
			Fields: actions.GithubPRCreatorConfigMeta(),
		},
		{
			Name:   "twitter-post",
			Label:  "Twitter Post",
			Fields: actions.TwitterPostConfigMeta(),
		},
		{
			Name:   "send-mail",
			Label:  "Send Mail",
			Fields: actions.SendMailConfigMeta(),
		},
		{
			Name:   "shell-command",
			Label:  "Shell Command",
			Fields: actions.ShellConfigMeta(),
		},
		{
			Name:   "custom",
			Label:  "Custom",
			Fields: action.CustomConfigMeta(),
		},
		{
			Name:   "scraper",
			Label:  "Scraper",
			Fields: []config.Field{},
		},
		{
			Name:   "wikipedia",
			Label:  "Wikipedia",
			Fields: []config.Field{},
		},
		{
			Name:   "browse",
			Label:  "Browse",
			Fields: []config.Field{},
		},
		{
			Name:   "counter",
			Label:  "Counter",
			Fields: []config.Field{},
		},
		{
			Name:   "call_agents",
			Label:  "Call Agents",
			Fields: []config.Field{},
		},
	}
}
