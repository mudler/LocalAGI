package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/services"
	"github.com/mudler/LocalAgent/webui"
)

var testModel = os.Getenv("LOCALAGENT_MODEL")
var multimodalModel = os.Getenv("LOCALAGENT_MULTIMODAL_MODEL")
var apiURL = os.Getenv("LOCALAGENT_LLM_API_URL")
var apiKey = os.Getenv("LOCALAGENT_API_KEY")
var timeout = os.Getenv("LOCALAGENT_TIMEOUT")
var stateDir = os.Getenv("LOCALAGENT_STATE_DIR")
var localRAG = os.Getenv("LOCALAGENT_LOCALRAG_URL")
var withLogs = os.Getenv("LOCALAGENT_ENABLE_CONVERSATIONS_LOGGING") == "true"
var apiKeysEnv = os.Getenv("LOCALAGENT_API_KEYS")

func init() {
	if testModel == "" {
		testModel = "hermes-2-pro-mistral"
	}
	if apiURL == "" {
		apiURL = "http://192.168.68.113:8080"
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

	// Create the agent pool
	pool, err := state.NewAgentPool(
		testModel,
		multimodalModel,
		apiURL,
		apiKey,
		stateDir,
		localRAG,
		services.Actions,
		services.Connectors,
		services.PromptBlocks,
		timeout,
		withLogs,
	)
	if err != nil {
		panic(err)
	}

	// Create the application
	app := webui.NewApp(webui.WithPool(pool), webui.WithApiKeys(apiKeys...))

	// Start the agents
	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	// Start the web server
	log.Fatal(app.Listen(":3000"))
}
