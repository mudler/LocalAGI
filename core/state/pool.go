package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	"github.com/sashabaranov/go-openai"

	models "github.com/mudler/LocalAGI/dbmodels"

	"github.com/google/uuid"
	"github.com/mudler/LocalAGI/pkg/xlog"
)

type AgentPool struct {
	sync.Mutex
	userId                               string
	file                                 string
	pooldir                              string
	pool                                 AgentPoolData
	agents                               map[string]*Agent
	managers                             map[string]sse.Manager
	agentStatus                          map[string]*Status
	defaultModel, defaultMultimodalModel string
	imageModel, localRAGAPI, localRAGKey string
	availableActions                     func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action
	connectors                           func(*AgentConfig) []Connector
	dynamicPrompt                        func(*AgentConfig) []DynamicPrompt
	timeout                              string
	conversationLogs                     string
	filters                              func(*AgentConfig) types.JobFilters
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
	userId, defaultModel, defaultMultimodalModel, imageModel,
	LocalRAGAPI string,
	availableActions func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action,
	connectors func(*AgentConfig) []Connector,
	promptBlocks func(*AgentConfig) []DynamicPrompt,
	filters func(*AgentConfig) types.JobFilters,
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
		defaultModel:           defaultModel,
		defaultMultimodalModel: defaultMultimodalModel,
		imageModel:             imageModel,
		localRAGAPI:            LocalRAGAPI,
		agents:                 make(map[string]*Agent),
		pool:                   poolMap,
		agentStatus:            make(map[string]*Status),
		managers:               make(map[string]sse.Manager),
		connectors:             connectors,
		filters:                filters,
		availableActions:       availableActions,
		dynamicPrompt:          promptBlocks,
		timeout:                timeout,
		conversationLogs:       conversationPath,
	}, nil
}

func NewEmptyAgentPool(
	userId, defaultModel, defaultMultimodalModel, imageModel,
	localRAGAPI string,
	availableActions func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action,
	connectors func(*AgentConfig) []Connector,
	promptBlocks func(*AgentConfig) []DynamicPrompt,
	filters func(*AgentConfig) types.JobFilters,
	timeout string,
	withLogs bool,
) *AgentPool {
	var conversationPath string
	if withLogs {
		conversationPath = fmt.Sprintf("/tmp/%s/conversations", userId)
		_ = os.MkdirAll(conversationPath, 0755)
	}

	return &AgentPool{
		userId:                 userId,
		defaultModel:           defaultModel,
		defaultMultimodalModel: defaultMultimodalModel,
		imageModel:             imageModel,
		localRAGAPI:            localRAGAPI,
		agents:                 make(map[string]*Agent),
		pool:                   make(map[string]AgentConfig),
		agentStatus:            make(map[string]*Status),
		managers:               make(map[string]sse.Manager),
		connectors:             connectors,
		availableActions:       availableActions,
		dynamicPrompt:          promptBlocks,
		timeout:                timeout,
		conversationLogs:       conversationPath,
	}
}

func replaceInvalidChars(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	return strings.ReplaceAll(s, " ", "_")
}

