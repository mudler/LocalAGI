package webui

import (
	"io"
	"time"
)

// CollectionSearchResult is a single search result (content + metadata) for API responses.
type CollectionSearchResult struct {
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	ID         string            `json:"id,omitempty"`
	Similarity float32           `json:"similarity,omitempty"`
}

// CollectionSourceInfo is a single external source for a collection.
type CollectionSourceInfo struct {
	URL             string    `json:"url"`
	UpdateInterval  int       `json:"update_interval"` // minutes
	LastUpdate      time.Time `json:"last_update"`
}

// CollectionsBackend is the interface used by REST handlers for collection operations.
// It is implemented by in-process state (embedded) or by an HTTP client (when LocalRAG URL is set).
type CollectionsBackend interface {
	ListCollections() ([]string, error)
	CreateCollection(name string) error
	Upload(collection, filename string, fileBody io.Reader) error
	ListEntries(collection string) ([]string, error)
	GetEntryContent(collection, entry string) (content string, chunkCount int, err error)
	Search(collection, query string, maxResults int) ([]CollectionSearchResult, error)
	Reset(collection string) error
	DeleteEntry(collection, entry string) (remainingEntries []string, err error)
	AddSource(collection, url string, intervalMin int) error
	RemoveSource(collection, url string) error
	ListSources(collection string) ([]CollectionSourceInfo, error)
	// EntryExists is used by upload handler to avoid duplicate entries.
	EntryExists(collection, entry string) bool
}
