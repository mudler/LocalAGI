package webui

import (
	"crypto/subtle"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mudler/localrecall/rag"
	"github.com/mudler/localrecall/rag/sources"
	"github.com/mudler/xlog"
	"github.com/sashabaranov/go-openai"
)

type collectionList map[string]*rag.PersistentKB

// collectionsState holds in-memory state for the collections API.
type collectionsState struct {
	mu              sync.RWMutex
	collections     collectionList
	sourceManager   *rag.SourceManager
	ensureCollection func(name string) (*rag.PersistentKB, bool) // get-or-create for internal RAG (agent name as collection)
}

// APIResponse represents a standardized API response (LocalRecall contract).
type collectionsAPIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *collectionsAPIError `json:"error,omitempty"`
}

type collectionsAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

const (
	errCodeNotFound       = "NOT_FOUND"
	errCodeInvalidRequest = "INVALID_REQUEST"
	errCodeInternalError  = "INTERNAL_ERROR"
	errCodeUnauthorized   = "UNAUTHORIZED"
	errCodeConflict       = "CONFLICT"
)

func collectionsSuccessResponse(message string, data interface{}) collectionsAPIResponse {
	return collectionsAPIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func collectionsErrorResponse(code, message, details string) collectionsAPIResponse {
	return collectionsAPIResponse{
		Success: false,
		Error: &collectionsAPIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

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

// RegisterCollectionRoutes mounts /api/collections* routes and initializes collections state.
func (app *App) RegisterCollectionRoutes(webapp *fiber.App, cfg *Config) {
	state := &collectionsState{
		collections:   collectionList{},
		sourceManager: rag.NewSourceManager(&sources.Config{}),
	}

	openaiConfig := openai.DefaultConfig(cfg.LLMAPIKey)
	openaiConfig.BaseURL = cfg.LLMAPIURL
	openAIClient := openai.NewClientWithConfig(openaiConfig)

	// Ensure dirs exist
	os.MkdirAll(cfg.CollectionDBPath, 0755)
	os.MkdirAll(cfg.FileAssets, 0755)

	// Load existing collections from disk
	colls := rag.ListAllCollections(cfg.CollectionDBPath)
	for _, c := range colls {
		collection := newVectorEngine(cfg.VectorEngine, openAIClient, cfg.LLMAPIURL, cfg.LLMAPIKey, c, cfg.CollectionDBPath, cfg.FileAssets, cfg.EmbeddingModel, cfg.DatabaseURL, cfg.MaxChunkingSize, cfg.ChunkOverlap)
		if collection != nil {
			state.collections[c] = collection
			state.sourceManager.RegisterCollection(c, collection)
		}
	}

	// Get-or-create for internal RAG (agents use collection name = agent name)
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

	app.collectionsState = state

	// Optional API key middleware for /api/collections
	apiKeys := cfg.CollectionAPIKeys
	if len(apiKeys) == 0 {
		apiKeys = cfg.ApiKeys
	}
	if len(apiKeys) > 0 {
		webapp.Use("/api/collections", func(c *fiber.Ctx) error {
			apiKey := c.Get("Authorization")
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
			for _, validKey := range apiKeys {
				if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validKey)) == 1 {
					return c.Next()
				}
			}
			return c.Status(fiber.StatusUnauthorized).JSON(collectionsErrorResponse(errCodeUnauthorized, "Unauthorized", "Invalid or missing API key"))
		})
	}

	// Route handlers close over state and config
	webapp.Post("/api/collections", app.createCollection(state, cfg, openAIClient))
	webapp.Get("/api/collections", app.listCollections(cfg))
	webapp.Post("/api/collections/:name/upload", app.uploadFile(state, cfg))
	webapp.Get("/api/collections/:name/entries", app.listFiles(state))
	webapp.Get("/api/collections/:name/entries/*", app.getEntryContent(state))
	webapp.Post("/api/collections/:name/search", app.searchCollection(state))
	webapp.Post("/api/collections/:name/reset", app.resetCollection(state))
	webapp.Delete("/api/collections/:name/entry/delete", app.deleteEntryFromCollection(state))
	webapp.Post("/api/collections/:name/sources", app.registerExternalSource(state))
	webapp.Delete("/api/collections/:name/sources", app.removeExternalSource(state))
	webapp.Get("/api/collections/:name/sources", app.listSources(state))
}

func (app *App) createCollection(state *collectionsState, cfg *Config, client *openai.Client) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var r struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		collection := newVectorEngine(cfg.VectorEngine, client, cfg.LLMAPIURL, cfg.LLMAPIKey, r.Name, cfg.CollectionDBPath, cfg.FileAssets, cfg.EmbeddingModel, cfg.DatabaseURL, cfg.MaxChunkingSize, cfg.ChunkOverlap)
		if collection == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to create collection", "unsupported or misconfigured vector engine"))
		}

		state.mu.Lock()
		state.collections[r.Name] = collection
		state.sourceManager.RegisterCollection(r.Name, collection)
		state.mu.Unlock()

		return c.Status(fiber.StatusCreated).JSON(collectionsSuccessResponse("Collection created successfully", map[string]interface{}{
			"name":       r.Name,
			"created_at": time.Now().Format(time.RFC3339),
		}))
	}
}

