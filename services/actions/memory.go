package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// indexCache avoids opening the same Bleve index path multiple times, which would
// deadlock (Bleve uses file locks; a second Open() on the same path blocks).
var (
	indexCache   = map[string]bleve.Index{}
	indexCacheMu sync.Mutex
)

type MemoryActions struct {
	index             bleve.Index
	indexPath         string
	customName        string
	customDescription string
}

type AddToMemoryAction struct{ *MemoryActions }
type ListMemoryAction struct{ *MemoryActions }
type RemoveFromMemoryAction struct{ *MemoryActions }
type SearchMemoryAction struct{ *MemoryActions }

// MemoryEntry matches the MCP memory structure (Bleve-backed).
type MemoryEntry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// NewMemoryActions returns the four memory actions (Add, List, Remove, Search) using a Bleve index at indexPath.
func NewMemoryActions(indexPath string, config map[string]string) (*AddToMemoryAction, *ListMemoryAction, *RemoveFromMemoryAction, *SearchMemoryAction) {
	ma := &MemoryActions{indexPath: indexPath}
	if config != nil {
		ma.customName = config["custom_name"]
		ma.customDescription = config["custom_description"]
	}
	idx, err := openOrCreateBleveIndex(indexPath)
	if err != nil {
		// Allow lazy init: index will be nil and operations will return this error
		ma.index = nil
	} else {
		ma.index = idx
	}
	return &AddToMemoryAction{ma}, &ListMemoryAction{ma}, &RemoveFromMemoryAction{ma}, &SearchMemoryAction{ma}
}

func openOrCreateBleveIndex(indexPath string) (bleve.Index, error) {
	indexCacheMu.Lock()
	if idx, ok := indexCache[indexPath]; ok {
		indexCacheMu.Unlock()
		return idx, nil
	}
	indexCacheMu.Unlock()

	var idx bleve.Index
	var err error
	if _, statErr := os.Stat(indexPath); statErr == nil {
		idx, err = bleve.Open(indexPath)
	} else {
		os.MkdirAll(filepath.Dir(indexPath), 0755)
		mapping := bleve.NewIndexMapping()
		entryMapping := bleve.NewDocumentMapping()

		nameFieldMapping := bleve.NewTextFieldMapping()
		nameFieldMapping.Analyzer = "standard"
		nameFieldMapping.Store = true
		entryMapping.AddFieldMappingsAt("name", nameFieldMapping)

		contentFieldMapping := bleve.NewTextFieldMapping()
		contentFieldMapping.Analyzer = "standard"
		contentFieldMapping.Store = true
		entryMapping.AddFieldMappingsAt("content", contentFieldMapping)

		dateFieldMapping := bleve.NewDateTimeFieldMapping()
		dateFieldMapping.Store = true
		entryMapping.AddFieldMappingsAt("created_at", dateFieldMapping)

		mapping.AddDocumentMapping("_default", entryMapping)
		idx, err = bleve.New(indexPath, mapping)
	}
	if err != nil {
		return nil, err
	}

	indexCacheMu.Lock()
	indexCache[indexPath] = idx
	indexCacheMu.Unlock()
	return idx, nil
}

func (m *MemoryActions) ensureIndex() error {
	if m.index != nil {
		return nil
	}
	idx, err := openOrCreateBleveIndex(m.indexPath)
	if err != nil {
		return err
	}
	m.index = idx
	return nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

type addToMemoryParams struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type removeFromMemoryParams struct {
	ID string `json:"id"`
}

type searchMemoryParams struct {
	Query string `json:"query"`
}

func (a *AddToMemoryAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	if err := a.ensureIndex(); err != nil {
		return types.ActionResult{}, err
	}
	var req addToMemoryParams
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}
	if req.Name == "" && req.Content == "" {
		return types.ActionResult{}, fmt.Errorf("name or content cannot both be empty")
	}
	entry := MemoryEntry{
		ID:        generateID(),
		Name:      req.Name,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}
	if err := a.index.Index(entry.ID, entry); err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to index memory entry: %w", err)
	}
	return types.ActionResult{
		Result:   fmt.Sprintf("Added memory entry: id=%s name=%q", entry.ID, entry.Name),
		Metadata: map[string]any{"id": entry.ID, "name": entry.Name, "content": entry.Content, "created_at": entry.CreatedAt},
	}, nil
}

