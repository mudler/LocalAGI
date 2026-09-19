package collections

import (
	"encoding/json"
	"fmt"
	"github.com/mudler/localrecall/rag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testModelBackend(t *testing.T, selected *CollectionModelSettings) (Backend, *State, *[]string) {
	t.Helper()
	var mu sync.Mutex
	models := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rerank" {
			if r.Header.Get("Authorization") != "Bearer secret" {
				t.Error("missing rerank authentication")
			}
			var body struct {
				Model     string
				Documents []string
			}
			json.NewDecoder(r.Body).Decode(&body)
			if body.Model != "ranker" {
				t.Errorf("reranker = %q", body.Model)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"results":[{"index":0,"relevance_score":0.8}]}`))
			return
		}
		var body struct{ Model string }
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		models = append(models, body.Model)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"embedding":[1,0,0],"index":0}],"model":"test","usage":{}}`))
	}))
	t.Cleanup(server.Close)
	b, st := NewInProcessBackend(&Config{LLMAPIURL: server.URL + "/v1", LLMAPIKey: "secret", CollectionDBPath: t.TempDir(), FileAssets: t.TempDir(), VectorEngine: "chromem", EmbeddingModel: "default", MaxChunkingSize: 200, ModelSettings: func(string) CollectionModelSettings { return *selected }})
	t.Cleanup(st.SourceManager.Stop)
	return b, st, &models
}

func TestCollectionModelOverridesAndReset(t *testing.T) {
	selected := CollectionModelSettings{EmbeddingModel: "custom", RerankerModel: "ranker"}
	b, st, models := testModelBackend(t, &selected)
	ragDB, _, ok := RAGProviderFromState(st)("agent")
	if !ok {
		t.Fatal("provider unavailable")
	}
	if err := ragDB.Store("a source document"); err != nil {
		t.Fatal(err)
	}
	result, err := b.Search("agent", "query", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Similarity != 0.8 || result[0].Metadata["source"] == "" {
		t.Fatalf("result lost score or citation: %+v", result)
	}
	embedded, err := ragDB.Search("query", 1)
	if err != nil || len(embedded) != 1 || !strings.Contains(embedded[0], "source") {
		t.Fatalf("embedded search: %v %v", embedded, err)
	}
	for _, model := range *models {
		if model != "custom" {
			t.Fatalf("unexpected embedding model %q", model)
		}
	}
	selected.EmbeddingModel = "changed"
	if _, err := b.Search("agent", "query", 1); err == nil || !strings.Contains(err.Error(), "reset") {
		t.Fatalf("expected model mismatch error, got %v", err)
	}
	if err := ragDB.Store("must not be indexed"); err == nil {
		t.Fatal("mixed embeddings accepted")
	}
	if err := b.Reset("agent"); err != nil {
		t.Fatal(err)
	}
	if err := ragDB.Store("fresh document"); err != nil {
		t.Fatal(err)
	}
	if got := (*models)[len(*models)-1]; got != "changed" {
		t.Fatalf("model after reset=%s", got)
	}
}

func TestRerankRejectsInvalidResults(t *testing.T) {
	for _, response := range []string{`{"results":[{"index":2,"relevance_score":1}]}`, `{"results":[{"index":0,"relevance_score":1},{"index":0,"relevance_score":0.5}]}`, `{"results":[{"index":0,"relevance_score":1e50}]}`, `{}`, `{"results":[]}`} {
		t.Run(response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(response)) }))
			defer server.Close()
			b := &backendInProcess{cfg: &Config{LLMAPIURL: server.URL}}
			if _, err := b.rerank("ranker", "query", []SearchResult{{Content: "one"}}, 1); err == nil {
				t.Fatal("invalid rerank response accepted")
			}
		})
	}
}

func TestRerankOrdersResultsAndPreservesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"index":0,"relevance_score":0.1},{"index":1,"relevance_score":0.9}]}`))
	}))
	defer server.Close()
	b := &backendInProcess{cfg: &Config{LLMAPIURL: server.URL}}
	out, err := b.rerank("ranker", "query", []SearchResult{{ID: "first", Content: "one"}, {ID: "second", Content: "two", Metadata: map[string]string{"source": "citation"}}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ID != "second" || out[0].Metadata["source"] != "citation" {
		t.Fatalf("wrong order or citation: %+v", out)
	}
}

func TestEmbeddingIdentityCannotEscapeDirectory(t *testing.T) {
	b := &backendInProcess{cfg: &Config{CollectionDBPath: t.TempDir()}}
	path := b.identityPath("../../outside")
	if !strings.HasPrefix(path, b.cfg.CollectionDBPath+"/embedding-models/") {
		t.Fatalf("identity escapes its directory: %s", path)
	}
}

func TestEmbeddingIdentitySurvivesRestartAndLegacyMigration(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint(legacy), func(t *testing.T) {
			selected := CollectionModelSettings{}
			backend, st, models := testModelBackend(t, &selected)
			b := backend.(*backendInProcess)
			db, _, _ := RAGProviderFromState(st)("agent")
			if err := db.Store("persisted document"); err != nil {
				t.Fatal(err)
			}
			st.SourceManager.Stop()
			if legacy {
				if err := os.Remove(b.identityPath("agent")); err != nil {
					t.Fatal(err)
				}
			}
			selected.EmbeddingModel = "changed"
			restarted, nextState := NewInProcessBackend(b.cfg)
			defer nextState.SourceManager.Stop()
			if _, err := restarted.Search("agent", "query", 1); err == nil {
				t.Fatal("restart accepted changed embedding model")
			}
			entries, err := restarted.ListEntries("agent")
			if err != nil || len(entries) != 1 {
				t.Fatalf("cannot inspect mismatched collection: %v %v", entries, err)
			}
			for _, model := range *models {
				if model != "default" {
					t.Fatalf("constructor used changed model: %s", model)
				}
			}
			names, err := restarted.ListCollections()
			if err != nil || len(names) != 1 {
				t.Fatalf("sidecar exposed as a collection: %v %v", names, err)
			}
			if err := restarted.Reset("agent"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProviderDoesNotResolveModelsWhileConstructing(t *testing.T) {
	selected := CollectionModelSettings{}
	backend, st, _ := testModelBackend(t, &selected)
	calls := 0
	backend.(*backendInProcess).cfg.ModelSettings = func(string) CollectionModelSettings { calls++; return selected }
	db, _, ok := RAGProviderFromState(st)("agent")
	if !ok || db == nil || calls != 0 {
		t.Fatalf("provider invoked resolver: %d", calls)
	}
}

func TestRerankDoesNotFollowRedirectsOrHideErrors(t *testing.T) {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer target.Close()
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusInternalServerError} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", target.URL)
			w.WriteHeader(status)
		}))
		b := &backendInProcess{cfg: &Config{LLMAPIURL: server.URL, LLMAPIKey: "secret"}}
		if _, err := b.rerank("ranker", "query", []SearchResult{{Content: "one"}}, 1); err == nil {
			t.Fatal("rerank failure ignored")
		}
		server.Close()
	}
	if reached {
		t.Fatal("rerank request followed credential-bearing redirect")
	}
}

func TestEmbeddedStorePreservesCompactionDatePrefix(t *testing.T) {
	selected := CollectionModelSettings{}
	b, st, _ := testModelBackend(t, &selected)
	db, _, _ := RAGProviderFromState(st)("agent")
	if err := db.Store("document"); err != nil {
		t.Fatal(err)
	}
	entries, err := b.ListEntries("agent")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasPrefix(filepath.Base(entries[0]), time.Now().Format("2006-01-02-")) {
		t.Fatalf("entry lacks compaction date prefix: %v", entries)
	}
}

func TestResetWithSourcesRequiresRestartBeforeRecreation(t *testing.T) {
	selected := CollectionModelSettings{}
	backend, st, _ := testModelBackend(t, &selected)
	b := backend.(*backendInProcess)
	if err := b.CreateCollection("agent"); err != nil {
		t.Fatal(err)
	}
	kb := st.Collections["agent"]
	if err := kb.AddExternalSource(&rag.ExternalSource{URL: "https://example.invalid/source", UpdateInterval: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if err := b.Reset("agent"); err != nil {
		t.Fatal(err)
	}
	selected.EmbeddingModel = "changed"
	if err := b.CreateCollection("agent"); err == nil || !strings.Contains(err.Error(), "restart") {
		t.Fatalf("expected restart guard, got %v", err)
	}
	if model, err := b.storedModel("agent", "changed"); err != nil || model != "default" {
		t.Fatalf("forgot identity while source might still write: %s %v", model, err)
	}
}

func TestRemovedSourcesStillPreventModelSwitch(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(fmt.Sprintf("reset=%t", reset), func(t *testing.T) {
			selected := CollectionModelSettings{}
			backend, st, _ := testModelBackend(t, &selected)
			b := backend.(*backendInProcess)
			if err := b.CreateCollection("agent"); err != nil {
				t.Fatal(err)
			}
			url := "https://example.invalid/source"
			// Seed persisted metadata without starting a network fetch. Removal
			// must remember that a source could already have an in-flight fetch.
			if err := st.Collections["agent"].AddExternalSource(&rag.ExternalSource{URL: url, UpdateInterval: time.Hour}); err != nil {
				t.Fatal(err)
			}
			if err := b.RemoveSource("agent", url); err != nil {
				t.Fatal(err)
			}
			if reset {
				if err := b.Reset("agent"); err != nil {
					t.Fatal(err)
				}
			}
			selected.EmbeddingModel = "changed"
			if err := b.CreateCollection("agent"); err == nil {
				t.Fatal("removed source allowed model switch while its fetch may still write")
			}
			if model, err := b.storedModel("agent", "changed"); err != nil || model != "default" {
				t.Fatalf("source identity changed: %q %v", model, err)
			}
		})
	}
}

func TestFailedModelSwitchDoesNotReuseOldEngine(t *testing.T) {
	selected := CollectionModelSettings{}
	backend, st, models := testModelBackend(t, &selected)
	b := backend.(*backendInProcess)
	if err := b.CreateCollection("agent"); err != nil {
		t.Fatal(err)
	}
	selected.EmbeddingModel = "changed"
	// Force the real constructor to fail after the new identity is saved.
	b.cfg.VectorEngine = "unavailable"
	if err := b.CreateCollection("agent"); err == nil {
		t.Fatal("expected engine initialization failure")
	}
	b.cfg.VectorEngine = "chromem"
	db, _, _ := RAGProviderFromState(st)("agent")
	if err := db.Store("retry after engine recovery"); err != nil {
		t.Fatal(err)
	}
	if got := (*models)[len(*models)-1]; got != "changed" {
		t.Fatalf("retry wrote vectors using %q under changed identity", got)
	}
}
