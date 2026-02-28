package webui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mudler/localrecall/rag"
	"github.com/mudler/localrecall/rag/sources"
	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai"
)

func newVectorEngine(
	vectorEngineType string,
	llmClient *openai.Client,
	apiURL, apiKey, collectionName, dbPath, fileAssets, embeddingModel, databaseURL string,
	maxChunkSize, chunkOverlap int,
) *rag.PersistentKB {
	switch vectorEngineType {
	case "chromem":
		xlog.Info("Chromem collection", "collectionName", collectionName, "dbPath", dbPath)
		return rag.NewPersistentChromeCollection(llmClient, collectionName, dbPath, fileAssets, embeddingModel, maxChunkSize, chunkOverlap)
	case "localai":
		xlog.Info("LocalAI collection", "collectionName", collectionName, "apiURL", apiURL)
		return rag.NewPersistentLocalAICollection(llmClient, apiURL, apiKey, collectionName, dbPath, fileAssets, embeddingModel, maxChunkSize, chunkOverlap)
	case "postgres":
		if databaseURL == "" {
			xlog.Error("DATABASE_URL is required for PostgreSQL engine")
			return nil
		}
		xlog.Info("PostgreSQL collection", "collectionName", collectionName, "databaseURL", databaseURL)
		return rag.NewPersistentPostgresCollection(llmClient, collectionName, dbPath, fileAssets, embeddingModel, maxChunkSize, chunkOverlap, databaseURL)
	default:
		xlog.Error("Unknown vector engine", "engine", vectorEngineType)
		return nil
	}
}

// collectionsBackendInProcess implements CollectionsBackend using in-process state.
type collectionsBackendInProcess struct {
	state        *collectionsState
	cfg          *Config
	openAIClient *openai.Client
}

var _ CollectionsBackend = (*collectionsBackendInProcess)(nil)

func (b *collectionsBackendInProcess) ListCollections() ([]string, error) {
	return rag.ListAllCollections(b.cfg.CollectionDBPath), nil
}

func (b *collectionsBackendInProcess) CreateCollection(name string) error {
	collection := newVectorEngine(b.cfg.VectorEngine, b.openAIClient, b.cfg.LLMAPIURL, b.cfg.LLMAPIKey, name, b.cfg.CollectionDBPath, b.cfg.FileAssets, b.cfg.EmbeddingModel, b.cfg.DatabaseURL, b.cfg.MaxChunkingSize, b.cfg.ChunkOverlap)
	if collection == nil {
		return fmt.Errorf("unsupported or misconfigured vector engine")
	}
	b.state.mu.Lock()
	b.state.collections[name] = collection
	b.state.sourceManager.RegisterCollection(name, collection)
	b.state.mu.Unlock()
	return nil
}

func (b *collectionsBackendInProcess) Upload(collection, filename string, fileBody io.Reader) error {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return fmt.Errorf("collection not found: %s", collection)
	}
	filePath := filepath.Join(b.cfg.FileAssets, filename)
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, fileBody); err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	return kb.Store(filePath, map[string]string{"created_at": now})
}

func (b *collectionsBackendInProcess) ListEntries(collection string) ([]string, error) {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	return kb.ListDocuments(), nil
}

func (b *collectionsBackendInProcess) GetEntryContent(collection, entry string) (string, int, error) {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return "", 0, fmt.Errorf("collection not found: %s", collection)
	}
	return kb.GetEntryFileContent(entry)
}

func (b *collectionsBackendInProcess) Search(collection, query string, maxResults int) ([]CollectionSearchResult, error) {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	if maxResults <= 0 {
		entries := kb.ListDocuments()
		if len(entries) >= 5 {
			maxResults = 5
		} else {
			maxResults = 1
		}
	}
	results, err := kb.Search(query, maxResults)
	if err != nil {
		return nil, err
	}
	out := make([]CollectionSearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, CollectionSearchResult{
			ID:         r.ID,
			Content:    r.Content,
			Metadata:   r.Metadata,
			Similarity: r.Similarity,
		})
	}
	return out, nil
}