func (a *ListMemoryAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	if err := a.ensureIndex(); err != nil {
		return types.ActionResult{}, err
	}
	query := bleve.NewMatchAllQuery()
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 10000
	searchRequest.Fields = []string{"name", "created_at"}
	searchRequest.SortBy([]string{"-created_at"})

	searchResult, err := a.index.Search(searchRequest)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to search index: %w", err)
	}

	type listEntry struct {
		Name      string
		CreatedAt time.Time
	}
	entries := make([]listEntry, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		e := listEntry{}
		if v, ok := hit.Fields["name"].(string); ok {
			e.Name = v
		}
		if v, ok := hit.Fields["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				e.CreatedAt = t
			}
		} else if v, ok := hit.Fields["created_at"].(time.Time); ok {
			e.CreatedAt = v
		}
		entries = append(entries, e)
	}

	outputResult := "Number of items in memory: " + strconv.Itoa(len(entries)) + "\n"
	for i, e := range entries {
		createdStr := e.CreatedAt.Format(time.RFC3339)
		outputResult += fmt.Sprintf("%d) %s (created_at: %s)\n", i, e.Name, createdStr)
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return types.ActionResult{
		Result:   outputResult,
		Metadata: map[string]any{"names": names, "entries": entries, "count": len(entries)},
	}, nil
}

func (a *RemoveFromMemoryAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	if err := a.ensureIndex(); err != nil {
		return types.ActionResult{}, err
	}
	var req removeFromMemoryParams
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}
	if req.ID == "" {
		return types.ActionResult{}, fmt.Errorf("id is required to remove a memory entry")
	}
	doc, err := a.index.Document(req.ID)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to check document: %w", err)
	}
	if doc == nil {
		return types.ActionResult{}, fmt.Errorf("memory entry with ID %q not found", req.ID)
	}
	if err := a.index.Delete(req.ID); err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to delete memory entry: %w", err)
	}
	return types.ActionResult{
		Result:   fmt.Sprintf("Removed memory entry with ID %q", req.ID),
		Metadata: map[string]any{"removed_id": req.ID},
	}, nil
}

func (a *SearchMemoryAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	if err := a.ensureIndex(); err != nil {
		return types.ActionResult{}, err
	}
	var req searchMemoryParams
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}
	if req.Query == "" {
		return types.ActionResult{}, fmt.Errorf("query cannot be empty")
	}
	nameQuery := bleve.NewMatchQuery(req.Query)
	nameQuery.SetField("name")
	contentQuery := bleve.NewMatchQuery(req.Query)
	contentQuery.SetField("content")
	disjunctionQuery := bleve.NewDisjunctionQuery(nameQuery, contentQuery)

	searchRequest := bleve.NewSearchRequest(disjunctionQuery)
	searchRequest.Size = 100
	searchRequest.Fields = []string{"name", "content", "created_at"}

	searchResult, err := a.index.Search(searchRequest)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to search index: %w", err)
	}

	results := make([]MemoryEntry, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		e := MemoryEntry{ID: hit.ID}
		if v, ok := hit.Fields["name"].(string); ok {
			e.Name = v
		}
		if v, ok := hit.Fields["content"].(string); ok {
			e.Content = v
		}
		if v, ok := hit.Fields["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				e.CreatedAt = t
			}
		} else if v, ok := hit.Fields["created_at"].(time.Time); ok {
			e.CreatedAt = v
		}
		results = append(results, e)
	}

	outputResult := fmt.Sprintf("Query: %q — %d result(s)\n", req.Query, len(results))
	for i, e := range results {
		outputResult += fmt.Sprintf("%d) [%s] %s — %s\n", i, e.ID, e.Name, e.Content)
	}

	return types.ActionResult{
		Result:   outputResult,
		Metadata: map[string]any{"query": req.Query, "results": results, "count": len(results)},
	}, nil
}

