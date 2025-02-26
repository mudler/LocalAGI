package main

import (
	"log"
	"os"

	"github.com/mudler/local-agent-framework/core/agent"
	"github.com/mudler/local-agent-framework/core/state"
	"github.com/mudler/local-agent-framework/pkg/llm"
	rag "github.com/mudler/local-agent-framework/pkg/vectorstore"
	"github.com/mudler/local-agent-framework/webui"
)

var testModel = os.Getenv("TEST_MODEL")
var apiURL = os.Getenv("API_URL")
var apiKey = os.Getenv("API_KEY")
var vectorStore = os.Getenv("VECTOR_STORE")
var timeout = os.Getenv("TIMEOUT")
var embeddingModel = os.Getenv("EMBEDDING_MODEL")

const defaultChunkSize = 4098

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
}

func main() {
	// current dir
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	stateDir := cwd + "/pool"
	os.MkdirAll(stateDir, 0755)

	var ragDB agent.RAGDB
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

	pool, err := state.NewAgentPool(testModel, apiURL, stateDir, ragDB, webui.Actions, webui.Connectors, timeout)
	if err != nil {
		panic(err)
	}

	app := webui.NewApp(webui.WithPool(pool))

	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	log.Fatal(app.Listen(":3000"))
}
