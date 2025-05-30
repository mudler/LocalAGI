package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	. "github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/sse"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/db"
	"github.com/mudler/LocalAGI/pkg/localrag"
	"github.com/mudler/LocalAGI/pkg/utils"

	models "github.com/mudler/LocalAGI/dbmodels"

	"github.com/mudler/LocalAGI/pkg/xlog"
)

type AgentPool struct {
	sync.Mutex
	userId                                       string
	file                                         string
	pooldir                                      string
	pool                                         AgentPoolData
	agents                                       map[string]*Agent
	managers                                     map[string]sse.Manager
	agentStatus                                  map[string]*Status
	apiURL, defaultModel, defaultMultimodalModel string
	imageModel, localRAGAPI, localRAGKey, apiKey string
	availableActions                             func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action
	connectors                                   func(*AgentConfig) []Connector
	dynamicPrompt                                func(*AgentConfig) []DynamicPrompt
	timeout                                      string
	conversationLogs                             string
}

type Status struct {
	ActionResults []types.ActionState
}

func (s *Status) addResult(result types.ActionState) {
	// If we have more than 10 results, remove the oldest one
	if len(s.ActionResults) > 10 {
		s.ActionResults = s.ActionResults[1:]
	}

	s.ActionResults = append(s.ActionResults, result)
}

func (s *Status) Results() []types.ActionState {
	return s.ActionResults
}

type AgentPoolData map[string]AgentConfig

// func loadPoolFromFile(path string) (*AgentPoolData, error) {
// 	data, err := os.ReadFile(path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	poolData := &AgentPoolData{}
// 	err = json.Unmarshal(data, poolData)
// 	return poolData, err
// }

func NewAgentPool(
	userId, defaultModel, defaultMultimodalModel, imageModel, apiURL, apiKey string,
	LocalRAGAPI string,
	availableActions func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action,
	connectors func(*AgentConfig) []Connector,
	promptBlocks func(*AgentConfig) []DynamicPrompt,
	timeout string,
	withLogs bool,
) (*AgentPool, error) {
	// 1. Load all agent configs from the DB
	var agents []models.Agent
	if err := db.DB.Where("UserId = ?", userId).Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to load agents from DB: %w", err)
	}

	// 2. Build in-memory config pool
	poolMap := make(map[string]AgentConfig)
	for _, a := range agents {
		var cfg AgentConfig
		if err := json.Unmarshal(a.Config, &cfg); err != nil {
			// Optionally log bad config, skip silently
			continue
		}
		poolMap[a.ID.String()] = cfg
	}

	// 3. Optional conversation log path (e.g., if you're still using file-based logging)
	var conversationPath string
	if withLogs {
		// Replace this with a DB/S3 backend later if desired
		conversationPath = fmt.Sprintf("/tmp/%s/conversations", userId)
		_ = os.MkdirAll(conversationPath, 0755)
	}

	// 4. Return fully initialized pool
	return &AgentPool{
		userId:                 userId,
		apiURL:                 apiURL,
		defaultModel:           defaultModel,
		defaultMultimodalModel: defaultMultimodalModel,
		imageModel:             imageModel,
		localRAGAPI:            LocalRAGAPI,
		apiKey:                 apiKey,
		agents:                 make(map[string]*Agent),
		pool:                   poolMap,
		agentStatus:            make(map[string]*Status),
		managers:               make(map[string]sse.Manager),
		connectors:             connectors,
		availableActions:       availableActions,
		dynamicPrompt:          promptBlocks,
		timeout:                timeout,
		conversationLogs:       conversationPath,
	}, nil
}


func replaceInvalidChars(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	return strings.ReplaceAll(s, " ", "_")
}

// CreateAgent adds a new agent to the pool
// and starts it.
// It also saves the state to the file.
func (a *AgentPool) CreateAgent(id string, agentConfig *AgentConfig) error {
	a.Lock()
	defer a.Unlock()
	id = replaceInvalidChars(id)
	agentConfig.Name = id
	if _, ok := a.pool[id]; ok {
		return fmt.Errorf("agent %s already exists", id)
	}
	a.pool[id] = *agentConfig
	// if err := a.save(); err != nil {
	// 	return err
	// }

	return a.startAgentWithConfig(id, agentConfig)
}

