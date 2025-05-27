package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/db"
	"github.com/mudler/LocalAGI/services"
	"github.com/mudler/LocalAGI/webui"
)

var (
	baseModel            string
	multimodalModel      string
	apiURL               string
	apiKey               string
	timeout              string
	stateDir             string
	localRAG             string
	withLogs             bool
	apiKeysEnv           string
	imageModel           string
	conversationDuration string
)

func init() {
	_ = godotenv.Load()

	baseModel = os.Getenv("LOCALAGI_MODEL")
	multimodalModel = os.Getenv("LOCALAGI_MULTIMODAL_MODEL")
	apiURL = os.Getenv("LOCALAGI_LLM_API_URL")
	apiKey = os.Getenv("LOCALAGI_LLM_API_KEY")
	timeout = os.Getenv("LOCALAGI_TIMEOUT")
	stateDir = os.Getenv("LOCALAGI_STATE_DIR")
	localRAG = os.Getenv("LOCALAGI_LOCALRAG_URL")
	withLogs = os.Getenv("LOCALAGI_ENABLE_CONVERSATIONS_LOGGING") == "true"
	apiKeysEnv = os.Getenv("LOCALAGI_API_KEYS")
	imageModel = os.Getenv("LOCALAGI_IMAGE_MODEL")
	conversationDuration = os.Getenv("LOCALAGI_CONVERSATION_DURATION")

	if baseModel == "" {
		panic("LOCALAGI_MODEL not set")
	}
	if apiURL == "" {
		panic("LOCALAGI_API_URL not set")
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

	db.ConnectDB()

	apiKeys := []string{}
	if apiKeysEnv != "" {
		apiKeys = strings.Split(apiKeysEnv, ",")
	}

	// Create the agent pool
	pool, err := state.NewAgentPool(
		"1",
		baseModel,
		multimodalModel,
		imageModel,
		apiURL,
		apiKey,
		stateDir,
		localRAG,
		services.Actions,
		services.Connectors,
		services.DynamicPrompts,
		timeout,
		withLogs,
	)
	if err != nil {
		panic(err)
	}

	// Create the application
	app := webui.NewApp(
		webui.WithPool(pool),
		webui.WithConversationStoreduration(conversationDuration),
		webui.WithApiKeys(apiKeys...),
		webui.WithLLMAPIUrl(apiURL),
		webui.WithLLMAPIKey(apiKey),
		webui.WithLLMModel(baseModel),
		webui.WithStateDir(stateDir),
	)

	// Start the agents
	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	// Start the web server
	log.Fatal(app.Listen(":3000"))
}
