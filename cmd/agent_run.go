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
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/services"
	"github.com/mudler/LocalAGI/services/skills"
	"github.com/spf13/cobra"
)

var (
	configFile string
	prompt     string
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
  3. Run an agent in foreground mode with a prompt:
       local-agi agent run my-agent --prompt "Your question here"

The agent runs in the foreground until interrupted (Ctrl+C).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgent,
}

func init() {
	agentRunCmd.Flags().StringVarP(&configFile, "config", "c", "", "path to agent JSON config file")
	agentRunCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "run in foreground mode with the given prompt and exit after response")
}

func runAgent(cmd *cobra.Command, args []string) error {
	agentName, agentConfig, err := resolveAgentConfig(args)
	if err != nil {
		return err
	}

	// If --prompt is provided, run in foreground mode
	if prompt != "" {
		return runAgentForeground(agentName, agentConfig, prompt)
	}

	return startStandaloneAgent(agentName, agentConfig)
}

// runAgentForeground runs an agent in foreground mode with a single prompt,
// prints the response, and exits.
func runAgentForeground(agentName string, agentConfig *state.AgentConfig, promptText string) error {
	// Load all environment variables
	env := LoadEnv()

	if env.Model == "" {
		env.Model = agentConfig.Model
	}
	if env.LLMAPIURL == "" {
		env.LLMAPIURL = agentConfig.APIURL
	}
	if env.LLMAPIKey == "" {
		env.LLMAPIKey = agentConfig.APIKey
	}

	if env.Model == "" {
		return fmt.Errorf("model not set: provide 'model' in config or set LOCALAGI_MODEL")
	}
	if env.LLMAPIURL == "" {
		return fmt.Errorf("API URL not set: provide 'api_url' in config or set LOCALAGI_LLM_API_URL")
	}

	if env.StateDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		env.StateDir = filepath.Join(cwd, "pool")
	}
	os.MkdirAll(env.StateDir, 0755)

	// Override config with resolved values
	agentConfig.Model = env.Model
	agentConfig.APIURL = env.LLMAPIURL
	agentConfig.APIKey = env.LLMAPIKey
	agentConfig.MultimodalModel = env.MultimodalModel
	agentConfig.TranscriptionModel = env.TranscriptionModel
	agentConfig.TranscriptionLanguage = env.TranscriptionLanguage
	agentConfig.TTSModel = env.TTSModel

	// Initialize skills service
	skillsService, err := skills.NewService(env.StateDir)
	if err != nil {
		return fmt.Errorf("failed to initialize skills service: %w", err)
	}

	// Build service factories
	actionsFactory := services.Actions(map[string]string{
		services.ActionConfigSSHBoxURL: env.SSHBoxURL,
		services.ConfigStateDir:        env.StateDir,
		services.CustomActionsDir:      env.CustomActionsDir,
	})
	dynamicPromptsFactory := services.DynamicPrompts(map[string]string{
		services.ConfigStateDir:   env.StateDir,
		services.CustomActionsDir: env.CustomActionsDir,
	})

	// Create the pool
	pool, err := state.NewAgentPool(
		env.Model, env.MultimodalModel, env.TranscriptionModel, env.TranscriptionLanguage, env.TTSModel,
		env.LLMAPIURL, env.LLMAPIKey, env.StateDir,
		actionsFactory, services.Connectors, dynamicPromptsFactory, services.Filters,
		env.Timeout, false, skillsService,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent pool: %w", err)
	}

	if env.LocalRAGURL != "" {
		pool.SetRAGProvider(state.NewHTTPRAGProvider(env.LocalRAGURL, env.LLMAPIKey))
	}

	// Start the agent
	if err := pool.StartAgentStandalone(agentName, agentConfig); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	a := pool.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q was not found after starting", agentName)
	}

	fmt.Fprintf(os.Stderr, "Running agent %q in foreground mode with prompt...\n", agentName)

	// Execute Ask with the prompt using WithText option
	result := a.Ask(types.WithText(promptText))

	// Print the result
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
		pool.Stop(agentName)
		return fmt.Errorf("agent error: %s", result.Error)
	}

	// Print the response
	fmt.Println(result.Text)

	// Clean up
	pool.Stop(agentName)
	return nil
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
	// Load all environment variables
	env := LoadEnv()

	if env.Model == "" {
		env.Model = config.Model
	}
	if env.LLMAPIURL == "" {
		env.LLMAPIURL = config.APIURL
	}
	if env.LLMAPIKey == "" {
		env.LLMAPIKey = config.APIKey
	}

	if env.Model == "" {
		return fmt.Errorf("model not set: provide 'model' in config or set LOCALAGI_MODEL")
	}
	if env.LLMAPIURL == "" {
		return fmt.Errorf("API URL not set: provide 'api_url' in config or set LOCALAGI_LLM_API_URL")
	}

	if env.StateDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		env.StateDir = filepath.Join(cwd, "pool")
	}
	os.MkdirAll(env.StateDir, 0755)

	// Override config with resolved values
	config.Model = env.Model
	config.APIURL = env.LLMAPIURL
	config.APIKey = env.LLMAPIKey
	config.MultimodalModel = env.MultimodalModel
	config.TranscriptionModel = env.TranscriptionModel
	config.TranscriptionLanguage = env.TranscriptionLanguage
	config.TTSModel = env.TTSModel

	if config.PeriodicRuns == "" {
		config.PeriodicRuns = "10m"
	}
	if config.SchedulerPollInterval == "" {
		config.SchedulerPollInterval = "30s"
	}

	// Initialize skills service
	skillsService, err := skills.NewService(env.StateDir)
	if err != nil {
		return fmt.Errorf("failed to initialize skills service: %w", err)
	}

	// Build service factories
	actionsFactory := services.Actions(map[string]string{
		services.ActionConfigSSHBoxURL: env.SSHBoxURL,
		services.ConfigStateDir:        env.StateDir,
		services.CustomActionsDir:      env.CustomActionsDir,
	})
	dynamicPromptsFactory := services.DynamicPrompts(map[string]string{
		services.ConfigStateDir:   env.StateDir,
		services.CustomActionsDir: env.CustomActionsDir,
	})

	// Create the pool and use it to start the agent
	pool, err := state.NewAgentPool(
		env.Model, env.MultimodalModel, env.TranscriptionModel, env.TranscriptionLanguage, env.TTSModel,
		env.LLMAPIURL, env.LLMAPIKey, env.StateDir,
		actionsFactory, services.Connectors, dynamicPromptsFactory, services.Filters,
		env.Timeout, false, skillsService,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent pool: %w", err)
	}

	if env.LocalRAGURL != "" {
		pool.SetRAGProvider(state.NewHTTPRAGProvider(env.LocalRAGURL, env.LLMAPIKey))
	}

	// Start the agent via the pool (handles all option building, connectors, etc.)
	if err := pool.StartAgentStandalone(name, config); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	a := pool.GetAgent(name)
	if a == nil {
		return fmt.Errorf("agent %q was not found after starting", name)
	}

	fmt.Fprintf(os.Stderr, "Starting agent %q (model: %s, api: %s)\n", name, env.Model, env.LLMAPIURL)
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