func (app *App) listCollections(cfg *Config) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		collectionsList := rag.ListAllCollections(cfg.CollectionDBPath)
		return c.JSON(collectionsSuccessResponse("Collections retrieved successfully", map[string]interface{}{
			"collections": collectionsList,
			"count":       len(collectionsList),
		}))
	}
}

func (app *App) uploadFile(state *collectionsState, cfg *Config) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		state.mu.RLock()
		collection, exists := state.collections[name]
		state.mu.RUnlock()
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		file, err := c.FormFile("file")
		if err != nil {
			xlog.Error("Failed to read file", err)
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Failed to read file", err.Error()))
		}

		f, err := file.Open()
		if err != nil {
			xlog.Error("Failed to open file", err)
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Failed to open file", err.Error()))
		}
		defer f.Close()

		filePath := filepath.Join(cfg.FileAssets, file.Filename)
		out, err := os.Create(filePath)
		if err != nil {
			xlog.Error("Failed to create file", err)
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to create file", err.Error()))
		}
		defer out.Close()

		_, err = io.Copy(out, f)
		if err != nil {
			xlog.Error("Failed to copy file", err)
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to copy file", err.Error()))
		}

		if collection.EntryExists(file.Filename) {
			xlog.Info("Entry already exists")
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeConflict, "Entry already exists", fmt.Sprintf("File '%s' has already been uploaded to collection '%s'", file.Filename, name)))
		}

		now := time.Now().Format(time.RFC3339)
		if err := collection.Store(filePath, map[string]string{"created_at": now}); err != nil {
			xlog.Error("Failed to store file", err)
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to store file", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("File uploaded successfully", map[string]interface{}{
			"filename":   file.Filename,
			"collection": name,
			"created_at": now,
		}))
	}
}

func (app *App) listFiles(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		state.mu.RLock()
		collection, exists := state.collections[name]
		state.mu.RUnlock()
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		entries := collection.ListDocuments()
		return c.JSON(collectionsSuccessResponse("Entries retrieved successfully", map[string]interface{}{
			"collection": name,
			"entries":    entries,
			"count":      len(entries),
		}))
	}
}

