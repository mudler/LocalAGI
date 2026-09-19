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
		ModelSettings: func(name string) (collections.CollectionModelSettings, error) {
			if cfg.Pool != nil {
				agent, err := cfg.Pool.GetCollectionConfig(name)
				if err != nil {
					return collections.CollectionModelSettings{}, err
				}
				if agent != nil {
					return collections.CollectionModelSettings{EmbeddingModel: agent.EmbeddingModel, RerankerModel: agent.RerankerModel}, nil
				}
			}
			return collections.CollectionModelSettings{}, nil
		},
	}
	return collections.NewInProcessBackend(collCfg)
}
