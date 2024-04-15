package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mudler/local-agent-framework/example/webui/connector"

	. "github.com/mudler/local-agent-framework/agent"
	"github.com/mudler/local-agent-framework/external"
)

type ConnectorConfig struct {
	Type   string `json:"type"` // e.g. Slack
	Config string `json:"config"`
}

type ActionsConfig struct {
	Name   string `json:"name"` // e.g. search
	Config string `json:"config"`
}

type AgentConfig struct {
	Connector []ConnectorConfig `json:"connectors" form:"connectors" `
	Actions   []ActionsConfig   `json:"actions" form:"actions"`
	// This is what needs to be part of ActionsConfig
	Model                 string `json:"model" form:"model"`
	Name                  string `json:"name" form:"name"`
	HUD                   bool   `json:"hud" form:"hud"`
	StandaloneJob         bool   `json:"standalone_job" form:"standalone_job"`
	RandomIdentity        bool   `json:"random_identity" form:"random_identity"`
	InitiateConversations bool   `json:"initiate_conversations" form:"initiate_conversations"`
	IdentityGuidance      string `json:"identity_guidance" form:"identity_guidance"`
	PeriodicRuns          string `json:"periodic_runs" form:"periodic_runs"`
	PermanentGoal         string `json:"permanent_goal" form:"permanent_goal"`
	EnableKnowledgeBase   bool   `json:"enable_kb" form:"enable_kb"`
	KnowledgeBaseResults  int    `json:"kb_results" form:"kb_results"`
	CanStopItself         bool   `json:"can_stop_itself" form:"can_stop_itself"`
	SystemPrompt          string `json:"system_prompt" form:"system_prompt"`
}

type AgentPool struct {
	sync.Mutex
	file          string
	pooldir       string
	pool          AgentPoolData
	agents        map[string]*Agent
	managers      map[string]Manager
	agentStatus   map[string]*Status
	apiURL, model string
	ragDB         RAGDB
}

type Status struct {
	results []ActionState
}

func (s *Status) addResult(result ActionState) {
	// If we have more than 10 results, remove the oldest one
	if len(s.results) > 10 {
		s.results = s.results[1:]
	}

	s.results = append(s.results, result)
}

func (s *Status) Results() []ActionState {
	return s.results
}

type AgentPoolData map[string]AgentConfig

func loadPoolFromFile(path string) (*AgentPoolData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	poolData := &AgentPoolData{}
	err = json.Unmarshal(data, poolData)
	return poolData, err
}

func NewAgentPool(model, apiURL, directory string, RagDB RAGDB) (*AgentPool, error) {
	// if file exists, try to load an existing pool.
	// if file does not exist, create a new pool.

	poolfile := filepath.Join(directory, "pool.json")

	if _, err := os.Stat(poolfile); err != nil {
		// file does not exist, create a new pool
		return &AgentPool{
			file:        poolfile,
			pooldir:     directory,
			apiURL:      apiURL,
			model:       model,
			ragDB:       RagDB,
			agents:      make(map[string]*Agent),
			pool:        make(map[string]AgentConfig),
			agentStatus: make(map[string]*Status),
			managers:    make(map[string]Manager),
		}, nil
	}

	poolData, err := loadPoolFromFile(poolfile)
	if err != nil {
		return nil, err
	}
	return &AgentPool{
		file:        poolfile,
		apiURL:      apiURL,
		pooldir:     directory,
		ragDB:       RagDB,
		model:       model,
		agents:      make(map[string]*Agent),
		managers:    make(map[string]Manager),
		agentStatus: map[string]*Status{},
		pool:        *poolData,
	}, nil
}

// CreateAgent adds a new agent to the pool
// and starts it.
// It also saves the state to the file.
func (a *AgentPool) CreateAgent(name string, agentConfig *AgentConfig) error {
	if _, ok := a.pool[name]; ok {
		return fmt.Errorf("agent %s already exists", name)
	}
	a.pool[name] = *agentConfig
	if err := a.Save(); err != nil {
		return err
	}

	return a.startAgentWithConfig(name, agentConfig)
}

func (a *AgentPool) List() []string {
	var agents []string
	for agent := range a.pool {
		agents = append(agents, agent)
	}
	return agents
}

