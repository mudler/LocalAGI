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
	"github.com/mudler/LocalAGI/pkg/localrag"
	"github.com/mudler/LocalAGI/pkg/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/xlog"
)

// SkillsProvider supplies the skills dynamic prompt and MCP session when skills are enabled for an agent.
type SkillsProvider interface {
	GetSkillsPrompt(config *AgentConfig) (DynamicPrompt, error)
	GetMCPSession(ctx context.Context) (*mcp.ClientSession, error)
}

// RAGProvider returns a RAGDB and optional compaction client for a collection (e.g. agent name).
// effectiveRAGURL/Key are pool/agent defaults; implementation may use them (HTTP) or ignore them (embedded).
type RAGProvider func(collectionName, effectiveRAGURL, effectiveRAGKey string) (RAGDB, KBCompactionClient, bool)

// NewHTTPRAGProvider returns a RAGProvider that uses the LocalRAG HTTP API. When effective URL/key are empty, baseURL/baseKey are used.
func NewHTTPRAGProvider(baseURL, baseKey string) RAGProvider {
	return func(collectionName, effectiveURL, effectiveKey string) (RAGDB, KBCompactionClient, bool) {
		url := effectiveURL
		if url == "" {
			url = baseURL
		}
		key := effectiveKey
		if key == "" {
			key = baseKey
		}
		wc := localrag.NewWrappedClient(url, key, collectionName)
		return wc, &wrappedClientCompactionAdapter{WrappedClient: wc}, true
	}
}

type AgentPool struct {
	sync.Mutex
	file                                                          string
	pooldir                                                       string
	pool                                                          AgentPoolData
	agents                                                        map[string]*Agent
	managers                                                      map[string]sse.Manager
	agentStatus                                                   map[string]*Status
	apiURL, defaultModel, defaultMultimodalModel, defaultTTSModel string
	defaultTranscriptionModel, defaultTranscriptionLanguage       string
	apiKey                                                        string
	ragProvider                                                   RAGProvider
	availableActions                                              func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action
	connectors                                                    func(*AgentConfig) []Connector
	dynamicPrompt                                                 func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []DynamicPrompt
	filters                                                       func(*AgentConfig) types.JobFilters
	timeout                                                       string
	conversationLogs                                              string
	skillsService                                                 SkillsProvider
}

