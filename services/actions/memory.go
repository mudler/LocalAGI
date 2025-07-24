package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"io"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// Remove global const and mutex, and add them as fields to a struct

type MemoryActions struct {
	filePath          string
	customName        string
	customDescription string
}

type AddToMemoryAction struct{ *MemoryActions }
type ListMemoryAction struct{ *MemoryActions }
type RemoveFromMemoryAction struct{ *MemoryActions }

// NewMemoryActions returns the three actions, using the provided filePath and config
func NewMemoryActions(filePath string, config map[string]string) (*AddToMemoryAction, *ListMemoryAction, *RemoveFromMemoryAction) {
	ma := &MemoryActions{filePath: filePath}
	if config != nil {
		ma.customName = config["custom_name"]
		ma.customDescription = config["custom_description"]
	}
	return &AddToMemoryAction{ma}, &ListMemoryAction{ma}, &RemoveFromMemoryAction{ma}
}

type addToMemoryParams struct {
	Item string `json:"item"`
}

type removeFromMemoryParams struct {
	Index *int   `json:"index,omitempty"`
	Value string `json:"value,omitempty"`
}

func (m *MemoryActions) readMemory() ([]string, error) {
	f, err := os.Open(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()
	var items []string
	if err := json.NewDecoder(f).Decode(&items); err != nil {
		if err == io.EOF {
			return []string{}, nil
		}
		return nil, err
	}
	return items, nil
}

func (m *MemoryActions) writeMemory(items []string) error {
	f, err := os.Create(m.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(items)
}

func (a *AddToMemoryAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var req addToMemoryParams
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}
	if req.Item == "" {
		return types.ActionResult{}, fmt.Errorf("item cannot be empty")
	}
	items, err := a.readMemory()
	if err != nil {
		return types.ActionResult{}, err
	}
	items = append(items, req.Item)
	if err := a.writeMemory(items); err != nil {
		return types.ActionResult{}, err
	}
	return types.ActionResult{
		Result:   fmt.Sprintf("Added item to memory: %s", req.Item),
		Metadata: map[string]any{"item": req.Item, "count": len(items)},
	}, nil
}

func (a *ListMemoryAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	items, err := a.readMemory()
	if err != nil {
		return types.ActionResult{}, err
	}

	outputResult := "Number of items in memory: " + strconv.Itoa(len(items)) + "\n"
	for i, item := range items {
		outputResult += fmt.Sprintf("%d) %s\n", i, item)
	}

	return types.ActionResult{
		Result:   outputResult,
		Metadata: map[string]any{"items": items},
	}, nil
}

func (a *RemoveFromMemoryAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var req removeFromMemoryParams
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}
	items, err := a.readMemory()
	if err != nil {
		return types.ActionResult{}, err
	}
	var removed string
	if req.Index != nil {
		idx := *req.Index
		if idx < 0 || idx >= len(items) {
			return types.ActionResult{}, fmt.Errorf("index out of range")
		}
		removed = items[idx]
		items = append(items[:idx], items[idx+1:]...)
	} else if req.Value != "" {
		found := false
		for i, v := range items {
			if v == req.Value {
				removed = v
				items = append(items[:i], items[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return types.ActionResult{}, fmt.Errorf("value not found in memory")
		}
	} else {
		return types.ActionResult{}, fmt.Errorf("must provide index or value to remove")
	}
	if err := a.writeMemory(items); err != nil {
		return types.ActionResult{}, err
	}
	return types.ActionResult{
		Result:   fmt.Sprintf("Removed item from memory: %s", removed),
		Metadata: map[string]any{"removed": removed, "count": len(items)},
	}, nil
}

func (a *AddToMemoryAction) Definition() types.ActionDefinition {
	name := "add_to_memory"
	description := "Add a string item to memory (stored in a JSON file)."
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
			"item": {
				Type:        jsonschema.String,
				Description: "The string item to add to memory.",
			},
		},
		Required: []string{"item"},
	}
}

func (a *ListMemoryAction) Definition() types.ActionDefinition {
	name := "list_memory"
	description := "List all items currently stored in memory."
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
	description := "Remove an item from memory by index or value."
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
			"index": {
				Type:        jsonschema.Integer,
				Description: "The index of the item to remove (optional, 0-based)",
			},
			"value": {
				Type:        jsonschema.String,
				Description: "The value of the item to remove (optional)",
			},
		},
		Required: []string{},
	}
}

func (a *AddToMemoryAction) Plannable() bool      { return true }
func (a *ListMemoryAction) Plannable() bool       { return true }
func (a *RemoveFromMemoryAction) Plannable() bool { return true }

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
			HelpText: "Custom description for the action (optional, defaults to 'Add a string item to memory (stored in a JSON file).')",
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
			HelpText: "Custom description for the action (optional, defaults to 'List all items currently stored in memory.')",
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
			HelpText: "Custom description for the action (optional, defaults to 'Remove an item from memory by index or value.')",
		},
	}
}