const (
	// Connectors
	ConnectorTelegram     = "telegram"
	ConnectorSlack        = "slack"
	ConnectorDiscord      = "discord"
	ConnectorGithubIssues = "github-issues"

	// Actions
	ActionSearch              = "search"
	ActionGithubIssueLabeler  = "github-issue-labeler"
	ActionGithubIssueOpener   = "github-issue-opener"
	ActionGithubIssueCloser   = "github-issue-closer"
	ActionGithubIssueSearcher = "github-issue-searcher"
)

var AvailableActions = []string{
	ActionSearch,
	ActionGithubIssueLabeler,
	ActionGithubIssueOpener,
	ActionGithubIssueCloser,
	ActionGithubIssueSearcher,
}

func (a *AgentConfig) availableActions(ctx context.Context) []Action {
	actions := []Action{}

	for _, action := range a.Actions {
		slog.Info("Set Action", action)

		var config map[string]string
		if err := json.Unmarshal([]byte(action.Config), &config); err != nil {
			slog.Info("Error unmarshalling action config", err)
			continue
		}
		slog.Info("Config", config)

		switch action.Name {
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
		}
	}

	return actions
}

type Connector interface {
	AgentResultCallback() func(state ActionState)
	AgentReasoningCallback() func(state ActionCurrentState) bool
	Start(a *Agent)
}

var AvailableConnectors = []string{
	ConnectorTelegram,
	ConnectorSlack,
	ConnectorDiscord,
	ConnectorGithubIssues,
}

func (a *AgentConfig) availableConnectors() []Connector {
	connectors := []Connector{}

	for _, c := range a.Connector {
		slog.Info("Set Connector", c)

		var config map[string]string
		if err := json.Unmarshal([]byte(c.Config), &config); err != nil {
			slog.Info("Error unmarshalling connector config", err)
			continue
		}
		slog.Info("Config", config)

		switch c.Type {
		case ConnectorTelegram:
			cc, err := connector.NewTelegramConnector(config)
			if err != nil {
				slog.Info("Error creating telegram connector", err)
				continue
			}

			connectors = append(connectors, cc)
		case ConnectorSlack:
			connectors = append(connectors, connector.NewSlack(config))
		case ConnectorDiscord:
			connectors = append(connectors, connector.NewDiscord(config))
		case ConnectorGithubIssues:
			connectors = append(connectors, connector.NewGithubIssueWatcher(config))
		}
	}
	return connectors
}

func (a *AgentPool) GetStatusHistory(name string) *Status {
	a.Lock()
	defer a.Unlock()
	return a.agentStatus[name]
}