func (b *collectionsBackendInProcess) Reset(collection string) error {
	b.state.mu.Lock()
	kb, exists := b.state.collections[collection]
	if exists {
		delete(b.state.collections, collection)
	}
	b.state.mu.Unlock()
	if !exists {
		return fmt.Errorf("collection not found: %s", collection)
	}
	return kb.Reset()
}

func (b *collectionsBackendInProcess) DeleteEntry(collection, entry string) ([]string, error) {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	if err := kb.RemoveEntry(entry); err != nil {
		return nil, err
	}
	return kb.ListDocuments(), nil
}

func (b *collectionsBackendInProcess) AddSource(collection, url string, intervalMin int) error {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return fmt.Errorf("collection not found: %s", collection)
	}
	b.state.sourceManager.RegisterCollection(collection, kb)
	return b.state.sourceManager.AddSource(collection, url, time.Duration(intervalMin)*time.Minute)
}

func (b *collectionsBackendInProcess) RemoveSource(collection, url string) error {
	return b.state.sourceManager.RemoveSource(collection, url)
}

func (b *collectionsBackendInProcess) ListSources(collection string) ([]CollectionSourceInfo, error) {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	srcs := kb.GetExternalSources()
	out := make([]CollectionSourceInfo, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, CollectionSourceInfo{
			URL:            s.URL,
			UpdateInterval: int(s.UpdateInterval.Minutes()),
			LastUpdate:     s.LastUpdate,
		})
	}
	return out, nil
}

func (b *collectionsBackendInProcess) EntryExists(collection, entry string) bool {
	b.state.mu.RLock()
	kb, exists := b.state.collections[collection]
	b.state.mu.RUnlock()
	if !exists {
		return false
	}
	return kb.EntryExists(entry)
}

// NewInProcessCollectionsBackend creates in-process state (load from disk, start sourceManager) and returns
// a CollectionsBackend and the state. The caller should set app.collectionsState = state for RAG provider.
func NewInProcessCollectionsBackend(cfg *Config) (CollectionsBackend, *collectionsState) {
	state := &collectionsState{
		collections:   collectionList{},
		sourceManager: rag.NewSourceManager(&sources.Config{}),
	}

	openaiConfig := openai.DefaultConfig(cfg.LLMAPIKey)
	openaiConfig.BaseURL = cfg.LLMAPIURL
	openAIClient := openai.NewClientWithConfig(openaiConfig)

	os.MkdirAll(cfg.CollectionDBPath, 0755)
	os.MkdirAll(cfg.FileAssets, 0755)

	colls := rag.ListAllCollections(cfg.CollectionDBPath)
	for _, c := range colls {
		collection := newVectorEngine(cfg.VectorEngine, openAIClient, cfg.LLMAPIURL, cfg.LLMAPIKey, c, cfg.CollectionDBPath, cfg.FileAssets, cfg.EmbeddingModel, cfg.DatabaseURL, cfg.MaxChunkingSize, cfg.ChunkOverlap)
		if collection != nil {
			state.collections[c] = collection
			state.sourceManager.RegisterCollection(c, collection)
		}
	}

	state.ensureCollection = func(name string) (*rag.PersistentKB, bool) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if kb, ok := state.collections[name]; ok && kb != nil {
			return kb, true
		}
		collection := newVectorEngine(cfg.VectorEngine, openAIClient, cfg.LLMAPIURL, cfg.LLMAPIKey, name, cfg.CollectionDBPath, cfg.FileAssets, cfg.EmbeddingModel, cfg.DatabaseURL, cfg.MaxChunkingSize, cfg.ChunkOverlap)
		if collection == nil {
			return nil, false
		}
		state.collections[name] = collection
		state.sourceManager.RegisterCollection(name, collection)
		return collection, true
	}

	state.sourceManager.Start()

	backend := &collectionsBackendInProcess{state: state, cfg: cfg, openAIClient: openAIClient}
	return backend, state
}
