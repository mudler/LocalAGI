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
		ModelSettings: func(name string) collections.CollectionModelSettings {
			if cfg.Pool != nil {
				if agent := cfg.Pool.GetConfig(name); agent != nil {
					return collections.CollectionModelSettings{EmbeddingModel: agent.EmbeddingModel, RerankerModel: agent.RerankerModel}
				}
			}
			return collections.CollectionModelSettings{}
		},
	}
	return collections.NewInProcessBackend(collCfg)
}