func (a *AgentPool) startAgentWithConfig(name string, config *AgentConfig) error {
	manager := NewManager(5)
	ctx := context.Background()
	model := a.model
	if config.Model != "" {
		model = config.Model
	}
	if config.PeriodicRuns == "" {
		config.PeriodicRuns = "10m"
	}

	connectors := config.availableConnectors()

	slog.Info("Creating agent", name)
	slog.Info("Model", model)
	slog.Info("API URL", a.apiURL)

	actions := config.availableActions(ctx)

	stateFile, characterFile := a.stateFiles(name)

	slog.Info("Actions", actions)
	opts := []Option{
		WithModel(model),
		WithLLMAPIURL(a.apiURL),
		WithContext(ctx),
		WithPeriodicRuns(config.PeriodicRuns),
		WithPermanentGoal(config.PermanentGoal),
		WithActions(
			actions...,
		),
		WithStateFile(stateFile),
		WithCharacterFile(characterFile),
		WithRAGDB(a.ragDB),
		WithAgentReasoningCallback(func(state ActionCurrentState) bool {
			slog.Info("Reasoning", state.Reasoning)
			manager.Send(
				NewMessage(
					fmt.Sprintf(`Thinking: %s`, htmlIfy(state.Reasoning)),
				).WithEvent("status"),
			)

			for _, c := range connectors {
				if !c.AgentReasoningCallback()(state) {
					return false
				}
			}
			return true
		}),
		WithSystemPrompt(config.SystemPrompt),
		WithAgentResultCallback(func(state ActionState) {
			a.Lock()
			if _, ok := a.agentStatus[name]; !ok {
				a.agentStatus[name] = &Status{}
			}

			a.agentStatus[name].addResult(state)
			a.Unlock()
			slog.Info("Reasoning", state.Reasoning)

			text := fmt.Sprintf(`Reasoning: %s
			Action taken: %+v
			Parameters: %+v
			Result: %s`,
				state.Reasoning,
				state.ActionCurrentState.Action.Definition().Name,
				state.ActionCurrentState.Params,
				state.Result)
			manager.Send(
				NewMessage(
					htmlIfy(
						text,
					),
				).WithEvent("status"),
			)

			for _, c := range connectors {
				c.AgentResultCallback()(state)
			}
		}),
	}
	if config.HUD {
		opts = append(opts, EnableHUD)
	}
	if os.Getenv("DEBUG") != "" {
		opts = append(opts, LogLevel(slog.LevelDebug))
	}
	if config.StandaloneJob {
		opts = append(opts, EnableStandaloneJob)
	}
	if config.CanStopItself {
		opts = append(opts, CanStopItself)
	}
	if config.InitiateConversations {
		opts = append(opts, EnableInitiateConversations)
	}
	if config.RandomIdentity {
		if config.IdentityGuidance != "" {
			opts = append(opts, WithRandomIdentity(config.IdentityGuidance))
		} else {
			opts = append(opts, WithRandomIdentity())
		}
	}

	if config.EnableKnowledgeBase {
		opts = append(opts, EnableKnowledgeBase)
	}

	if config.KnowledgeBaseResults > 0 {
		opts = append(opts, EnableKnowledgeBaseWithResults(config.KnowledgeBaseResults))
	}

	slog.Info("Starting agent", "name", name, "config", config)
	agent, err := New(opts...)
	if err != nil {
		return err
	}

	a.agents[name] = agent
	a.managers[name] = manager

	go func() {
		if err := agent.Run(); err != nil {
			slog.Info("Agent stop: ", err.Error())
		}
	}()

	for _, c := range connectors {
		go c.Start(agent)
	}

	go func() {
		for {
			time.Sleep(1 * time.Second) // Send a message every seconds
			manager.Send(NewMessage(
				htmlIfy(agent.State().String()),
			).WithEvent("hud"))
		}
	}()

	return nil
}

// Starts all the agents in the pool
func (a *AgentPool) StartAll() error {
	for name, config := range a.pool {
		if a.agents[name] != nil { // Agent already started
			continue
		}
		if err := a.startAgentWithConfig(name, &config); err != nil {
			return err
		}
	}
	return nil
}

func (a *AgentPool) StopAll() {
	for _, agent := range a.agents {
		agent.Stop()
	}
}

func (a *AgentPool) Stop(name string) {
	if agent, ok := a.agents[name]; ok {
		agent.Stop()
	}
}

func (a *AgentPool) Start(name string) error {
	if agent, ok := a.agents[name]; ok {
		err := agent.Run()
		if err != nil {
			return fmt.Errorf("agent %s failed to start: %w", name, err)
		}
		slog.Info("Agent started", "name", name)
		return nil
	}
	if config, ok := a.pool[name]; ok {
		return a.startAgentWithConfig(name, &config)
	}

	return fmt.Errorf("agent %s not found", name)
}

func (a *AgentPool) stateFiles(name string) (string, string) {
	stateFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.state.json", name))
	characterFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.character.json", name))

	return stateFile, characterFile
}

func (a *AgentPool) Remove(name string) error {

	// Cleanup character and state
	stateFile, characterFile := a.stateFiles(name)

	os.Remove(stateFile)
	os.Remove(characterFile)

	a.Stop(name)
	delete(a.agents, name)
	delete(a.pool, name)
	if err := a.Save(); err != nil {
		return err
	}
	return nil
}

func (a *AgentPool) Save() error {
	data, err := json.MarshalIndent(a.pool, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.file, data, 0644)
}

func (a *AgentPool) GetAgent(name string) *Agent {
	return a.agents[name]
}

func (a *AgentPool) GetConfig(name string) *AgentConfig {
	agent, exists := a.pool[name]
	if !exists {
		return nil
	}
	return &agent
}

func (a *AgentPool) GetManager(name string) Manager {
	return a.managers[name]
}
