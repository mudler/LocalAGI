package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/services"
	"github.com/mudler/LocalAgent/webui"
)

var testModel = os.Getenv("TEST_MODEL")
var apiURL = os.Getenv("API_URL")
var apiKey = os.Getenv("API_KEY")
var timeout = os.Getenv("TIMEOUT")
var stateDir = os.Getenv("STATE_DIR")
var localRAG = os.Getenv("LOCAL_RAG")

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

	// Create the agent pool
	pool, err := state.NewAgentPool(
		testModel,
		apiURL,
		apiKey,
		stateDir,
		localRAG,
		services.Actions,
		services.Connectors,
		services.PromptBlocks,
		timeout,
	)
	if err != nil {
		panic(err)
	}

	// Create the application
	app := webui.NewApp(webui.WithPool(pool))

	// Start the agents
	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	// Start the web server
	log.Fatal(app.Listen(":3000"))
}
