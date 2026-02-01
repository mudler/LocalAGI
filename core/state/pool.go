package state

import (
	"context"
	"encoding/base64"
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
	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/mudler/LocalAGI/pkg/localrag"
	"github.com/mudler/LocalAGI/pkg/utils"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/mudler/xlog"
)

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
	imageModel, localRAGAPI, localRAGKey, apiKey                  string
	availableActions                                              func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action
	connectors                                                    func(*AgentConfig) []Connector
	dynamicPrompt                                                 func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []DynamicPrompt
	filters                                                       func(*AgentConfig) types.JobFilters
	timeout                                                       string
	conversationLogs                                              string
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
	defaultModel, defaultMultimodalModel, defaultTranscriptionModel, defaultTranscriptionLanguage, defaultTTSModel, imageModel, apiURL, apiKey, directory string,
	LocalRAGAPI string,
	availableActions func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []types.Action,
	connectors func(*AgentConfig) []Connector,
	promptBlocks func(*AgentConfig) func(ctx context.Context, pool *AgentPool) []DynamicPrompt,
	filters func(*AgentConfig) types.JobFilters,
	timeout string,
	withLogs bool,
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
			imageModel:                   imageModel,
			localRAGAPI:                  LocalRAGAPI,
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
		}, nil
	}

	poolData, err := loadPoolFromFile(poolfile)
	if err != nil {
		return nil, err
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
		imageModel:                   imageModel,
		apiKey:                       apiKey,
		agents:                       make(map[string]*Agent),
		managers:                     make(map[string]sse.Manager),
		agentStatus:                  map[string]*Status{},
		pool:                         *poolData,
		connectors:                   connectors,
		localRAGAPI:                  LocalRAGAPI,
		dynamicPrompt:                promptBlocks,
		filters:                      filters,
		availableActions:             availableActions,
		timeout:                      timeout,
		conversationLogs:             conversationPath,
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

	go func(ac AgentConfig) {
		// Create the agent avatar
		if err := createAgentAvatar(a.apiURL, a.apiKey, a.defaultModel, a.imageModel, a.pooldir, ac); err != nil {
			xlog.Error("Failed to create agent avatar", "error", err)
		}
	}(a.pool[name])

	return a.startAgentWithConfig(name, agentConfig, nil)
}

