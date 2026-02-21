package webui

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/localrecall/rag"
	"github.com/mudler/xlog"
)

// internalRAGAdapter implements agent.RAGDB by calling the in-process *rag.PersistentKB directly (no HTTP).
type internalRAGAdapter struct {
	mu         sync.RWMutex
	collection string
	kb         *rag.PersistentKB
}

var _ agent.RAGDB = (*internalRAGAdapter)(nil)

func newInternalRAGAdapter(collection string, kb *rag.PersistentKB) *internalRAGAdapter {
	return &internalRAGAdapter{collection: collection, kb: kb}
}

func (a *internalRAGAdapter) Store(s string) error {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return fmt.Errorf("collection not available")
	}
	t := time.Now()
	dateTime := t.Format("2006-01-02-15-04-05")
	hash := md5.Sum([]byte(s))
	fileName := fmt.Sprintf("%s-%s.txt", dateTime, hex.EncodeToString(hash[:]))
	tempdir, err := os.MkdirTemp("", "localrag")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempdir)
	f := filepath.Join(tempdir, fileName)
	if err := os.WriteFile(f, []byte(s), 0644); err != nil {
		return err
	}
	meta := map[string]string{"created_at": t.Format(time.RFC3339)}
	return kb.Store(f, meta)
}

func (a *internalRAGAdapter) Reset() error {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return fmt.Errorf("collection not available")
	}
	return kb.Reset()
}

func (a *internalRAGAdapter) Search(s string, similarEntries int) ([]string, error) {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return nil, fmt.Errorf("collection not available")
	}
	results, err := kb.Search(s, similarEntries)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, fmt.Sprintf("%s (%+v)", r.Content, r.Metadata))
	}
	return out, nil
}

func (a *internalRAGAdapter) Count() int {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return 0
	}
	return kb.Count()
}

// internalCompactionAdapter implements state.KBCompactionClient for the same in-process collection.
type internalCompactionAdapter struct {
	mu         sync.RWMutex
	collection string
	kb         *rag.PersistentKB
}

var _ state.KBCompactionClient = (*internalCompactionAdapter)(nil)

func (a *internalCompactionAdapter) Collection() string {
	return a.collection
}

func (a *internalCompactionAdapter) ListEntries() ([]string, error) {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return nil, fmt.Errorf("collection not available")
	}
	return kb.ListDocuments(), nil
}

func (a *internalCompactionAdapter) GetEntryContent(entry string) (content string, chunkCount int, err error) {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return "", 0, fmt.Errorf("collection not available")
	}
	return kb.GetEntryFileContent(entry)
}

func (a *internalCompactionAdapter) Store(filePath string) error {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return fmt.Errorf("collection not available")
	}
	meta := map[string]string{"created_at": time.Now().Format(time.RFC3339)}
	return kb.Store(filePath, meta)
}

func (a *internalCompactionAdapter) DeleteEntry(entry string) error {
	a.mu.RLock()
	kb := a.kb
	a.mu.RUnlock()
	if kb == nil {
		return fmt.Errorf("collection not available")
	}
	return kb.RemoveEntry(entry)
}

// CollectionsRAGProvider returns a provider that the pool can use when no LocalRAG URL is set.
// It returns (RAGDB, KBCompactionClient, true) for a collection name, creating the collection on first use if needed.
func (app *App) CollectionsRAGProvider() func(collectionName string) (agent.RAGDB, state.KBCompactionClient, bool) {
	return func(collectionName string) (agent.RAGDB, state.KBCompactionClient, bool) {
		if app.collectionsState == nil {
			return nil, nil, false
		}
		name := strings.TrimSpace(strings.ToLower(collectionName))
		if name == "" {
			return nil, nil, false
		}
		var kb *rag.PersistentKB
		app.collectionsState.mu.RLock()
		kb, ok := app.collectionsState.collections[name]
		ensure := app.collectionsState.ensureCollection
		app.collectionsState.mu.RUnlock()
		if !ok || kb == nil {
			if ensure == nil {
				xlog.Debug("internal RAG: no ensureCollection", "collection", name)
				return nil, nil, false
			}
			var created bool
			kb, created = ensure(name)
			if !created || kb == nil {
				xlog.Debug("internal RAG: ensure collection failed", "collection", name)
				return nil, nil, false
			}
		}
		ragAdapter := newInternalRAGAdapter(name, kb)
		compAdapter := &internalCompactionAdapter{collection: name, kb: kb}
		return ragAdapter, compAdapter, true
	}
}
