package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mudler/LocalAGI/core/agent"
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

// Collection / knowledge base env
var collectionDBPath = os.Getenv("COLLECTION_DB_PATH")
var fileAssets = os.Getenv("FILE_ASSETS")
var vectorEngine = os.Getenv("VECTOR_ENGINE")
var embeddingModel = os.Getenv("EMBEDDING_MODEL")
var maxChunkingSizeEnv = os.Getenv("MAX_CHUNKING_SIZE")
var chunkOverlapEnv = os.Getenv("CHUNK_OVERLAP")
var databaseURL = os.Getenv("DATABASE_URL")

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

	// Collection defaults when env unset
	if collectionDBPath == "" {
		collectionDBPath = filepath.Join(stateDir, "collections")
	}
	if fileAssets == "" {
		fileAssets = filepath.Join(stateDir, "assets")
	}
	if vectorEngine == "" {
		vectorEngine = "chromem"
	}
	if embeddingModel == "" {
		embeddingModel = "granite-embedding-107m-multilingual"
	}
	maxChunkingSize := 400
	if maxChunkingSizeEnv != "" {
		if n, err := strconv.Atoi(maxChunkingSizeEnv); err == nil {
			maxChunkingSize = n
		}
	}
	chunkOverlap := 0
	if chunkOverlapEnv != "" {
		if n, err := strconv.Atoi(chunkOverlapEnv); err == nil {
			chunkOverlap = n
		}
	}

	apiKeys := []string{}
	if apiKeysEnv != "" {
		apiKeys = strings.Split(apiKeysEnv, ",")
	}

	// Skills service (optional: provides skills prompt and MCP when agents have EnableSkills)
	skillsService, err := skills.NewService(stateDir)
	if err != nil {
		panic(err)
	}

	// Create the agent pool (RAG provider set below after app is created)
	pool, err := state.NewAgentPool(
		baseModel,
		multimodalModel,
		transcriptionModel,
		transcriptionLanguage,
		ttsModel,
		apiURL,
		apiKey,
		stateDir,
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

	// Create the application (registers collection routes and sets up in-process RAG state)
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
		webui.WithCollectionDBPath(collectionDBPath),
		webui.WithFileAssets(fileAssets),
		webui.WithVectorEngine(vectorEngine),
		webui.WithEmbeddingModel(embeddingModel),
		webui.WithMaxChunkingSize(maxChunkingSize),
		webui.WithChunkOverlap(chunkOverlap),
		webui.WithDatabaseURL(databaseURL),
		webui.WithLocalRAGURL(localRAG),
	)

	// Single RAG provider: HTTP client when URL set, in-process when not
	if localRAG != "" {
		pool.SetRAGProvider(state.NewHTTPRAGProvider(localRAG, apiKey))
	} else {
		embedded := app.CollectionsRAGProvider()
		pool.SetRAGProvider(func(collectionName, _, _ string) (agent.RAGDB, state.KBCompactionClient, bool) {
			return embedded(collectionName)
		})
	}

	// Start the agents
	if err := pool.StartAll(); err != nil {
		panic(err)
	}

	// Start the web server
	log.Fatal(app.Listen(":3000"))
}