// getEntryContent handles GET /api/collections/:name/entries/:entry (Fiber uses * for the rest of path).
func (app *App) getEntryContent(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		entryParam := c.Params("*")
		if entryParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", "entry path required"))
		}
		entry, err := url.PathUnescape(entryParam)
		if err != nil {
			entry = entryParam
		}

		state.mu.RLock()
		collection, exists := state.collections[name]
		state.mu.RUnlock()
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		content, chunkCount, err := collection.GetEntryFileContent(entry)
		if err != nil {
			if strings.Contains(err.Error(), "entry not found") {
				return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Entry not found", fmt.Sprintf("Entry '%s' does not exist in collection '%s'", entry, name)))
			}
			if strings.Contains(err.Error(), "not implemented") || strings.Contains(err.Error(), "unsupported file type") {
				return c.Status(fiber.StatusNotImplemented).JSON(collectionsErrorResponse(errCodeInternalError, "Not supported", err.Error()))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to get entry content", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("Entry content retrieved successfully", map[string]interface{}{
			"collection":  name,
			"entry":       entry,
			"content":     content,
			"chunk_count": chunkCount,
		}))
	}
}

func (app *App) searchCollection(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		state.mu.RLock()
		collection, exists := state.collections[name]
		state.mu.RUnlock()
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		var r struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		if r.MaxResults == 0 {
			if len(collection.ListDocuments()) >= 5 {
				r.MaxResults = 5
			} else {
				r.MaxResults = 1
			}
		}

		results, err := collection.Search(r.Query, r.MaxResults)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to search collection", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("Search completed successfully", map[string]interface{}{
			"query":       r.Query,
			"max_results": r.MaxResults,
			"results":     results,
			"count":       len(results),
		}))
	}
}

func (app *App) resetCollection(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		state.mu.Lock()
		collection, exists := state.collections[name]
		if exists {
			delete(state.collections, name)
		}
		state.mu.Unlock()

		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		if err := collection.Reset(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to reset collection", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("Collection reset successfully", map[string]interface{}{
			"collection": name,
			"reset_at":   time.Now().Format(time.RFC3339),
		}))
	}
}

func (app *App) deleteEntryFromCollection(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		state.mu.RLock()
		collection, exists := state.collections[name]
		state.mu.RUnlock()
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		var r struct {
			Entry string `json:"entry"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		if err := collection.RemoveEntry(r.Entry); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to remove entry", err.Error()))
		}

		remainingEntries := collection.ListDocuments()
		return c.JSON(collectionsSuccessResponse("Entry deleted successfully", map[string]interface{}{
			"deleted_entry":     r.Entry,
			"remaining_entries": remainingEntries,
			"entry_count":       len(remainingEntries),
		}))
	}
}

func (app *App) registerExternalSource(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		state.mu.RLock()
		collection, exists := state.collections[name]
		state.mu.RUnlock()
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		var r struct {
			URL            string `json:"url"`
			UpdateInterval int    `json:"update_interval"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		if r.UpdateInterval < 1 {
			r.UpdateInterval = 60
		}

		state.sourceManager.RegisterCollection(name, collection)
		if err := state.sourceManager.AddSource(name, r.URL, time.Duration(r.UpdateInterval)*time.Minute); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to register source", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("External source registered successfully", map[string]interface{}{
			"collection":      name,
			"url":             r.URL,
			"update_interval": r.UpdateInterval,
		}))
	}
}

func (app *App) removeExternalSource(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")

		var r struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		if err := state.sourceManager.RemoveSource(name, r.URL); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to remove source", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("External source removed successfully", map[string]interface{}{
			"collection": name,
			"url":        r.URL,
		}))
	}
}

func (app *App) listSources(state *collectionsState) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		state.mu.RLock()
		collection, exists := state.collections[name]
		state.mu.RUnlock()
		if !exists {
			return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
		}

		srcs := collection.GetExternalSources()
		sourcesList := make([]map[string]interface{}, 0, len(srcs))
		for _, source := range srcs {
			sourcesList = append(sourcesList, map[string]interface{}{
				"url":             source.URL,
				"update_interval": int(source.UpdateInterval.Minutes()),
				"last_update":     source.LastUpdate.Format(time.RFC3339),
			})
		}

		return c.JSON(collectionsSuccessResponse("Sources retrieved successfully", map[string]interface{}{
			"collection": name,
			"sources":    sourcesList,
			"count":      len(sourcesList),
		}))
	}
}
