package main

import (
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/donseba/go-htmx"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"github.com/mudler/local-agent-framework/core/agent"
	"github.com/mudler/local-agent-framework/core/state"
	"github.com/mudler/local-agent-framework/pkg/llm"
	rag "github.com/mudler/local-agent-framework/pkg/vectorstore"
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

//go:embed views/*
var viewsfs embed.FS

//go:embed public/*
var embeddedFiles embed.FS

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

	pool, err := state.NewAgentPool(testModel, apiURL, stateDir, ragDB, Actions, Connectors, timeout)
	if err != nil {
		panic(err)
	}

	app := &App{
		htmx: htmx.New(),
		pool: pool,
	}

	if err := pool.StartAll(); err != nil {
		panic(err)
	}
	engine := html.NewFileSystem(http.FS(viewsfs), ".html")
	// Initialize a new Fiber app
	// Pass the engine to the Views
	webapp := fiber.New(fiber.Config{
		Views: engine,
	})

	RegisterRoutes(webapp, pool, app)

	log.Fatal(webapp.Listen(":3000"))
}