func (a *AgentPool) RecreateAgent(name string, agentConfig *AgentConfig) error {
	a.Lock()
	defer a.Unlock()

	oldAgent := a.agents[name]
	var o *types.Observable
	obs := oldAgent.Observer()
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

	a.pool[name] = *agentConfig
	delete(a.agents, name)

	if err := a.save(); err != nil {
		if obs != nil {
			o.Completion = &types.Completion{Error: err.Error()}
			obs.Update(*o)
		}
		return err
	}

	if err := a.startAgentWithConfig(name, agentConfig, obs); err != nil {
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

func createAgentAvatar(APIURL, APIKey, model, imageModel, avatarDir string, agent AgentConfig) error {
	client := llm.NewClient(APIKey, APIURL+"/v1", "10m")

	if imageModel == "" {
		return fmt.Errorf("image model not set")
	}

	if model == "" {
		return fmt.Errorf("default model not set")
	}

	imagePath := filepath.Join(avatarDir, "avatars", fmt.Sprintf("%s.png", agent.Name))
	if _, err := os.Stat(imagePath); err == nil {
		// Image already exists
		xlog.Debug("Avatar already exists", "path", imagePath)
		return nil
	}

	var results struct {
		ImagePrompt string `json:"image_prompt"`
	}

	err := llm.GenerateTypedJSONWithGuidance(
		context.Background(),
		llm.NewClient(APIKey, APIURL, "10m"),
		"Generate a prompt that I can use to create a random avatar for the bot '"+agent.Name+"', the description of the bot is: "+agent.Description,
		model,
		jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"image_prompt": {
					Type:        jsonschema.String,
					Description: "The prompt to generate the image",
				},
			},
			Required: []string{"image_prompt"},
		}, &results)
	if err != nil {
		return fmt.Errorf("failed to generate image prompt: %w", err)
	}

	if results.ImagePrompt == "" {
		xlog.Error("Failed to generate image prompt")
		return fmt.Errorf("failed to generate image prompt")
	}

	req := openai.ImageRequest{
		Prompt:         results.ImagePrompt,
		Model:          imageModel,
		Size:           openai.CreateImageSize256x256,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := client.CreateImage(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to generate image: %w", err)
	}

	if len(resp.Data) == 0 {
		return fmt.Errorf("failed to generate image")
	}

	imageJson := resp.Data[0].B64JSON

	os.MkdirAll(filepath.Join(avatarDir, "avatars"), 0755)

	// Save the image to the agent directory
	imageData, err := base64.StdEncoding.DecodeString(imageJson)
	if err != nil {
		return err
	}

	return os.WriteFile(imagePath, imageData, 0644)
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

func (a *AgentPool) startAgentWithConfig(name string, config *AgentConfig, obs Observer) error {
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

	// XXX: Why do we update the pool config from an Agent's config?
	if config.APIURL != "" {
		a.apiURL = config.APIURL
	} else {
		config.APIURL = a.apiURL
	}

	if config.APIKey != "" {
		a.apiKey = config.APIKey
	} else {
		config.APIKey = a.apiKey
	}

	if config.LocalRAGURL != "" {
		a.localRAGAPI = config.LocalRAGURL
	}

	if config.LocalRAGAPIKey != "" {
		a.localRAGKey = config.LocalRAGAPIKey
	}

	connectors := a.connectors(config)
	promptBlocks := a.dynamicPrompt(config)(ctx, a)
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
		"api_url", a.apiURL,
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
		WithModel(model),
		WithLLMAPIURL(a.apiURL),
		WithContext(ctx),
		WithMCPServers(config.MCPServers...),
		WithTranscriptionModel(transcriptionModel),
		WithTranscriptionLanguage(transcriptionLanguage),
		WithTTSModel(ttsModel),
		WithPeriodicRuns(config.PeriodicRuns),
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
		WithLLMAPIKey(a.apiKey),
		WithTimeout(a.timeout),
		WithAgentReasoningCallback(func(state types.ActionCurrentState) bool {
			xlog.Info(
				"Agent is thinking",
				"agent", name,
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
		if config.IdentityGuidance != "" {
			opts = append(opts, WithRandomIdentity(config.IdentityGuidance))
		} else {
			opts = append(opts, WithRandomIdentity())
		}
	}

	var ragClient *localrag.WrappedClient
	if config.EnableKnowledgeBase {
		ragClient = localrag.NewWrappedClient(a.localRAGAPI, a.localRAGKey, name)
		opts = append(opts, WithRAGDB(ragClient), EnableKnowledgeBase)
		if config.EnableKBCompaction {
			interval := config.KBCompactionInterval
			if interval == "" {
				interval = "daily"
			}
			summarize := config.KBCompactionSummarize
			opts = append(opts, EnableKBCompaction, WithKBCompactionInterval(interval), WithKBCompactionSummarize(summarize))
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

	if config.KnowledgeBaseResults > 0 {
		opts = append(opts, EnableKnowledgeBaseWithResults(config.KnowledgeBaseResults))
	}

	if config.ParallelJobs > 0 {
		opts = append(opts, WithParallelJobs(config.ParallelJobs))
	}

	if config.EnableEvaluation {
		opts = append(opts, EnableEvaluation())
	}

	if config.MaxEvaluationLoops > 0 {
		opts = append(opts, WithMaxEvaluationLoops(config.MaxEvaluationLoops))
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

	if config.EnableKnowledgeBase && config.EnableKBCompaction && ragClient != nil {
		go runCompactionTicker(ctx, ragClient, config, a.apiURL, a.apiKey, model)
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
		if err := a.startAgentWithConfig(name, &config, nil); err != nil {
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
		return a.startAgentWithConfig(name, &config, nil)
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

	// remove avatar
	os.Remove(filepath.Join(a.pooldir, "avatars", fmt.Sprintf("%s.png", name)))

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
	return os.WriteFile(a.file, data, 0644)
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
