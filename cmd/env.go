package cmd

import (
	"os"
	"strconv"
	"strings"
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
	LocalRAGURL             string
	CustomActionsDir        string
	SSHBoxURL               string
	CollectionDBPath        string
	FileAssets              string
	
	// Conversation settings
	EnableConversationsLogging bool
	APIKeys                   []string
	ConversationDuration      string
	
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

// envOrDefault returns the environment variable value if set, otherwise the fallback.
func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
