package webui

import (
	"github.com/mudler/LocalAGI/webui/collections"
)

// NewInProcessCollectionsBackend delegates to the collections sub-package.
func NewInProcessCollectionsBackend(cfg *Config) (CollectionsBackend, *CollectionsState) {
	collCfg := &collections.Config{
		LLMAPIURL:        cfg.LLMAPIURL,
		LLMAPIKey:        cfg.LLMAPIKey,
		LLMModel:         cfg.LLMModel,
		CollectionDBPath: cfg.CollectionDBPath,
		FileAssets:       cfg.FileAssets,
		VectorEngine:     cfg.VectorEngine,
		EmbeddingModel:   cfg.EmbeddingModel,
		MaxChunkingSize:  cfg.MaxChunkingSize,
		ChunkOverlap:     cfg.ChunkOverlap,
		DatabaseURL:      cfg.DatabaseURL,
	}
	return collections.NewInProcessBackend(collCfg)
}