// CreateAgent adds a new agent to the pool
// and starts it.
// It also saves the state to the file.
func (a *AgentPool) CreateAgent(id string, agentConfig *AgentConfig) error {
	id = replaceInvalidChars(id)
	agentConfig.Name = id

	a.Lock()
	defer a.Unlock()

	// Check if agent already exists
	if existingAgent := a.agents[id]; existingAgent != nil {
		return fmt.Errorf("agent %s already exists", id)
	}

	// Insert into the pool
	a.pool[id] = *agentConfig

	return a.startAgentWithConfig(id, agentConfig, nil)
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

func (a *AgentPool) startAgentWithConfig(id string, config *AgentConfig, obs Observer) error {
	var manager sse.Manager
	if m, ok := a.managers[id]; ok {
		manager = m
	} else {
		manager = sse.NewManager(5)
	}
	ctx := context.Background()
	model := a.defaultModel
	multimodalModel := a.defaultMultimodalModel

	if config.MultimodalModel != "" {
		multimodalModel = config.MultimodalModel
	}

	if config.Model != "" {
		model = config.Model
	} else {
		config.Model = model
	}

	if config.PeriodicRuns == "" {
		config.PeriodicRuns = "10m"
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
	filters := a.filters(config)

	actionsLog := []string{}
	for _, action := range actions {
		actionsLog = append(actionsLog, action.Definition().Name.String())
	}

	connectorLog := []string{}
	for _, connector := range connectors {
		connectorLog = append(connectorLog, fmt.Sprintf("%+v", connector))
	}

	filtersLog := []string{}
	for _, filter := range filters {
		filtersLog = append(filtersLog, filter.Name())
	}

	xlog.Info(
		"Creating agent",
		"id", id,
		"model", model,
		"actions", actionsLog,
		"connectors", connectorLog,
		"filters", filtersLog,
	)

	// dynamicPrompts := []map[string]string{}
	// for _, p := range config.DynamicPrompts {
	// 	dynamicPrompts = append(dynamicPrompts, p.ToMap())
	// }

	fmt.Printf("DEBUG: Creating agent with config - RandomIdentity: %v, IdentityGuidance: '%s'\n", config.RandomIdentity, config.IdentityGuidance)
	if obs == nil {
		obs = NewSSEObserver(id, manager)
	}

	opts := []Option{
		WithModel(model),
		WithContext(ctx),
		WithMCPServers(config.MCPServers...),
		WithPeriodicRuns(config.PeriodicRuns),
		WithPermanentGoal(config.PermanentGoal),
		WithPrompts(promptBlocks...),
		WithJobFilters(filters...),
		//	WithDynamicPrompts(dynamicPrompts...),
		WithCharacter(Character{
			Name: id,
		}),
		WithActions(
			actions...,
		),
		WithTimeout(a.timeout),
		WithRAGDB(localrag.NewWrappedClient(a.localRAGAPI, a.localRAGKey, id)),
		WithUserID(uuid.MustParse(a.userId)),
		WithAgentID(uuid.MustParse(id)),
		WithNewConversationSubscriber(func(msg openai.ChatCompletionMessage) {
			// Route reminder and other new conversation messages through SSE
			messageID := fmt.Sprintf("reminder-%d", time.Now().UnixNano())
			data := map[string]interface{}{
				"id":        messageID,
				"sender":    "agent",
				"content":   msg.Content,
				"createdAt": time.Now().Format(time.RFC3339),
			}
			jsonData, err := json.Marshal(data)
			if err != nil {
				xlog.Error("Error marshaling reminder message", "error", err)
				return
			}
			manager.Send(
				sse.NewMessage(string(jsonData)).WithEvent("json_message"),
			)
		}),
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
		WithLLMAPIURL(os.Getenv("LOCALAGI_LLM_API_URL")),
		WithLLMAPIKey(os.Getenv("LOCALAGI_LLM_API_KEY")),
		WithObserver(obs),
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
		fmt.Printf("DEBUG: RandomIdentity is enabled, IdentityGuidance: '%s'\n", config.IdentityGuidance)
		if config.IdentityGuidance != "" {
			opts = append(opts, WithRandomIdentity(config.IdentityGuidance))
			fmt.Printf("DEBUG: Added WithRandomIdentity with guidance, total opts count: %d\n", len(opts))
		} else {
			opts = append(opts, WithRandomIdentity())
			fmt.Printf("DEBUG: Added WithRandomIdentity without guidance, total opts count: %d\n", len(opts))
		}
	} else {
		fmt.Printf("DEBUG: RandomIdentity is disabled\n")
	}

	if config.EnableKnowledgeBase {
		if config.KnowledgeBaseResults > 0 {
			opts = append(opts, EnableKnowledgeBaseWithResults(config.KnowledgeBaseResults))
		} else {
			opts = append(opts, EnableKnowledgeBase)
		}
	}

	if config.EnableReasoning {
		opts = append(opts, EnableForceReasoning)
	}

	if config.StripThinkingTags {
		opts = append(opts, EnableStripThinkingTags)
	}

	if config.LoopDetectionSteps > 0 {
		opts = append(opts, WithLoopDetectionSteps(config.LoopDetectionSteps))
	}

	opts = append(opts, WithMySQLForSummaries())

	if config.ParallelJobs > 0 {
		opts = append(opts, WithParallelJobs(config.ParallelJobs))
	}

	if config.EnableEvaluation {
		opts = append(opts, EnableEvaluation())
		if config.MaxEvaluationLoops > 0 {
			opts = append(opts, WithMaxEvaluationLoops(config.MaxEvaluationLoops))
		}
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
		if err := a.startAgentWithConfig(id, &config, nil); err != nil {
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
		return a.startAgentWithConfig(id, &config, nil)
	}

	return fmt.Errorf("agent %s not found", id)
}

func (a *AgentPool) Remove(id string) error {
	a.Lock()
	defer a.Unlock()

	// Stop the running agent
	a.stop(id)

	// Remove from in-memory maps
	delete(a.agents, id)
	delete(a.pool, id)
	delete(a.agentStatus, id)
	delete(a.managers, id)

	xlog.Info("Removed agent from memory", "id", id)
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

func (a *AgentPool) IsAgentActive(id string) bool {
	a.Lock()
	defer a.Unlock()
	if agent := a.agents[id]; agent != nil {
		return !agent.Paused()
	}
	return false
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

// SetManager sets the SSE manager for a specific agent
func (a *AgentPool) SetManager(id string, manager sse.Manager) {
	a.Lock()
	defer a.Unlock()
	a.managers[id] = manager
}

// GetUserID returns the user ID for this agent pool
func (a *AgentPool) GetUserID() string {
	return a.userId
}

// RemoveAgentOnly removes only the agent instance without touching the manager or other data
func (a *AgentPool) RemoveAgentOnly(id string) {
	a.Lock()
	defer a.Unlock()

	// Remove only the agent instance, keep manager, pool config, and status
	delete(a.agents, id)

	xlog.Info("Removed agent instance only", "id", id)
}

// CreateAgentWithExistingManager creates an agent but reuses the existing SSE manager
func (a *AgentPool) CreateAgentWithExistingManager(id string, agentConfig *AgentConfig, notStart bool) error {
	id = replaceInvalidChars(id)
	agentConfig.Name = id

	a.Lock()

	// Check if agent already exists
	if existingAgent := a.agents[id]; existingAgent != nil {
		if notStart {
			// Remove only the agent instance, keep manager, pool config, and status
			delete(a.agents, id)
		} else {
			a.Unlock()
			return fmt.Errorf("agent %s already exists", id)
		}
	}

	// Insert into the pool
	a.pool[id] = *agentConfig

	// Get existing manager before starting agent
	existingManager := a.managers[id]

	a.Unlock()

	oldAgent := a.agents[id]
	var o *types.Observable
	var obs Observer
	if oldAgent != nil {
		obs = oldAgent.Observer()
		if obs != nil {
			o = obs.NewObservable()
			o.Name = "Restarting Agent"
			o.Icon = "sync"
			o.Creation = &types.Creation{}
			obs.Update(*o)
		}
	}

	// Start agent (this will create a new manager)
	err := a.startAgentWithConfig(id, agentConfig, obs)
	if err != nil {
		if obs != nil && o != nil {
			o.Completion = &types.Completion{Error: err.Error()}
			obs.Update(*o)
		}
		return err
	}

	// Restore the existing manager if it exists
	if existingManager != nil {
		a.Lock()
		a.managers[id] = existingManager
		a.Unlock()
		xlog.Info("Restored existing SSE manager", "id", id)
	}

	if obs != nil && o != nil {
		o.Completion = &types.Completion{}
		obs.Update(*o)
	}

	return nil
}
