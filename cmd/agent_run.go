package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/services"
	"github.com/mudler/LocalAGI/services/skills"
	"github.com/mudler/xlog"
	"github.com/spf13/cobra"
)

var (
	configFile string
)

var agentRunCmd = &cobra.Command{
	Use:   "run [agent_name]",
	Short: "Run an agent standalone",
	Long: `Run an agent without starting the web server.

Two modes are supported:
  1. Run an agent by name from the registry (pool.json):
       local-agi agent run my-agent

  2. Run an agent from a JSON config file:
       local-agi agent run --config agent.json

The agent runs in the foreground until interrupted (Ctrl+C).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgent,
}

func init() {
	agentRunCmd.Flags().StringVarP(&configFile, "config", "c", "", "path to agent JSON config file")
}

func runAgent(cmd *cobra.Command, args []string) error {
	agentName, agentConfig, err := resolveAgentConfig(args)
	if err != nil {
		return err
	}

	return startStandaloneAgent(agentName, agentConfig)
}

// resolveAgentConfig determines the agent name and config from either
// a registry lookup or a JSON config file.
func resolveAgentConfig(args []string) (string, *state.AgentConfig, error) {
	if configFile != "" && len(args) > 0 {
		return "", nil, fmt.Errorf("cannot specify both --config and agent name; use one or the other")
	}

	if configFile == "" && len(args) == 0 {
		return "", nil, fmt.Errorf("either an agent name or --config <file> is required")
	}

	if configFile != "" {
		return loadConfigFromFile(configFile)
	}

	return loadConfigFromRegistry(args[0])
}

// loadConfigFromFile reads and validates an agent config from a JSON file.
func loadConfigFromFile(path string) (string, *state.AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var config state.AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	if err := validateConfig(&config); err != nil {
		return "", nil, fmt.Errorf("invalid config in %q: %w", path, err)
	}

	name := config.Name
	if name == "" {
		// Derive name from filename
		base := filepath.Base(path)
		name = base[:len(base)-len(filepath.Ext(base))]
		config.Name = name
	}

	return name, &config, nil
}

// loadConfigFromRegistry loads an agent config from the pool.json registry.
func loadConfigFromRegistry(name string) (string, *state.AgentConfig, error) {
	stateDir := os.Getenv("LOCALAGI_STATE_DIR")
	if stateDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		stateDir = filepath.Join(cwd, "pool")
	}

	poolFile := filepath.Join(stateDir, "pool.json")
	data, err := os.ReadFile(poolFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read pool file %q: %w\nEnsure LOCALAGI_STATE_DIR is set or a pool/ directory exists", poolFile, err)
	}

	var pool map[string]state.AgentConfig
	if err := json.Unmarshal(data, &pool); err != nil {
		return "", nil, fmt.Errorf("failed to parse pool file %q: %w", poolFile, err)
	}

	config, exists := pool[name]
	if !exists {
		available := make([]string, 0, len(pool))
		for k := range pool {
			available = append(available, k)
		}
		return "", nil, fmt.Errorf("agent %q not found in registry\nAvailable agents: %v", name, available)
	}

	return name, &config, nil
}

// validateConfig checks that required fields are present in the config.
func validateConfig(config *state.AgentConfig) error {
	// Model and API URL can come from env vars, so they're not strictly required in config.
	// But we validate that the config is at least parseable (already done by JSON unmarshal).
	return nil
}

// startStandaloneAgent creates and runs a single agent without the web server.
func startStandaloneAgent(name string, config *state.AgentConfig) error {
	// Resolve defaults from environment variables
	model := envOrDefault("LOCALAGI_MODEL", config.Model)
	apiURL := envOrDefault("LOCALAGI_LLM_API_URL", config.APIURL)
	apiKey := envOrDefault("LOCALAGI_LLM_API_KEY", config.APIKey)
	multimodalModel := envOrDefault("LOCALAGI_MULTIMODAL_MODEL", config.MultimodalModel)
	transcriptionModel := envOrDefault("LOCALAGI_TRANSCRIPTION_MODEL", config.TranscriptionModel)
	transcriptionLanguage := envOrDefault("LOCALAGI_TRANSCRIPTION_LANGUAGE", config.TranscriptionLanguage)
	ttsModel := envOrDefault("LOCALAGI_TTS_MODEL", config.TTSModel)
	timeout := envOrDefault("LOCALAGI_TIMEOUT", "5m")
	stateDir := envOrDefault("LOCALAGI_STATE_DIR", "")
	localRAG := os.Getenv("LOCALAGI_LOCALRAG_URL")
	customActionsDir := os.Getenv("LOCALAGI_CUSTOM_ACTIONS_DIR")
	sshBoxURL := os.Getenv("LOCALAGI_SSHBOX_URL")

	if model == "" {
		return fmt.Errorf("model not set: provide 'model' in config or set LOCALAGI_MODEL")
	}
	if apiURL == "" {
		return fmt.Errorf("API URL not set: provide 'api_url' in config or set LOCALAGI_LLM_API_URL")
	}

	if stateDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		stateDir = filepath.Join(cwd, "pool")
	}
	os.MkdirAll(stateDir, 0755)

	// Override config with resolved values
	config.Model = model
	config.APIURL = apiURL
	config.APIKey = apiKey
	config.MultimodalModel = multimodalModel
	config.TranscriptionModel = transcriptionModel
	config.TranscriptionLanguage = transcriptionLanguage
	config.TTSModel = ttsModel

	if config.PeriodicRuns == "" {
		config.PeriodicRuns = "10m"
	}
	if config.SchedulerPollInterval == "" {
		config.SchedulerPollInterval = "30s"
	}

	// Initialize skills service
	skillsService, err := skills.NewService(stateDir)
	if err != nil {
		return fmt.Errorf("failed to initialize skills service: %w", err)
	}

	// Build actions, connectors, prompts, filters using the same service factories
	actionsFactory := services.Actions(map[string]string{
		services.ActionConfigSSHBoxURL: sshBoxURL,
		services.ConfigStateDir:        stateDir,
		services.CustomActionsDir:      customActionsDir,
	})
	connectorsFactory := services.Connectors
	dynamicPromptsFactory := services.DynamicPrompts(map[string]string{
		services.ConfigStateDir:   stateDir,
		services.CustomActionsDir: customActionsDir,
	})
	filtersFactory := services.Filters

	// Create a minimal pool to satisfy factory functions that need it
	pool, err := state.NewAgentPool(
		model, multimodalModel, transcriptionModel, transcriptionLanguage, ttsModel,
		apiURL, apiKey, stateDir,
		actionsFactory, connectorsFactory, dynamicPromptsFactory, filtersFactory,
		timeout, false, skillsService,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent pool: %w", err)
	}

	// Set up RAG provider
	if localRAG != "" {
		pool.SetRAGProvider(state.NewHTTPRAGProvider(localRAG, apiKey))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build agent options using pool's factories
	connectors := connectorsFactory(config)
	promptBlocks := dynamicPromptsFactory(config)(ctx, pool)
	if skillsService != nil && config.EnableSkills {
		if prompt, err := skillsService.GetSkillsPrompt(config); err == nil && prompt != nil {
			promptBlocks = append(promptBlocks, prompt)
		}
	}
	actions := actionsFactory(config)(ctx, pool)
	filters := filtersFactory(config)

	stateFile := filepath.Join(stateDir, fmt.Sprintf("%s.state.json", name))
	characterFile := filepath.Join(stateDir, fmt.Sprintf("%s.character.json", name))

	opts := []agent.Option{
		agent.WithSchedulerStorePath(filepath.Join(stateDir, fmt.Sprintf("scheduler-%s.json", name))),
		agent.WithModel(model),
		agent.WithLLMAPIURL(apiURL),
		agent.WithLLMAPIKey(apiKey),
		agent.WithContext(ctx),
		agent.WithMCPServers(config.MCPServers...),
		agent.WithTranscriptionModel(transcriptionModel),
		agent.WithTranscriptionLanguage(transcriptionLanguage),
		agent.WithTTSModel(ttsModel),
		agent.WithPeriodicRuns(config.PeriodicRuns),
		agent.WithSchedulerPollInterval(config.SchedulerPollInterval),
		agent.WithPermanentGoal(config.PermanentGoal),
		agent.WithMCPSTDIOServers(config.MCPSTDIOServers...),
		agent.WithPrompts(promptBlocks...),
		agent.WithJobFilters(filters...),
		agent.WithMCPPrepareScript(config.MCPPrepareScript),
		agent.WithCharacter(agent.Character{Name: name}),
		agent.WithActions(actions...),
		agent.WithStateFile(stateFile),
		agent.WithCharacterFile(characterFile),
		agent.WithTimeout(timeout),
		agent.WithSystemPrompt(config.SystemPrompt),
		agent.WithInnerMonologueTemplate(config.InnerMonologueTemplate),
		agent.WithSchedulerTaskTemplate(config.SchedulerTaskTemplate),
		agent.WithMultimodalModel(multimodalModel),
		agent.WithLastMessageDuration(config.LastMessageDuration),
		agent.WithAgentReasoningCallback(func(s types.ActionCurrentState) bool {
			var actionName types.ActionDefinitionName
			if s.Action != nil {
				actionName = s.Action.Definition().Name
			}
			xlog.Info("Agent reasoning", "agent", name, "reasoning", s.Reasoning, "action", actionName, "params", s.Params)
			for _, c := range connectors {
				if !c.AgentReasoningCallback()(s) {
					return false
				}
			}
			return true
		}),
		agent.WithAgentResultCallback(func(s types.ActionState) {
			var actionName types.ActionDefinitionName
			if s.ActionCurrentState.Action != nil {
				actionName = s.ActionCurrentState.Action.Definition().Name
			}
			xlog.Info("Agent result", "agent", name, "reasoning", s.Reasoning, "action", actionName, "result", s.Result)
			for _, c := range connectors {
				c.AgentResultCallback()(s)
			}
		}),
	}

	// Apply boolean/optional config flags
	if config.HUD {
		opts = append(opts, agent.EnableHUD)
	}
	if config.StandaloneJob {
		opts = append(opts, agent.EnableStandaloneJob)
	}
	if config.LongTermMemory {
		opts = append(opts, agent.EnableLongTermMemory)
	}
	if config.SummaryLongTermMemory {
		opts = append(opts, agent.EnableSummaryMemory)
	}
	if config.ConversationStorageMode != "" {
		opts = append(opts, agent.WithConversationStorageMode(agent.ConversationStorageMode(config.ConversationStorageMode)))
	}
	if config.CanStopItself {
		opts = append(opts, agent.CanStopItself)
	}
	if config.CanPlan {
		opts = append(opts, agent.EnablePlanning)
	}
	if config.PlanReviewerModel != "" {
		opts = append(opts, agent.WithPlanReviewerLLM(config.PlanReviewerModel))
	}
	if config.DisableSinkState {
		opts = append(opts, agent.DisableSinkState)
	}
	if config.InitiateConversations {
		opts = append(opts, agent.EnableInitiateConversations)
	}
	if config.RandomIdentity {
		if config.IdentityGuidance != "" {
			opts = append(opts, agent.WithRandomIdentity(config.IdentityGuidance))
		} else {
			opts = append(opts, agent.WithRandomIdentity())
		}
	}
	if skillsService != nil && config.EnableSkills {
		if session, err := skillsService.GetMCPSession(ctx); err == nil && session != nil {
			opts = append(opts, agent.WithMCPSession(session))
		}
	}
	if config.EnableReasoning {
		opts = append(opts, agent.EnableForceReasoning)
	}
	if config.EnableGuidedTools {
		opts = append(opts, agent.EnableGuidedTools)
	}
	if config.StripThinkingTags {
		opts = append(opts, agent.EnableStripThinkingTags)
	}
	if config.EnableAutoCompaction {
		opts = append(opts, agent.EnableAutoCompaction)
	}
	if config.AutoCompactionThreshold > 0 {
		opts = append(opts, agent.WithAutoCompactionThreshold(config.AutoCompactionThreshold))
	}
	if config.KnowledgeBaseResults > 0 {
		opts = append(opts, agent.EnableKnowledgeBaseWithResults(config.KnowledgeBaseResults))
	}
	if config.ParallelJobs > 0 {
		opts = append(opts, agent.WithParallelJobs(config.ParallelJobs))
	}
	if config.CancelPreviousOnNewMessage != nil {
		opts = append(opts, agent.WithCancelPreviousOnNewMessage(*config.CancelPreviousOnNewMessage))
	} else {
		opts = append(opts, agent.WithCancelPreviousOnNewMessage(true))
	}
	if config.EnableEvaluation {
		opts = append(opts, agent.EnableEvaluation())
	}
	if config.MaxEvaluationLoops > 0 {
		opts = append(opts, agent.WithMaxEvaluationLoops(config.MaxEvaluationLoops))
	}
	if config.MaxAttempts > 0 {
		opts = append(opts, agent.WithMaxAttempts(config.MaxAttempts))
	}
	if config.LoopDetection > 0 {
		opts = append(opts, agent.WithLoopDetection(config.LoopDetection))
	}
	if config.EnableForceReasoningTool {
		opts = append(opts, agent.EnableForceReasoningTool)
	}

	// Handle Knowledge Base
	if config.EnableKnowledgeBase {
		ragProvider := pool.GetRAGProvider()
		if ragProvider != nil {
			effectiveRAGURL := config.LocalRAGURL
			effectiveRAGKey := config.LocalRAGAPIKey
			if db, _, ok := ragProvider(name, effectiveRAGURL, effectiveRAGKey); ok && db != nil {
				opts = append(opts, agent.WithRAGDB(db), agent.EnableKnowledgeBase)
				kbAutoSearch := config.KBAutoSearch
				if !config.KBAutoSearch && !config.KBAsTools {
					kbAutoSearch = true
				}
				opts = append(opts, agent.WithKBAutoSearch(kbAutoSearch))
				if config.KBAsTools {
					kbResults := config.KnowledgeBaseResults
					if kbResults <= 0 {
						kbResults = 5
					}
					searchAction, addAction := agent.NewKBWrapperActions(db, kbResults)
					opts = append(opts, agent.WithActions(searchAction, addAction))
				}
			}
		}
	}

	// Create the agent
	a, err := agent.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Starting agent %q (model: %s, api: %s)\n", name, model, apiURL)
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n")

	// Start agent in background
	agentDone := make(chan error, 1)
	go func() {
		agentDone <- a.Run()
	}()

	// Start connectors
	for _, c := range connectors {
		go c.Start(a)
	}

	// Wait for interrupt or agent completion
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "\nReceived %v, stopping agent...\n", sig)
		a.Stop()
		// Give agent a moment to clean up
		select {
		case <-agentDone:
		case <-time.After(5 * time.Second):
			fmt.Fprintf(os.Stderr, "Agent did not stop within 5s, exiting\n")
		}
	case err := <-agentDone:
		if err != nil {
			return fmt.Errorf("agent stopped with error: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Agent %q completed\n", name)
	}

	return nil
}

// envOrDefault returns the environment variable value if set, otherwise the fallback.
func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
