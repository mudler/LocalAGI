package cmd

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
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the LocalAGI web server",
	Long:  "Start the LocalAGI web server with the agent pool and web UI.",
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	baseModel := os.Getenv("LOCALAGI_MODEL")
	multimodalModel := os.Getenv("LOCALAGI_MULTIMODAL_MODEL")
	transcriptionModel := os.Getenv("LOCALAGI_TRANSCRIPTION_MODEL")
	transcriptionLanguage := os.Getenv("LOCALAGI_TRANSCRIPTION_LANGUAGE")
	ttsModel := os.Getenv("LOCALAGI_TTS_MODEL")
	apiURL := os.Getenv("LOCALAGI_LLM_API_URL")
	apiKey := os.Getenv("LOCALAGI_LLM_API_KEY")
	timeout := os.Getenv("LOCALAGI_TIMEOUT")
	stateDir := os.Getenv("LOCALAGI_STATE_DIR")
	localRAG := os.Getenv("LOCALAGI_LOCALRAG_URL")
	withLogs := os.Getenv("LOCALAGI_ENABLE_CONVERSATIONS_LOGGING") == "true"
	apiKeysEnv := os.Getenv("LOCALAGI_API_KEYS")
	conversationDuration := os.Getenv("LOCALAGI_CONVERSATION_DURATION")
	customActionsDir := os.Getenv("LOCALAGI_CUSTOM_ACTIONS_DIR")
	sshBoxURL := os.Getenv("LOCALAGI_SSHBOX_URL")

	collectionDBPath := os.Getenv("COLLECTION_DB_PATH")
	fileAssets := os.Getenv("FILE_ASSETS")
	vectorEngine := os.Getenv("VECTOR_ENGINE")
	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	maxChunkingSizeEnv := os.Getenv("MAX_CHUNKING_SIZE")
	chunkOverlapEnv := os.Getenv("CHUNK_OVERLAP")
	databaseURL := os.Getenv("DATABASE_URL")

	if baseModel == "" {
		return cmd.Help()
	}
	if apiURL == "" {
		return cmd.Help()
	}
	if timeout == "" {
		timeout = "5m"
	}
	if stateDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		stateDir = filepath.Join(cwd, "pool")
	}

	os.MkdirAll(stateDir, 0755)

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

	skillsService, err := skills.NewService(stateDir)
	if err != nil {
		return err
	}

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
		return err
	}

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

	if localRAG != "" {
		pool.SetRAGProvider(state.NewHTTPRAGProvider(localRAG, apiKey))
	} else {
		embedded := app.CollectionsRAGProvider()
		pool.SetRAGProvider(func(collectionName, _, _ string) (agent.RAGDB, state.KBCompactionClient, bool) {
			return embedded(collectionName)
		})
	}

	if err := pool.StartAll(); err != nil {
		return err
	}

	log.Fatal(app.Listen(":3000"))
	return nil
}