func (a *AddToMemoryAction) Definition() types.ActionDefinition {
	name := "add_to_memory"
	description := "Add a new entry to memory storage (name and/or content). Stored in a Bleve index."
	if a.customName != "" {
		name = a.customName
	}
	if a.customDescription != "" {
		description = a.customDescription
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(name),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type:        jsonschema.String,
				Description: "The name/title of the memory entry.",
			},
			"content": {
				Type:        jsonschema.String,
				Description: "The content to store in memory.",
			},
		},
		Required: []string{},
	}
}

func (a *ListMemoryAction) Definition() types.ActionDefinition {
	name := "list_memory"
	description := "List all memory entry names."
	if a.customName != "" {
		name = a.customName
	}
	if a.customDescription != "" {
		description = a.customDescription
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(name),
		Description: description,
		Properties:  map[string]jsonschema.Definition{},
		Required:    []string{},
	}
}

func (a *RemoveFromMemoryAction) Definition() types.ActionDefinition {
	name := "remove_from_memory"
	description := "Remove a memory entry by ID."
	if a.customName != "" {
		name = a.customName
	}
	if a.customDescription != "" {
		description = a.customDescription
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(name),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"id": {
				Type:        jsonschema.String,
				Description: "The ID of the memory entry to remove.",
			},
		},
		Required: []string{"id"},
	}
}

func (a *SearchMemoryAction) Definition() types.ActionDefinition {
	name := "search_memory"
	description := "Search memory entries by name and content using full-text search."
	if a.customName != "" {
		name = a.customName
	}
	if a.customDescription != "" {
		description = a.customDescription
	}
	return types.ActionDefinition{
		Name:        types.ActionDefinitionName(name),
		Description: description,
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "The search query to find matching memory entries.",
			},
		},
		Required: []string{"query"},
	}
}

func (a *AddToMemoryAction) Plannable() bool       { return true }
func (a *ListMemoryAction) Plannable() bool        { return true }
func (a *RemoveFromMemoryAction) Plannable() bool  { return true }
func (a *SearchMemoryAction) Plannable() bool      { return true }

// AddToMemoryConfigMeta returns the metadata for AddToMemory action configuration fields
func AddToMemoryConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "custom_name",
			Label:    "Custom Name",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom name for the action (optional, defaults to 'add_to_memory')",
		},
		{
			Name:     "custom_description",
			Label:    "Custom Description",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom description for the action (optional)",
		},
	}
}

// ListMemoryConfigMeta returns the metadata for ListMemory action configuration fields
func ListMemoryConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "custom_name",
			Label:    "Custom Name",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom name for the action (optional, defaults to 'list_memory')",
		},
		{
			Name:     "custom_description",
			Label:    "Custom Description",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom description for the action (optional)",
		},
	}
}

// RemoveFromMemoryConfigMeta returns the metadata for RemoveFromMemory action configuration fields
func RemoveFromMemoryConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "custom_name",
			Label:    "Custom Name",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom name for the action (optional, defaults to 'remove_from_memory')",
		},
		{
			Name:     "custom_description",
			Label:    "Custom Description",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom description for the action (optional)",
		},
	}
}

// SearchMemoryConfigMeta returns the metadata for SearchMemory action configuration fields
func SearchMemoryConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "custom_name",
			Label:    "Custom Name",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom name for the action (optional, defaults to 'search_memory')",
		},
		{
			Name:     "custom_description",
			Label:    "Custom Description",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom description for the action (optional)",
		},
	}
}
