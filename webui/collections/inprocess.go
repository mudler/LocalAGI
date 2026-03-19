package collections

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

// backendInProcess implements Backend using in-process state.
type backendInProcess struct {
	state        *State
	cfg          *Config
	openAIClient *openai.Client
}

var _ Backend = (*backendInProcess)(nil)

func (b *backendInProcess) ListCollections() ([]string, error) {
	return rag.ListAllCollections(b.cfg.CollectionDBPath), nil
}

func (b *backendInProcess) CreateCollection(name string) error {
	collection := newVectorEngine(b.cfg.VectorEngine, b.openAIClient, b.cfg.LLMAPIURL, b.cfg.LLMAPIKey, name, b.cfg.CollectionDBPath, b.cfg.FileAssets, b.cfg.EmbeddingModel, b.cfg.DatabaseURL, b.cfg.MaxChunkingSize, b.cfg.ChunkOverlap)
	if collection == nil {
		return fmt.Errorf("unsupported or misconfigured vector engine")
	}
	b.state.Mu.Lock()
	b.state.Collections[name] = collection
	b.state.SourceManager.RegisterCollection(name, collection)
	b.state.Mu.Unlock()
	return nil
}

func (b *backendInProcess) Upload(collection, filename string, fileBody io.Reader) error {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return fmt.Errorf("collection not found: %s", collection)
	}
	// Write to a temp file; kb.Store will copy it into the correct UUID
	// subdirectory under the collection's asset dir.
	tmpDir, err := os.MkdirTemp("", "localagi-upload")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, filename)
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, fileBody); err != nil {
		out.Close()
		return err
	}
	out.Close()
	now := time.Now().Format(time.RFC3339)
	return kb.Store(tmpPath, map[string]string{"created_at": now})
}

func (b *backendInProcess) ListEntries(collection string) ([]string, error) {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	keys := kb.ListDocuments()
	entries := make([]string, len(keys))
	for i, k := range keys {
		entries[i] = filepath.Base(k)
	}
	return entries, nil
}

func (b *backendInProcess) GetEntryContent(collection, entry string) (string, int, error) {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return "", 0, fmt.Errorf("collection not found: %s", collection)
	}
	return kb.GetEntryFileContent(entry)
}

func (b *backendInProcess) Search(collection, query string, maxResults int) ([]SearchResult, error) {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	if maxResults <= 0 {
		keys := kb.ListDocuments()
		if len(keys) >= 5 {
			maxResults = 5
		} else {
			maxResults = 1
		}
	}
	results, err := kb.Search(query, maxResults)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, SearchResult{
			ID:         r.ID,
			Content:    r.Content,
			Metadata:   r.Metadata,
			Similarity: r.Similarity,
		})
	}
	return out, nil
}

func (b *backendInProcess) Reset(collection string) error {
	b.state.Mu.Lock()
	kb, exists := b.state.Collections[collection]
	if exists {
		delete(b.state.Collections, collection)
	}
	b.state.Mu.Unlock()
	if !exists {
		return fmt.Errorf("collection not found: %s", collection)
	}
	return kb.Reset()
}

func (b *backendInProcess) DeleteEntry(collection, entry string) ([]string, error) {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	if err := kb.RemoveEntry(entry); err != nil {
		return nil, err
	}
	keys := kb.ListDocuments()
	entries := make([]string, len(keys))
	for i, k := range keys {
		entries[i] = filepath.Base(k)
	}
	return entries, nil
}

func (b *backendInProcess) AddSource(collection, url string, intervalMin int) error {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return fmt.Errorf("collection not found: %s", collection)
	}
	b.state.SourceManager.RegisterCollection(collection, kb)
	return b.state.SourceManager.AddSource(collection, url, time.Duration(intervalMin)*time.Minute)
}

func (b *backendInProcess) RemoveSource(collection, url string) error {
	return b.state.SourceManager.RemoveSource(collection, url)
}

func (b *backendInProcess) ListSources(collection string) ([]SourceInfo, error) {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	srcs := kb.GetExternalSources()
	out := make([]SourceInfo, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, SourceInfo{
			URL:            s.URL,
			UpdateInterval: int(s.UpdateInterval.Minutes()),
			LastUpdate:     s.LastUpdate,
		})
	}
	return out, nil
}

func (b *backendInProcess) GetEntryFilePath(collection, entry string) (string, error) {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return "", fmt.Errorf("collection not found: %s", collection)
	}
	return kb.GetEntryFilePath(entry)
}

func (b *backendInProcess) EntryExists(collection, entry string) bool {
	b.state.Mu.RLock()
	kb, exists := b.state.Collections[collection]
	b.state.Mu.RUnlock()
	if !exists {
		return false
	}
	return kb.EntryExists(entry)
}

// NewInProcessBackend creates in-process state (load from disk, start sourceManager) and returns
// a Backend and the State. The caller can use RAGProviderFromState to create a RAG provider.
func NewInProcessBackend(cfg *Config) (Backend, *State) {
	st := &State{
		Collections:   CollectionList{},
		SourceManager: rag.NewSourceManager(&sources.Config{}),
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
			st.Collections[c] = collection
			st.SourceManager.RegisterCollection(c, collection)
		}
	}

	st.EnsureCollection = func(name string) (*rag.PersistentKB, bool) {
		st.Mu.Lock()
		defer st.Mu.Unlock()
		if kb, ok := st.Collections[name]; ok && kb != nil {
			return kb, true
		}
		collection := newVectorEngine(cfg.VectorEngine, openAIClient, cfg.LLMAPIURL, cfg.LLMAPIKey, name, cfg.CollectionDBPath, cfg.FileAssets, cfg.EmbeddingModel, cfg.DatabaseURL, cfg.MaxChunkingSize, cfg.ChunkOverlap)
		if collection == nil {
			return nil, false
		}
		st.Collections[name] = collection
		st.SourceManager.RegisterCollection(name, collection)
		return collection, true
	}

	st.SourceManager.Start()

	backend := &backendInProcess{state: st, cfg: cfg, openAIClient: openAIClient}
	return backend, st
}
