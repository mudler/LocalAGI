package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/services"
	"github.com/mudler/LocalAGI/services/skills"
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

// startStandaloneAgent creates and runs a single agent using the pool,
// without starting the web server.
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

	// Build service factories
	actionsFactory := services.Actions(map[string]string{
		services.ActionConfigSSHBoxURL: sshBoxURL,
		services.ConfigStateDir:        stateDir,
		services.CustomActionsDir:      customActionsDir,
	})
	dynamicPromptsFactory := services.DynamicPrompts(map[string]string{
		services.ConfigStateDir:   stateDir,
		services.CustomActionsDir: customActionsDir,
	})

	// Create the pool and use it to start the agent
	pool, err := state.NewAgentPool(
		model, multimodalModel, transcriptionModel, transcriptionLanguage, ttsModel,
		apiURL, apiKey, stateDir,
		actionsFactory, services.Connectors, dynamicPromptsFactory, services.Filters,
		timeout, false, skillsService,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent pool: %w", err)
	}

	if localRAG != "" {
		pool.SetRAGProvider(state.NewHTTPRAGProvider(localRAG, apiKey))
	}

	// Start the agent via the pool (handles all option building, connectors, etc.)
	if err := pool.StartAgentStandalone(name, config); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	a := pool.GetAgent(name)
	if a == nil {
		return fmt.Errorf("agent %q was not found after starting", name)
	}

	fmt.Fprintf(os.Stderr, "Starting agent %q (model: %s, api: %s)\n", name, model, apiURL)
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n")

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	fmt.Fprintf(os.Stderr, "\nReceived %v, stopping agent...\n", sig)
	pool.Stop(name)

	// Give agent a moment to clean up
	time.Sleep(2 * time.Second)

	return nil
}

// envOrDefault returns the environment variable value if set, otherwise the fallback.
func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
