package webui

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mudler/LocalAGI/pkg/localrag"
)

// collectionsBackendHTTP implements CollectionsBackend using the LocalRAG HTTP API.
type collectionsBackendHTTP struct {
	client *localrag.Client
}

var _ CollectionsBackend = (*collectionsBackendHTTP)(nil)

// NewCollectionsBackendHTTP returns a CollectionsBackend that delegates to the given HTTP client.
func NewCollectionsBackendHTTP(client *localrag.Client) CollectionsBackend {
	return &collectionsBackendHTTP{client: client}
}

func (b *collectionsBackendHTTP) ListCollections() ([]string, error) {
	return b.client.ListCollections()
}

func (b *collectionsBackendHTTP) CreateCollection(name string) error {
	return b.client.CreateCollection(name)
}

func (b *collectionsBackendHTTP) Upload(collection, filename string, fileBody io.Reader) error {
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
	if err := out.Close(); err != nil {
		return err
	}
	return b.client.Store(collection, tmpPath)
}

func (b *collectionsBackendHTTP) ListEntries(collection string) ([]string, error) {
	return b.client.ListEntries(collection)
}

func (b *collectionsBackendHTTP) GetEntryContent(collection, entry string) (string, int, error) {
	return b.client.GetEntryContent(collection, entry)
}

func (b *collectionsBackendHTTP) Search(collection, query string, maxResults int) ([]CollectionSearchResult, error) {
	if maxResults <= 0 {
		maxResults = 5
	}
	results, err := b.client.Search(collection, query, maxResults)
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

func (b *collectionsBackendHTTP) Reset(collection string) error {
	return b.client.Reset(collection)
}

func (b *collectionsBackendHTTP) DeleteEntry(collection, entry string) ([]string, error) {
	return b.client.DeleteEntry(collection, entry)
}

func (b *collectionsBackendHTTP) AddSource(collection, url string, intervalMin int) error {
	return b.client.AddSource(collection, url, intervalMin)
}

func (b *collectionsBackendHTTP) RemoveSource(collection, url string) error {
	return b.client.RemoveSource(collection, url)
}

func (b *collectionsBackendHTTP) ListSources(collection string) ([]CollectionSourceInfo, error) {
	srcs, err := b.client.ListSources(collection)
	if err != nil {
		return nil, err
	}
	out := make([]CollectionSourceInfo, 0, len(srcs))
	for _, s := range srcs {
		var lastUpdate time.Time
		if s.LastUpdate != "" {
			lastUpdate, _ = time.Parse(time.RFC3339, s.LastUpdate)
		}
		out = append(out, CollectionSourceInfo{
			URL:            s.URL,
			UpdateInterval: s.UpdateInterval,
			LastUpdate:     lastUpdate,
		})
	}
	return out, nil
}

func (b *collectionsBackendHTTP) EntryExists(collection, entry string) bool {
	entries, err := b.client.ListEntries(collection)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e == entry {
			return true
		}
	}
	return false
}
