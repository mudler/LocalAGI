package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/donseba/go-htmx"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	. "github.com/mudler/local-agent-framework/agent"
	"github.com/mudler/local-agent-framework/llm"
	"github.com/mudler/local-agent-framework/llm/rag"
)

var testModel = os.Getenv("TEST_MODEL")
var apiURL = os.Getenv("API_URL")
var apiKey = os.Getenv("API_KEY")
var vectorStore = os.Getenv("VECTOR_STORE")
var kbdisableIndexing = os.Getenv("KBDISABLEINDEX")

const defaultChunkSize = 4098

func init() {
	if testModel == "" {
		testModel = "hermes-2-pro-mistral"
	}
	if apiURL == "" {
		apiURL = "http://192.168.68.113:8080"
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

	var dbStore RAGDB
	lai := llm.NewClient(apiKey, apiURL+"/v1")

	switch vectorStore {
	case "localai":
		laiStore := rag.NewStoreClient(apiURL, apiKey)
		dbStore = rag.NewLocalAIRAGDB(laiStore, lai)
	default:
		var err error
		dbStore, err = rag.NewChromemDB("local-agent-framework", stateDir, lai)
		if err != nil {
			panic(err)
		}
	}

	pool, err := NewAgentPool(testModel, apiURL, stateDir, dbStore)
	if err != nil {
		panic(err)
	}

	db, err := NewInMemoryDB(stateDir, dbStore)
	if err != nil {
		panic(err)
	}

	if len(db.Database) > 0 && kbdisableIndexing != "true" {
		fmt.Println("Loading knowledgebase from disk, to skip run with KBDISABLEINDEX=true")
		if err := db.SaveToStore(); err != nil {
			fmt.Println("Error storing in the KB", err)
		}
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

	RegisterRoutes(webapp, pool, db, app)

	log.Fatal(webapp.Listen(":3000"))
}