func (a *AgentPool) List() []string {
	a.Lock()
	defer a.Unlock()

	var agents []string
	for agent := range a.pool {
		agents = append(agents, agent)
	}
	// return a sorted list
	sort.SliceStable(agents, func(i, j int) bool {
		return agents[i] < agents[j]
	})
	return agents
}

func (a *AgentPool) GetStatusHistory(id string) *Status {
	a.Lock()
	defer a.Unlock()
	return a.agentStatus[id]
}

func (a *AgentPool) startAgentWithConfig(id string, config *AgentConfig) error {
	manager := sse.NewManager(5)
	ctx := context.Background()
	model := a.defaultModel
	multimodalModel := a.defaultMultimodalModel

	if config.MultimodalModel != "" {
		multimodalModel = config.MultimodalModel
	}

	if config.Model != "" {
		model = config.Model
	}

	if config.PeriodicRuns == "" {
		config.PeriodicRuns = "10m"
	}

	if config.APIURL != "" {
		a.apiURL = config.APIURL
	}

	if config.APIKey != "" {
		a.apiKey = config.APIKey
	}

	if config.LocalRAGURL != "" {
		a.localRAGAPI = config.LocalRAGURL
	}

	if config.LocalRAGAPIKey != "" {
		a.localRAGKey = config.LocalRAGAPIKey
	}

	connectors := a.connectors(config)
	promptBlocks := a.dynamicPrompt(config)
	actions := a.availableActions(config)(ctx, a)
	stateFile, characterFile := a.stateFiles(id)

	actionsLog := []string{}
	for _, action := range actions {
		actionsLog = append(actionsLog, action.Definition().Name.String())
	}

	connectorLog := []string{}
	for _, connector := range connectors {
		connectorLog = append(connectorLog, fmt.Sprintf("%+v", connector))
	}

	xlog.Info(
		"Creating agent",
		"id", id,
		"model", model,
		"api_url", a.apiURL,
		"actions", actionsLog,
		"connectors", connectorLog,
	)

	// dynamicPrompts := []map[string]string{}
	// for _, p := range config.DynamicPrompts {
	// 	dynamicPrompts = append(dynamicPrompts, p.ToMap())
	// }

	opts := []Option{
		WithModel(model),
		WithLLMAPIURL(a.apiURL),
		WithContext(ctx),
		WithMCPServers(config.MCPServers...),
		WithPeriodicRuns(config.PeriodicRuns),
		WithPermanentGoal(config.PermanentGoal),
		WithPrompts(promptBlocks...),
		//	WithDynamicPrompts(dynamicPrompts...),
		WithCharacter(Character{
			Name: id,
		}),
		WithActions(
			actions...,
		),
		WithStateFile(stateFile),
		WithCharacterFile(characterFile),
		WithLLMAPIKey(a.apiKey),
		WithTimeout(a.timeout),
		WithRAGDB(localrag.NewWrappedClient(a.localRAGAPI, a.localRAGKey, id)),
		WithAgentReasoningCallback(func(state types.ActionCurrentState) bool {
			xlog.Info(
				"Agent is thinking",
				"agent", id,
				"reasoning", state.Reasoning,
				"action", state.Action.Definition().Name,
				"params", state.Params,
			)

			manager.Send(
				sse.NewMessage(
					fmt.Sprintf(`Thinking: %s`, utils.HTMLify(state.Reasoning)),
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
		WithMultimodalModel(multimodalModel),
		WithAgentResultCallback(func(state types.ActionState) {
			a.Lock()
			if _, ok := a.agentStatus[id]; !ok {
				a.agentStatus[id] = &Status{}
			}

			a.agentStatus[id].addResult(state)
			a.Unlock()
			xlog.Debug(
				"Calling agent result callback",
			)

			text := fmt.Sprintf(`Reasoning: %s
			Action taken: %+v
			Parameters: %+v
			Result: %s`,
				state.Reasoning,
				state.ActionCurrentState.Action.Definition().Name,
				state.ActionCurrentState.Params,
				state.Result)
			manager.Send(
				sse.NewMessage(
					utils.HTMLify(
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

	if a.conversationLogs != "" {
		opts = append(opts, WithConversationsPath(a.conversationLogs))
	}

	if config.StandaloneJob {
		opts = append(opts, EnableStandaloneJob)
	}

	if config.LongTermMemory {
		opts = append(opts, EnableLongTermMemory)
	}

	if config.SummaryLongTermMemory {
		opts = append(opts, EnableSummaryMemory)
	}

	if config.CanStopItself {
		opts = append(opts, CanStopItself)
	}

	if config.CanPlan {
		opts = append(opts, EnablePlanning)
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

	if config.EnableReasoning {
		opts = append(opts, EnableForceReasoning)
	}

	if config.KnowledgeBaseResults > 0 {
		opts = append(opts, EnableKnowledgeBaseWithResults(config.KnowledgeBaseResults))
	}

	if config.LoopDetectionSteps > 0 {
		opts = append(opts, WithLoopDetectionSteps(config.LoopDetectionSteps))
	}

	xlog.Info("Starting agent", "id", id, "config", config)

	agent, err := New(opts...)
	if err != nil {
		return err
	}

	a.agents[id] = agent
	a.managers[id] = manager

	go func() {
		if err := agent.Run(); err != nil {
			xlog.Error("Agent stopped", "error", err.Error(), "id", id)
		}
	}()

	xlog.Info("Starting connectors", "id", id, "config", config)

	for _, c := range connectors {
		go c.Start(agent)
	}

	go func() {
		for {
			time.Sleep(1 * time.Second) // Send a message every seconds
			manager.Send(sse.NewMessage(
				utils.HTMLify(agent.State().String()),
			).WithEvent("hud"))
		}
	}()

	xlog.Info("Agent started", "id", id)

	return nil
}

// Starts all the agents in the pool
func (a *AgentPool) StartAll() error {
	a.Lock()
	defer a.Unlock()
	for id, config := range a.pool {
		if a.agents[id] != nil { // Agent already started
			continue
		}
		if err := a.startAgentWithConfig(id, &config); err != nil {
			xlog.Error("Failed to start agent", "id", id, "error", err)
		}
	}
	return nil
}

func (a *AgentPool) StopAll() {
	a.Lock()
	defer a.Unlock()
	for _, agent := range a.agents {
		agent.Stop()
	}
}

func (a *AgentPool) Stop(id string) {
	a.Lock()
	defer a.Unlock()
	a.stop(id)
}

func (a *AgentPool) stop(id string) {
	if agent, ok := a.agents[id]; ok {
		agent.Stop()
	}
}
func (a *AgentPool) Start(id string) error {
	a.Lock()
	defer a.Unlock()
	if agent, ok := a.agents[id]; ok {
		err := agent.Run()
		if err != nil {
			return fmt.Errorf("agent %s failed to start: %w", id, err)
		}
		xlog.Info("Agent started", "id", id)
		return nil
	}
	if config, ok := a.pool[id]; ok {
		return a.startAgentWithConfig(id, &config)
	}

	return fmt.Errorf("agent %s not found", id)
}

func (a *AgentPool) stateFiles(id string) (string, string) {
	stateFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.state.json", id))
	characterFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.character.json", id))

	return stateFile, characterFile
}

func (a *AgentPool) Remove(name string) error {
	a.Lock()
	defer a.Unlock()

	// Stop the running agent
	a.stop(name)

	// Remove from in-memory maps
	delete(a.agents, name)
	delete(a.pool, name)
	delete(a.agentStatus, name)
	delete(a.managers, name)

	xlog.Info("Removed agent from memory", "name", name)
	return nil
}


func (a *AgentPool) Save() error {
	a.Lock()
	defer a.Unlock()
	return a.save()
}

func (a *AgentPool) save() error {
	data, err := json.MarshalIndent(a.pool, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.file, data, 0644)
}

func (a *AgentPool) GetAgent(id string) *Agent {
	a.Lock()
	defer a.Unlock()
	return a.agents[id]
}

func (a *AgentPool) AllAgents() []string {
	a.Lock()
	defer a.Unlock()
	var agents []string
	for agent := range a.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (a *AgentPool) GetConfig(id string) *AgentConfig {
	a.Lock()
	defer a.Unlock()
	agent, exists := a.pool[id]
	if !exists {
		return nil
	}
	return &agent
}

func (a *AgentPool) GetManager(id string) sse.Manager {
	a.Lock()
	defer a.Unlock()
	return a.managers[id]
}
