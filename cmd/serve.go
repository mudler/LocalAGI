package cmd

import (
	"log"
	"os"
	"path/filepath"

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
	// Load all environment variables
	env := LoadEnv()

	if env.Model == "" {
		return cmd.Help()
	}
	if env.LLMAPIURL == "" {
		return cmd.Help()
	}

	if env.StateDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		env.StateDir = filepath.Join(cwd, "pool")
	}

	os.MkdirAll(env.StateDir, 0755)

	if env.CollectionDBPath == "" {
		env.CollectionDBPath = filepath.Join(env.StateDir, "collections")
	}
	if env.FileAssets == "" {
		env.FileAssets = filepath.Join(env.StateDir, "assets")
	}

	apiKeys := env.APIKeys
	if len(apiKeys) == 0 {
		apiKeys = []string{}
	}

	skillsService, err := skills.NewService(env.StateDir)
	if err != nil {
		return err
	}

	pool, err := state.NewAgentPool(
		env.Model,
		env.MultimodalModel,
		env.TranscriptionModel,
		env.TranscriptionLanguage,
		env.TTSModel,
		env.LLMAPIURL,
		env.LLMAPIKey,
		env.StateDir,
		services.Actions(map[string]string{
			services.ActionConfigSSHBoxURL: env.SSHBoxURL,
			services.ConfigStateDir:        env.StateDir,
			services.CustomActionsDir:      env.CustomActionsDir,
		}),
		services.Connectors,
		services.DynamicPrompts(map[string]string{
			services.ConfigStateDir:   env.StateDir,
			services.CustomActionsDir: env.CustomActionsDir,
		}),
		services.Filters,
		env.Timeout,
		env.EnableConversationsLogging,
		skillsService,
	)
	if err != nil {
		return err
	}

	app := webui.NewApp(
		webui.WithPool(pool),
		webui.WithSkillsService(skillsService),
		webui.WithConversationStoreduration(env.ConversationDuration),
		webui.WithApiKeys(apiKeys...),
		webui.WithLLMAPIUrl(env.LLMAPIURL),
		webui.WithLLMAPIKey(env.LLMAPIKey),
		webui.WithLLMModel(env.Model),
		webui.WithCustomActionsDir(env.CustomActionsDir),
		webui.WithStateDir(env.StateDir),
		webui.WithCollectionDBPath(env.CollectionDBPath),
		webui.WithFileAssets(env.FileAssets),
		webui.WithVectorEngine(env.VectorEngine),
		webui.WithEmbeddingModel(env.EmbeddingModel),
		webui.WithMaxChunkingSize(env.MaxChunkingSize),
		webui.WithChunkOverlap(env.ChunkOverlap),
		webui.WithDatabaseURL(env.DatabaseURL),
		webui.WithLocalRAGURL(env.LocalRAGURL),
	)

	if env.LocalRAGURL != "" {
		pool.SetRAGProvider(state.NewHTTPRAGProvider(env.LocalRAGURL, env.LLMAPIKey))
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
