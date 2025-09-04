package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	ActionGithubIssueEditor              = "github-issue-editor"
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
	ActionGithubRepositorySearchFiles    = "github-repository-search-files"
	ActionGithubRepositoryListFiles      = "github-repository-list-files"
	ActionScraper                        = "scraper"
	ActionWikipedia                      = "wikipedia"
	ActionBrowse                         = "browse"
	ActionTwitterPost                    = "twitter-post"
	ActionSendMail                       = "send-mail"
	ActionGenerateImage                  = "generate_image"
	ActionCounter                        = "counter"
	ActionCallAgents                     = "call_agents"
	ActionShellcommand                   = "shell-command"
	ActionSendTelegramMessage            = "send-telegram-message"
	ActionSetReminder                    = "set_reminder"
	ActionListReminders                  = "list_reminders"
	ActionRemoveReminder                 = "remove_reminder"
	ActionAddToMemory                    = "add_to_memory"
	ActionListMemory                     = "list_memory"
	ActionRemoveFromMemory               = "remove_from_memory"
)

var AvailableActions = []string{
	ActionSearch,
	ActionCustom,
	ActionGithubIssueLabeler,
	ActionGithubIssueOpener,
	ActionGithubIssueEditor,
	ActionGithubIssueCloser,
	ActionGithubIssueSearcher,
	ActionGithubRepositoryGet,
	ActionGithubGetAllContent,
	ActionGithubRepositorySearchFiles,
	ActionGithubRepositoryListFiles,
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
	ActionSendTelegramMessage,
	ActionSetReminder,
	ActionListReminders,
	ActionRemoveReminder,
	ActionAddToMemory,
	ActionListMemory,
	ActionRemoveFromMemory,
}

var DefaultActions = []config.FieldGroup{
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
		Name:   "add_to_memory",
		Label:  "Add to Memory",
		Fields: actions.AddToMemoryConfigMeta(),
	},
	{
		Name:   "list_memory",
		Label:  "List Memory",
		Fields: actions.ListMemoryConfigMeta(),
	},
	{
		Name:   "remove_from_memory",
		Label:  "Remove from Memory",
		Fields: actions.RemoveFromMemoryConfigMeta(),
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
		Name:   "github-issue-editor",
		Label:  "GitHub Issue Editor",
		Fields: actions.GithubIssueEditorConfigMeta(),
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
		Name:   "github-repository-search-files",
		Label:  "GitHub Repository Search Files",
		Fields: actions.GithubRepositorySearchFilesConfigMeta(),
	},
	{
		Name:   "github-repository-list-files",
		Label:  "GitHub Repository List Files",
		Fields: actions.GithubRepositoryListFilesConfigMeta(),
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
		Fields: actions.CallAgentConfigMeta(),
	},
	{
		Name:   "send-telegram-message",
		Label:  "Send Telegram Message",
		Fields: actions.SendTelegramMessageConfigMeta(),
	},
	{
		Name:   "set_reminder",
		Label:  "Set Reminder",
		Fields: []config.Field{},
	},
	{
		Name:   "list_reminders",
		Label:  "List Reminders",
		Fields: []config.Field{},
	},
	{
		Name:   "remove_reminder",
		Label:  "Remove Reminder",
		Fields: []config.Field{},
	},
}

const (
	ActionConfigBrowserAgentRunner = "browser-agent-runner-base-url"
	ActionConfigDeepResearchRunner = "deep-research-runner-base-url"
	ActionConfigSSHBoxURL          = "sshbox-url"
	ConfigStateDir                 = "state-dir"
)

func CustomActions(customActionsDir string) (allActions []types.Action) {
	files, err := os.ReadDir(customActionsDir)
	if err != nil {
		xlog.Error("Error reading custom actions directory", "error", err)
		return allActions
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".go" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(customActionsDir, file.Name()))
		if err != nil {
			xlog.Error("Error reading custom action file", "error", err, "file", file.Name())
			continue
		}
		a, err := Action(ActionCustom, "", map[string]string{
			"name":   strings.TrimSuffix(file.Name(), ".go"),
			"code":   string(content),
			"unsafe": "false",
		}, nil, map[string]string{})
		if err != nil {
			xlog.Error("Error creating custom action", "error", err, "file", file.Name())
			continue
		}
		allActions = append(allActions, a)
	}
	return
}

func Actions(actionsConfigs map[string]string, customActionsDir string) func(a *state.AgentConfig) func(ctx context.Context, pool *state.AgentPool) []types.Action {
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

			// Now we will scan a directory for custom actions
			if customActionsDir != "" {
				allActions = append(allActions, CustomActions(customActionsDir)...)
			}

			return allActions
		}
	}

}

func Action(name, agentName string, config map[string]string, pool *state.AgentPool, actionsConfigs map[string]string) (types.Action, error) {
	var a types.Action
	var err error

	if config == nil {
		config = map[string]string{}
	}

	memoryFilePath := memoryPath(agentName, actionsConfigs)

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
	case ActionGithubIssueEditor:
		a = actions.NewGithubIssueEditor(config)
	case ActionGithubIssueCloser:
		a = actions.NewGithubIssueCloser(config)
	case ActionGithubIssueSearcher:
		a = actions.NewGithubIssueSearch(config)
	case ActionBrowserAgentRunner:
		a = actions.NewBrowserAgentRunner(config, actionsConfigs[ActionConfigBrowserAgentRunner])
	case ActionDeepResearchRunner:
		a = actions.NewDeepResearchRunner(config, actionsConfigs[ActionConfigDeepResearchRunner])
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
	case ActionGithubRepositorySearchFiles:
		a = actions.NewGithubRepositorySearchFiles(config)
	case ActionGithubRepositoryListFiles:
		a = actions.NewGithubRepositoryListFiles(config)
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
		a = actions.NewShell(config, actionsConfigs[ActionConfigSSHBoxURL])
	case ActionSendTelegramMessage:
		a = actions.NewSendTelegramMessageRunner(config)
	case ActionSetReminder:
		a = action.NewReminder()
	case ActionListReminders:
		a = action.NewListReminders()
	case ActionRemoveReminder:
		a = action.NewRemoveReminder()
	case ActionAddToMemory:
		a, _, _ = actions.NewMemoryActions(memoryFilePath, config)
	case ActionListMemory:
		_, a, _ = actions.NewMemoryActions(memoryFilePath, config)
	case ActionRemoveFromMemory:
		_, _, a = actions.NewMemoryActions(memoryFilePath, config)
	default:
		xlog.Error("Action not found", "name", name)
		return nil, fmt.Errorf("Action not found")
	}

	if err != nil {
		return nil, err
	}

	return a, nil
}

func ActionsConfigMeta(customActionDir string) []config.FieldGroup {
	all := slices.Clone(DefaultActions)

	if customActionDir != "" {
		actions := CustomActions(customActionDir)

		for _, a := range actions {
			all = append(all, config.FieldGroup{
				Name:  a.Definition().Name.String(),
				Label: a.Definition().Name.String(),
				Fields: []config.Field{
					{
						Name:     "name",
						Label:    "Name",
						Type:     config.FieldTypeText,
						HelpText: "Name of the custom action",
					},
					{
						Name:     "description",
						Label:    "Description",
						Type:     config.FieldTypeTextarea,
						HelpText: "Description of the custom action",
					},
				},
			})
		}
	}
	return all
}
