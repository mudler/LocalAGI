package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/state"
	"github.com/mudler/LocalAgent/pkg/llm"
	rag "github.com/mudler/LocalAgent/pkg/vectorstore"
	"github.com/mudler/LocalAgent/webui"
)

var testModel = os.Getenv("TEST_MODEL")
var apiURL = os.Getenv("API_URL")
var apiKey = os.Getenv("API_KEY")
var vectorStore = os.Getenv("VECTOR_STORE")
var timeout = os.Getenv("TIMEOUT")
var embeddingModel = os.Getenv("EMBEDDING_MODEL")
var stateDir = os.Getenv("STATE_DIR")

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

func ragDB() (ragDB agent.RAGDB) {
	lai := llm.NewClient(apiKey, apiURL+"/v1", timeout)

	switch vectorStore {
	case "localai":
		laiStore := rag.NewStoreClient(apiURL, apiKey)
		ragDB = rag.NewLocalAIRAGDB(laiStore, lai)
	default:
		var err error
		ragDB, err = rag.NewChromemDB("local-agent-framework", stateDir, lai, embeddingModel)
		if err != nil {
			panic(err)
		}
	}

	return
}

func main() {
	// make sure state dir exists
	os.MkdirAll(stateDir, 0755)

	// Initialize rag DB connection
	ragDB := ragDB()

	// Create the agent pool
	pool, err := state.NewAgentPool(testModel, apiURL, stateDir, ragDB, webui.Actions, webui.Connectors, timeout)
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