// SetRAGProvider sets the single RAG provider (HTTP or embedded). Must be called after pool creation.
func (a *AgentPool) SetRAGProvider(fn RAGProvider) {
	a.Lock()
	defer a.Unlock()
	a.ragProvider = fn
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

func loadPoolFromFile(path string) (*AgentPoolData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	poolData := &AgentPoolData{}
	err = json.Unmarshal(data, poolData)
	return poolData, err
}

func NewAgentPool(
	defaultModel, defaultMultimodalModel, defaultTranscriptionModel, defaultTranscriptionLanguage, defaultTTSModel, apiURL, apiKey, directory string,
	availableActions func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action,
	connectors func(*AgentConfig) []Connector,
	promptBlocks func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []DynamicPrompt,
	filters func(*AgentConfig) types.JobFilters,
	timeout string,
	withLogs bool,
	skillsService SkillsProvider,
) (*AgentPool, error) {
	// if file exists, try to load an existing pool.
	// if file does not exist, create a new pool.

	poolfile := filepath.Join(directory, "pool.json")

	conversationPath := ""
	if withLogs {
		conversationPath = filepath.Join(directory, "conversations")
	}
	if _, err := os.Stat(poolfile); err != nil {
		// file does not exist, create a new pool
		return &AgentPool{
			file:                         poolfile,
			pooldir:                      directory,
			apiURL:                       apiURL,
			defaultModel:                 defaultModel,
			defaultMultimodalModel:       defaultMultimodalModel,
			defaultTranscriptionModel:    defaultTranscriptionModel,
			defaultTranscriptionLanguage: defaultTranscriptionLanguage,
			defaultTTSModel:              defaultTTSModel,
			apiKey:                       apiKey,
			agents:                       make(map[string]*Agent),
			pool:                         make(map[string]AgentConfig),
			agentStatus:                  make(map[string]*Status),
			managers:                     make(map[string]sse.Manager),
			connectors:                   connectors,
			availableActions:             availableActions,
			dynamicPrompt:                promptBlocks,
			filters:                      filters,
			timeout:                      timeout,
			conversationLogs:             conversationPath,
			skillsService:                skillsService,
		}, nil
	}

	poolData, err := loadPoolFromFile(poolfile)
	if err != nil {
		bakPath := poolfile + ".bak"
		poolData, err = loadPoolFromFile(bakPath)
		if err != nil {
			xlog.Warn("Pool file invalid and backup missing or invalid, starting with empty pool", "poolfile", poolfile, "error", err)
			poolData = &AgentPoolData{}
		} else {
			xlog.Info("Recovered pool from backup, repairing main file", "poolfile", poolfile)
			if repairData, _ := json.MarshalIndent(poolData, "", "  "); len(repairData) > 0 {
				_ = os.WriteFile(poolfile, repairData, 0644)
			}
		}
	}
	return &AgentPool{
		file:                         poolfile,
		apiURL:                       apiURL,
		pooldir:                      directory,
		defaultModel:                 defaultModel,
		defaultMultimodalModel:       defaultMultimodalModel,
		defaultTranscriptionModel:    defaultTranscriptionModel,
		defaultTranscriptionLanguage: defaultTranscriptionLanguage,
		defaultTTSModel:              defaultTTSModel,
		apiKey:                       apiKey,
		agents:                       make(map[string]*Agent),
		managers:                     make(map[string]sse.Manager),
		agentStatus:                  map[string]*Status{},
		pool:                         *poolData,
		connectors:                   connectors,
		dynamicPrompt:                promptBlocks,
		filters:                      filters,
		availableActions:             availableActions,
		timeout:                      timeout,
		conversationLogs:             conversationPath,
		skillsService:                skillsService,
	}, nil
}

func replaceInvalidChars(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	return strings.ReplaceAll(s, " ", "_")
}

// CreateAgent adds a new agent to the pool
// and starts it.
// It also saves the state to the file.
func (a *AgentPool) CreateAgent(name string, agentConfig *AgentConfig) error {
	a.Lock()
	defer a.Unlock()
	name = replaceInvalidChars(name)
	agentConfig.Name = name
	if _, ok := a.pool[name]; ok {
		return fmt.Errorf("agent %s already exists", name)
	}
	a.pool[name] = *agentConfig
	if err := a.save(); err != nil {
		return err
	}

	return a.startAgentWithConfig(name, a.pooldir, agentConfig, nil)
}

func (a *AgentPool) RecreateAgent(name string, agentConfig *AgentConfig) error {
	a.Lock()
	defer a.Unlock()

	oldAgent := a.agents[name]
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
		stateFile, characterFile := a.stateFiles(name)
		os.Remove(stateFile)
		os.Remove(characterFile)
		oldAgent.Stop()
	}

	a.pool[name] = *agentConfig
	delete(a.agents, name)

	if err := a.save(); err != nil {
		if obs != nil {
			o.Completion = &types.Completion{Error: err.Error()}
			obs.Update(*o)
		}
		return err
	}

	if err := a.startAgentWithConfig(name, a.pooldir, agentConfig, obs); err != nil {
		if obs != nil {
			o.Completion = &types.Completion{Error: err.Error()}
			obs.Update(*o)
		}
		return err
	}

	if obs != nil {
		o.Completion = &types.Completion{}
		obs.Update(*o)
	}

	return nil
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

func (a *AgentPool) GetStatusHistory(name string) *Status {
	a.Lock()
	defer a.Unlock()
	return a.agentStatus[name]
}

func (a *AgentPool) startAgentWithConfig(name, pooldir string, config *AgentConfig, obs Observer) error {
	var manager sse.Manager
	if m, ok := a.managers[name]; ok {
		manager = m
	} else {
		manager = sse.NewManager(5)
	}
	ctx := context.Background()
	model := a.defaultModel
	multimodalModel := a.defaultMultimodalModel
	transcriptionModel := a.defaultTranscriptionModel
	transcriptionLanguage := a.defaultTranscriptionLanguage
	ttsModel := a.defaultTTSModel

	if config.MultimodalModel != "" {
		multimodalModel = config.MultimodalModel
	}

	if config.TranscriptionModel != "" {
		transcriptionModel = config.TranscriptionModel
	}

	if config.TranscriptionLanguage != "" {
		transcriptionLanguage = config.TranscriptionLanguage
	}
	if config.TTSModel != "" {
		ttsModel = config.TTSModel
	}

	if config.Model != "" {
		model = config.Model
	} else {
		config.Model = model
	}

	if config.PeriodicRuns == "" {
		config.PeriodicRuns = "10m"
	}

	if config.SchedulerPollInterval == "" {
		config.SchedulerPollInterval = "30s"
	}

	// Use agent-specific config when set, otherwise pool defaults. Do not update pool from agent config.
	effectiveAPIURL := a.apiURL
	if config.APIURL != "" {
		effectiveAPIURL = config.APIURL
	} else {
		config.APIURL = a.apiURL
	}
	effectiveAPIKey := a.apiKey
	if config.APIKey != "" {
		effectiveAPIKey = config.APIKey
	} else {
		config.APIKey = a.apiKey
	}
	effectiveLocalRAGAPI := config.LocalRAGURL
	effectiveLocalRAGKey := config.LocalRAGAPIKey

	connectors := a.connectors(config)
	promptBlocks := a.dynamicPrompt(config)(ctx, a)
	if a.skillsService != nil && config.EnableSkills {
		if prompt, err := a.skillsService.GetSkillsPrompt(config); err == nil && prompt != nil {
			promptBlocks = append(promptBlocks, prompt)
		}
	}
	actions := a.availableActions(config)(ctx, a)
	filters := a.filters(config)
	stateFile, characterFile := a.stateFiles(name)

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
		"name", name,
		"model", model,
		"api_url", effectiveAPIURL,
		"actions", actionsLog,
		"connectors", connectorLog,
		"filters", filtersLog,
	)

	// dynamicPrompts := []map[string]string{}
	// for _, p := range config.DynamicPrompts {
	// 	dynamicPrompts = append(dynamicPrompts, p.ToMap())
	// }

	if obs == nil {
		obs = NewSSEObserver(name, manager)
	}

	opts := []Option{
		WithSchedulerStorePath(filepath.Join(pooldir, fmt.Sprintf("scheduler-%s.json", name))),
		WithModel(model),
		WithLLMAPIURL(effectiveAPIURL),
		WithContext(ctx),
		WithMCPServers(config.MCPServers...),
		WithTranscriptionModel(transcriptionModel),
		WithTranscriptionLanguage(transcriptionLanguage),
		WithTTSModel(ttsModel),
		WithPeriodicRuns(config.PeriodicRuns),
		WithSchedulerPollInterval(config.SchedulerPollInterval),
		WithPermanentGoal(config.PermanentGoal),
		WithMCPSTDIOServers(config.MCPSTDIOServers...),
		WithPrompts(promptBlocks...),
		WithJobFilters(filters...),
		WithMCPPrepareScript(config.MCPPrepareScript),
		//	WithDynamicPrompts(dynamicPrompts...),
		WithCharacter(Character{
			Name: name,
		}),
		WithActions(
			actions...,
		),
		WithStateFile(stateFile),
		WithCharacterFile(characterFile),
		WithLLMAPIKey(effectiveAPIKey),
		WithTimeout(a.timeout),
		WithAgentReasoningCallback(func(state types.ActionCurrentState) bool {
			var actionName types.ActionDefinitionName
			if state.Action != nil {
				actionName = state.Action.Definition().Name
			}
			xlog.Info(
				"Agent is thinking",
				"agent", name,
				"reasoning", state.Reasoning,
				"action", actionName,
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
		WithInnerMonologueTemplate(config.InnerMonologueTemplate),
		WithSchedulerTaskTemplate(config.SchedulerTaskTemplate),
		WithMultimodalModel(multimodalModel),
		WithLastMessageDuration(config.LastMessageDuration),
		WithAgentResultCallback(func(state types.ActionState) {
			a.Lock()
			if _, ok := a.agentStatus[name]; !ok {
				a.agentStatus[name] = &Status{}
			}

			a.agentStatus[name].addResult(state)
			a.Unlock()
			xlog.Debug(
				"Calling agent result callback",
			)

			var actionName types.ActionDefinitionName
			if state.ActionCurrentState.Action != nil {
				actionName = state.ActionCurrentState.Action.Definition().Name
			}
			text := fmt.Sprintf(`Reasoning: %s
			Action taken: %+v
			Parameters: %+v
			Result: %s`,
				state.Reasoning,
				actionName,
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

	if config.ConversationStorageMode != "" {
		opts = append(opts, WithConversationStorageMode(ConversationStorageMode(config.ConversationStorageMode)))
	}

	if config.CanStopItself {
		opts = append(opts, CanStopItself)
	}

	if config.CanPlan {
		opts = append(opts, EnablePlanning)
	}

	if config.PlanReviewerModel != "" {
		opts = append(opts, WithPlanReviewerLLM(config.PlanReviewerModel))
	}

	if config.DisableSinkState {
		opts = append(opts, DisableSinkState)
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

	if a.skillsService != nil && config.EnableSkills {
		if session, err := a.skillsService.GetMCPSession(ctx); err == nil && session != nil {
			opts = append(opts, WithMCPSession(session))
		}
	}

	var ragDB RAGDB
	var compactionClient KBCompactionClient
	if config.EnableKnowledgeBase && a.ragProvider != nil {
		if db, comp, ok := a.ragProvider(name, effectiveLocalRAGAPI, effectiveLocalRAGKey); ok && db != nil {
			ragDB = db
			compactionClient = comp
		}
	}
	if ragDB != nil {
		opts = append(opts, WithRAGDB(ragDB), EnableKnowledgeBase)
		kbAutoSearch := config.KBAutoSearch
		if !config.KBAutoSearch && !config.KBAsTools {
			kbAutoSearch = true
		}
		opts = append(opts, WithKBAutoSearch(kbAutoSearch))
		if config.KBAsTools {
			kbResults := config.KnowledgeBaseResults
			if kbResults <= 0 {
				kbResults = 5
			}
			searchAction, addAction := NewKBWrapperActions(ragDB, kbResults)
			opts = append(opts, WithActions(searchAction, addAction))
		}
	}

	if config.EnableReasoning {
		opts = append(opts, EnableForceReasoning)
	}

	if config.EnableGuidedTools {
		opts = append(opts, EnableGuidedTools)
	}

	if config.StripThinkingTags {
		opts = append(opts, EnableStripThinkingTags)
	}

	if config.EnableAutoCompaction {
		opts = append(opts, EnableAutoCompaction)
	}

	if config.AutoCompactionThreshold > 0 {
		opts = append(opts, WithAutoCompactionThreshold(config.AutoCompactionThreshold))
	}

	if config.KnowledgeBaseResults > 0 {
		opts = append(opts, EnableKnowledgeBaseWithResults(config.KnowledgeBaseResults))
	}

	if config.ParallelJobs > 0 {
		opts = append(opts, WithParallelJobs(config.ParallelJobs))
	}

	if config.CancelPreviousOnNewMessage != nil {
		opts = append(opts, WithCancelPreviousOnNewMessage(*config.CancelPreviousOnNewMessage))
	} else {
		opts = append(opts, WithCancelPreviousOnNewMessage(true))
	}

	if config.EnableEvaluation {
		opts = append(opts, EnableEvaluation())
	}

	if config.MaxEvaluationLoops > 0 {
		opts = append(opts, WithMaxEvaluationLoops(config.MaxEvaluationLoops))
	}

	if config.MaxAttempts > 0 {
		opts = append(opts, WithMaxAttempts(config.MaxAttempts))
	}

	if config.LoopDetection > 0 {
		opts = append(opts, WithLoopDetection(config.LoopDetection))
	}

	if config.EnableForceReasoningTool {
		opts = append(opts, EnableForceReasoningTool)
	}

	xlog.Info("Starting agent", "name", name, "config", config)

	agent, err := New(opts...)
	if err != nil {
		return err
	}

	a.agents[name] = agent
	a.managers[name] = manager

	go func() {
		if err := agent.Run(); err != nil {
			xlog.Error("Agent stopped", "error", err.Error(), "name", name)
		}
	}()

	if config.EnableKnowledgeBase && config.EnableKBCompaction && compactionClient != nil {
		go runCompactionTicker(ctx, compactionClient, config, effectiveAPIURL, effectiveAPIKey, model)
	}

	xlog.Info("Starting connectors", "name", name, "config", config)

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

	xlog.Info("Agent started", "name", name)

	return nil
}

// Starts all the agents in the pool
func (a *AgentPool) StartAll() error {
	a.Lock()
	defer a.Unlock()
	for name, config := range a.pool {
		if a.agents[name] != nil { // Agent already started
			continue
		}
		if err := a.startAgentWithConfig(name, a.pooldir, &config, nil); err != nil {
			xlog.Error("Failed to start agent", "name", name, "error", err)
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

func (a *AgentPool) Stop(name string) {
	a.Lock()
	defer a.Unlock()
	a.stop(name)
}

func (a *AgentPool) stop(name string) {
	if agent, ok := a.agents[name]; ok {
		agent.Stop()
	}
}
func (a *AgentPool) Start(name string) error {
	a.Lock()
	defer a.Unlock()
	if agent, ok := a.agents[name]; ok {
		err := agent.Run()
		if err != nil {
			return fmt.Errorf("agent %s failed to start: %w", name, err)
		}
		xlog.Info("Agent started", "name", name)
		return nil
	}
	if config, ok := a.pool[name]; ok {
		return a.startAgentWithConfig(name, a.pooldir, &config, nil)
	}

	return fmt.Errorf("agent %s not found", name)
}

func (a *AgentPool) stateFiles(name string) (string, string) {
	stateFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.state.json", name))
	characterFile := filepath.Join(a.pooldir, fmt.Sprintf("%s.character.json", name))

	return stateFile, characterFile
}

func (a *AgentPool) Remove(name string) error {
	a.Lock()
	defer a.Unlock()
	// Cleanup character and state
	stateFile, characterFile := a.stateFiles(name)

	os.Remove(stateFile)
	os.Remove(characterFile)

	a.stop(name)
	delete(a.agents, name)
	delete(a.pool, name)

	if err := a.save(); err != nil {
		return err
	}
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
	tmpPath := a.file + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, a.file); err != nil {
		os.Remove(tmpPath)
		return err
	}
	bakPath := a.file + ".bak"
	if err := os.WriteFile(bakPath, data, 0644); err != nil {
		// best-effort; main file is already good
		xlog.Warn("Failed to write pool backup", "path", bakPath, "error", err)
	}
	return nil
}

func (a *AgentPool) GetAgent(name string) *Agent {
	a.Lock()
	defer a.Unlock()
	return a.agents[name]
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

func (a *AgentPool) GetConfig(name string) *AgentConfig {
	a.Lock()
	defer a.Unlock()
	agent, exists := a.pool[name]
	if !exists {
		return nil
	}
	return &agent
}

func (a *AgentPool) GetManager(name string) sse.Manager {
	a.Lock()
	defer a.Unlock()
	return a.managers[name]
}
