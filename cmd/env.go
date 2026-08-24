package cmd

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mudler/LocalAGI/core/conversations"
	"github.com/mudler/LocalAGI/core/scheduler"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/xlog"
)

// Env contains all environment variables used by LocalAGI
type Env struct {
	// Model and API configuration
	Model                   string
	LLMAPIURL               string
	LLMAPIKey               string
	MultimodalModel         string
	TranscriptionModel      string
	TranscriptionLanguage   string
	TTSModel                string
	Timeout                 string
	
	// Directories and paths
	StateDir                string
	LocalAGIURL             string
	LocalRAGURL             string
	CustomActionsDir        string
	SSHBoxURL               string
	CollectionDBPath        string
	FileAssets              string
	
	// Conversation settings
	EnableConversationsLogging bool
	APIKeys                   []string
	ConversationDuration      string
	
	// Retention settings. A zero value disables the individual limit.
	ConversationsMaxAge        time.Duration
	ConversationsMaxPerAgent   int
	ConversationsPruneInterval time.Duration
	SchedulerMaxRunsPerTask    int
	SchedulerMaxRunAge         time.Duration
	SchedulerDedupeTasks       bool
	SchedulerMaxTasksPerAgent  int

	// RAG/Vector settings
	VectorEngine              string
	EmbeddingModel            string
	MaxChunkingSize           int
	ChunkOverlap              int
	DatabaseURL               string
}

// LoadEnv reads all environment variables and returns an Env struct
func LoadEnv() Env {
	env := Env{
		Model:                    envOrDefault("LOCALAGI_MODEL", ""),
		LLMAPIURL:                envOrDefault("LOCALAGI_LLM_API_URL", ""),
		LLMAPIKey:                envOrDefault("LOCALAGI_LLM_API_KEY", ""),
		MultimodalModel:          envOrDefault("LOCALAGI_MULTIMODAL_MODEL", ""),
		TranscriptionModel:       envOrDefault("LOCALAGI_TRANSCRIPTION_MODEL", ""),
		TranscriptionLanguage:    envOrDefault("LOCALAGI_TRANSCRIPTION_LANGUAGE", ""),
		TTSModel:                 envOrDefault("LOCALAGI_TTS_MODEL", ""),
		Timeout:                  envOrDefault("LOCALAGI_TIMEOUT", "5m"),
		StateDir:                 envOrDefault("LOCALAGI_STATE_DIR", ""),
		LocalAGIURL:              envOrDefault("LOCALAGI_BASE_URL", ":3000"),
		LocalRAGURL:              os.Getenv("LOCALAGI_LOCALRAG_URL"),
		CustomActionsDir:         os.Getenv("LOCALAGI_CUSTOM_ACTIONS_DIR"),
		SSHBoxURL:                os.Getenv("LOCALAGI_SSHBOX_URL"),
		EnableConversationsLogging: os.Getenv("LOCALAGI_ENABLE_CONVERSATIONS_LOGGING") == "true",
		ConversationDuration:     os.Getenv("LOCALAGI_CONVERSATION_DURATION"),
		CollectionDBPath:         os.Getenv("COLLECTION_DB_PATH"),
		FileAssets:               os.Getenv("FILE_ASSETS"),
		VectorEngine:             os.Getenv("VECTOR_ENGINE"),
		EmbeddingModel:           os.Getenv("EMBEDDING_MODEL"),
		DatabaseURL:              os.Getenv("DATABASE_URL"),

		ConversationsMaxAge:        envDuration("LOCALAGI_CONVERSATIONS_MAX_AGE", 720*time.Hour),
		ConversationsMaxPerAgent:   envInt("LOCALAGI_CONVERSATIONS_MAX_PER_AGENT", 200),
		ConversationsPruneInterval: envDuration("LOCALAGI_CONVERSATIONS_PRUNE_INTERVAL", time.Hour),
		SchedulerMaxRunsPerTask:    envInt("LOCALAGI_SCHEDULER_MAX_RUNS_PER_TASK", 20),
		SchedulerMaxRunAge:         envDuration("LOCALAGI_SCHEDULER_MAX_RUN_AGE", 720*time.Hour),
		SchedulerDedupeTasks:       envBool("LOCALAGI_SCHEDULER_DEDUPE_TASKS", true),
		SchedulerMaxTasksPerAgent:  envInt("LOCALAGI_SCHEDULER_MAX_TASKS_PER_AGENT", 100),
	}
	
	// Parse APIKeys from comma-separated string
	if apiKeysEnv := os.Getenv("LOCALAGI_API_KEYS"); apiKeysEnv != "" {
		env.APIKeys = strings.Split(apiKeysEnv, ",")
	}
	
	// Parse numeric values
	if maxChunkingSizeEnv := os.Getenv("MAX_CHUNKING_SIZE"); maxChunkingSizeEnv != "" {
		if n, err := strconv.Atoi(maxChunkingSizeEnv); err == nil {
			env.MaxChunkingSize = n
		}
	}
	
	if chunkOverlapEnv := os.Getenv("CHUNK_OVERLAP"); chunkOverlapEnv != "" {
		if n, err := strconv.Atoi(chunkOverlapEnv); err == nil {
			env.ChunkOverlap = n
		}
	}
	
	// Set defaults for empty values
	if env.VectorEngine == "" {
		env.VectorEngine = "chromem"
	}
	if env.EmbeddingModel == "" {
		env.EmbeddingModel = "granite-embedding-107m-multilingual"
	}
	if env.MaxChunkingSize == 0 {
		env.MaxChunkingSize = 400
	}
	
	return env
}

// Limits builds the pool limits from the parsed environment.
func (e Env) Limits() state.PoolLimits {
	return state.PoolLimits{
		Conversations: conversations.RetentionPolicy{
			MaxAge:      e.ConversationsMaxAge,
			MaxPerAgent: e.ConversationsMaxPerAgent,
		},
		ConversationSweep: e.ConversationsPruneInterval,
		SchedulerRuns: scheduler.RetentionPolicy{
			MaxRunsPerTask: e.SchedulerMaxRunsPerTask,
			MaxRunAge:      e.SchedulerMaxRunAge,
		},
		SchedulerCreation: scheduler.CreationPolicy{
			Dedupe:           e.SchedulerDedupeTasks,
			MaxTasksPerAgent: e.SchedulerMaxTasksPerAgent,
		},
	}
}

// envDuration reads a duration, accepting a day suffix ("30d") that
// time.ParseDuration does not. An unset value takes the fallback; an
// unparseable one keeps the fallback rather than silently disabling a limit.
func envDuration(envKey string, fallback time.Duration) time.Duration {
	v := os.Getenv(envKey)
	if v == "" {
		return fallback
	}

	d, err := scheduler.ParseDuration(v)
	if err != nil {
		xlog.Warn("Ignoring unparseable duration, using default", "env", envKey, "value", v, "default", fallback)
		return fallback
	}

	return d
}

// envInt reads an integer, keeping the fallback when the value is unset or
// unparseable.
func envInt(envKey string, fallback int) int {
	v := os.Getenv(envKey)
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		xlog.Warn("Ignoring unparseable integer, using default", "env", envKey, "value", v, "default", fallback)
		return fallback
	}

	return n
}

// envBool reads a boolean, keeping the fallback when the value is unset or
// unparseable.
func envBool(envKey string, fallback bool) bool {
	v := os.Getenv(envKey)
	if v == "" {
		return fallback
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		xlog.Warn("Ignoring unparseable boolean, using default", "env", envKey, "value", v, "default", fallback)
		return fallback
	}

	return b
}

// envOrDefault returns the environment variable value if set, otherwise the fallback.
func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
