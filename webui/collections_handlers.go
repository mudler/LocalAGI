package webui

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mudler/localrecall/rag"
	"github.com/mudler/xlog"
)

type collectionList map[string]*rag.PersistentKB

// collectionsState holds in-memory state for the collections API.
type collectionsState struct {
	mu               sync.RWMutex
	collections      collectionList
	sourceManager    *rag.SourceManager
	ensureCollection func(name string) (*rag.PersistentKB, bool) // get-or-create for internal RAG (agent name as collection)
}

// APIResponse represents a standardized API response (LocalRecall contract).
type collectionsAPIResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	Data    interface{}          `json:"data,omitempty"`
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

// collectionsAPIKeyFromRequest returns the API key from the same sources as the main keyauth: Authorization, x-api-key, xi-api-key, cookie:token.
func collectionsAPIKeyFromRequest(c *fiber.Ctx) string {
	if v := c.Get("Authorization"); v != "" {
		return strings.TrimPrefix(strings.TrimSpace(v), "Bearer ")
	}
	if v := c.Get("x-api-key"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := c.Get("xi-api-key"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := c.Cookies("token"); v != "" {
		return v
	}
	return ""
}

// RegisterCollectionRoutes mounts /api/collections* routes. backend is either from NewInProcessCollectionsBackend or NewCollectionsBackendHTTP.
func (app *App) RegisterCollectionRoutes(webapp *fiber.App, cfg *Config, backend CollectionsBackend) {
	webapp.Post("/api/collections", app.createCollection(backend))
	webapp.Get("/api/collections", app.listCollections(backend))
	webapp.Post("/api/collections/:name/upload", app.uploadFile(backend))
	webapp.Get("/api/collections/:name/entries", app.listFiles(backend))
	webapp.Get("/api/collections/:name/entries/*", app.getEntryContent(backend))
	webapp.Post("/api/collections/:name/search", app.searchCollection(backend))
	webapp.Post("/api/collections/:name/reset", app.resetCollection(backend))
	webapp.Delete("/api/collections/:name/entry/delete", app.deleteEntryFromCollection(backend))
	webapp.Post("/api/collections/:name/sources", app.registerExternalSource(backend))
	webapp.Delete("/api/collections/:name/sources", app.removeExternalSource(backend))
	webapp.Get("/api/collections/:name/sources", app.listSources(backend))
}

func collectionErrStatus(err error, collection string) int {
	if err == nil {
		return 0
	}
	if strings.Contains(err.Error(), "collection not found") {
		return fiber.StatusNotFound
	}
	if strings.Contains(err.Error(), "entry not found") {
		return fiber.StatusNotFound
	}
	return fiber.StatusInternalServerError
}

func (app *App) createCollection(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var r struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}
		if err := backend.CreateCollection(r.Name); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to create collection", err.Error()))
		}
		return c.Status(fiber.StatusCreated).JSON(collectionsSuccessResponse("Collection created successfully", map[string]interface{}{
			"name":       r.Name,
			"created_at": time.Now().Format(time.RFC3339),
		}))
	}
}

func (app *App) listCollections(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		collectionsList, err := backend.ListCollections()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to list collections", err.Error()))
		}
		return c.JSON(collectionsSuccessResponse("Collections retrieved successfully", map[string]interface{}{
			"collections": collectionsList,
			"count":       len(collectionsList),
		}))
	}
}

func (app *App) uploadFile(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
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

		if backend.EntryExists(name, file.Filename) {
			xlog.Info("Entry already exists")
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeConflict, "Entry already exists", fmt.Sprintf("File '%s' has already been uploaded to collection '%s'", file.Filename, name)))
		}

		if err := backend.Upload(name, file.Filename, f); err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				return c.Status(status).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
			}
			xlog.Error("Failed to store file", err)
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to store file", err.Error()))
		}

		now := time.Now().Format(time.RFC3339)
		return c.JSON(collectionsSuccessResponse("File uploaded successfully", map[string]interface{}{
			"filename":   file.Filename,
			"collection": name,
			"created_at": now,
		}))
	}
}

func (app *App) listFiles(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		entries, err := backend.ListEntries(name)
		if err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				return c.Status(status).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to list entries", err.Error()))
		}
		return c.JSON(collectionsSuccessResponse("Entries retrieved successfully", map[string]interface{}{
			"collection": name,
			"entries":    entries,
			"count":      len(entries),
		}))
	}
}

// getEntryContent handles GET /api/collections/:name/entries/:entry (Fiber uses * for the rest of path).
func (app *App) getEntryContent(backend CollectionsBackend) func(c *fiber.Ctx) error {
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

		content, chunkCount, err := backend.GetEntryContent(name, entry)
		if err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				if strings.Contains(err.Error(), "entry not found") {
					return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Entry not found", fmt.Sprintf("Entry '%s' does not exist in collection '%s'", entry, name)))
				}
				return c.Status(fiber.StatusNotFound).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
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

func (app *App) searchCollection(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		var r struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		results, err := backend.Search(name, r.Query, r.MaxResults)
		if err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				return c.Status(status).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
			}
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

func (app *App) resetCollection(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		if err := backend.Reset(name); err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				return c.Status(status).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to reset collection", err.Error()))
		}
		return c.JSON(collectionsSuccessResponse("Collection reset successfully", map[string]interface{}{
			"collection": name,
			"reset_at":   time.Now().Format(time.RFC3339),
		}))
	}
}

func (app *App) deleteEntryFromCollection(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		var r struct {
			Entry string `json:"entry"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		remainingEntries, err := backend.DeleteEntry(name, r.Entry)
		if err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				return c.Status(status).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to remove entry", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("Entry deleted successfully", map[string]interface{}{
			"deleted_entry":     r.Entry,
			"remaining_entries": remainingEntries,
			"entry_count":       len(remainingEntries),
		}))
	}
}

func (app *App) registerExternalSource(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
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

		if err := backend.AddSource(name, r.URL, r.UpdateInterval); err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				return c.Status(status).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to register source", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("External source registered successfully", map[string]interface{}{
			"collection":      name,
			"url":             r.URL,
			"update_interval": r.UpdateInterval,
		}))
	}
}

func (app *App) removeExternalSource(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		var r struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&r); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(collectionsErrorResponse(errCodeInvalidRequest, "Invalid request", err.Error()))
		}

		if err := backend.RemoveSource(name, r.URL); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to remove source", err.Error()))
		}

		return c.JSON(collectionsSuccessResponse("External source removed successfully", map[string]interface{}{
			"collection": name,
			"url":        r.URL,
		}))
	}
}

func (app *App) listSources(backend CollectionsBackend) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		srcs, err := backend.ListSources(name)
		if err != nil {
			if status := collectionErrStatus(err, name); status == fiber.StatusNotFound {
				return c.Status(status).JSON(collectionsErrorResponse(errCodeNotFound, "Collection not found", fmt.Sprintf("Collection '%s' does not exist", name)))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(collectionsErrorResponse(errCodeInternalError, "Failed to list sources", err.Error()))
		}

		sourcesList := make([]map[string]interface{}, 0, len(srcs))
		for _, source := range srcs {
			sourcesList = append(sourcesList, map[string]interface{}{
				"url":             source.URL,
				"update_interval": source.UpdateInterval,
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
