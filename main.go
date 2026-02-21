package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/services"
	"github.com/mudler/LocalAGI/services/skills"
	"github.com/mudler/LocalAGI/webui"
)

var baseModel = os.Getenv("LOCALAGI_MODEL")
var multimodalModel = os.Getenv("LOCALAGI_MULTIMODAL_MODEL")
var transcriptionModel = os.Getenv("LOCALAGI_TRANSCRIPTION_MODEL")
var transcriptionLanguage = os.Getenv("LOCALAGI_TRANSCRIPTION_LANGUAGE")
var ttsModel = os.Getenv("LOCALAGI_TTS_MODEL")
var apiURL = os.Getenv("LOCALAGI_LLM_API_URL")
var apiKey = os.Getenv("LOCALAGI_LLM_API_KEY")
var timeout = os.Getenv("LOCALAGI_TIMEOUT")
var stateDir = os.Getenv("LOCALAGI_STATE_DIR")
var localRAG = os.Getenv("LOCALAGI_LOCALRAG_URL")
var withLogs = os.Getenv("LOCALAGI_ENABLE_CONVERSATIONS_LOGGING") == "true"
var apiKeysEnv = os.Getenv("LOCALAGI_API_KEYS")
var conversationDuration = os.Getenv("LOCALAGI_CONVERSATION_DURATION")
var customActionsDir = os.Getenv("LOCALAGI_CUSTOM_ACTIONS_DIR")
var sshBoxURL = os.Getenv("LOCALAGI_SSHBOX_URL")

func init() {
	if baseModel == "" {
		panic("LOCALAGI_MODEL not set")
	}
	if apiURL == "" {
		panic("LOCALAGI_LLM_API_URL not set")
	}
	if timeout == "" {
		timeout = "5m"
	}
	if stateDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		stateDir = filepath.Join(cwd, "pool")
	}
}

func main() {
	// make sure state dir exists
	os.MkdirAll(stateDir, 0755)

	apiKeys := []string{}
	if apiKeysEnv != "" {
		apiKeys = strings.Split(apiKeysEnv, ",")
	}

	// Skills service (optional: provides skills prompt and MCP when agents have EnableSkills)
	skillsService, err := skills.NewService(stateDir)
	if err != nil {
		panic(err)
	}

	// Create the agent pool
	pool, err := state.NewAgentPool(
		baseModel,
		multimodalModel,
		transcriptionModel,
		transcriptionLanguage,
		ttsModel,
		apiURL,
		apiKey,
		stateDir,
		localRAG,
		services.Actions(map[string]string{
			services.ActionConfigSSHBoxURL: sshBoxURL,
			services.ConfigStateDir:        stateDir,
			services.CustomActionsDir:      customActionsDir,
		}),
		services.Connectors,
		services.DynamicPrompts(map[string]string{
			services.ConfigStateDir:   stateDir,
			services.CustomActionsDir: customActionsDir,
		}),
		services.Filters,
		timeout,
		withLogs,
		skillsService,
	)
	if err != nil {
		panic(err)
	}

	// Create the application (this registers collection routes and sets up in-process RAG state)
	app := webui.NewApp(
		webui.WithPool(pool),
		webui.WithSkillsService(skillsService),
		webui.WithConversationStoreduration(conversationDuration),
		webui.WithApiKeys(apiKeys...),
		webui.WithLLMAPIUrl(apiURL),
		webui.WithLLMAPIKey(apiKey),
		webui.WithLLMModel(baseModel),
		webui.WithCustomActionsDir(customActionsDir),
		webui.WithStateDir(stateDir),
	)

	// When no LocalRAG URL is set, agents with knowledge base use in-process collections (no HTTP client).
	if localRAG == "" {
		pool.SetInternalRAGProvider(app.CollectionsRAGProvider())
	}

	// Start the agents
	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	// Start the web server
	log.Fatal(app.Listen(":3000"))
}
