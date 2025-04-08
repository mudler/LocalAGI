package actions

import (
	"context"
	"fmt"
	"sync"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// CounterAction manages named counters that can be created, updated, and queried
type CounterAction struct {
	counters map[string]int
	mutex    sync.RWMutex
}

// NewCounter creates a new counter action
func NewCounter(config map[string]string) *CounterAction {
	return &CounterAction{
		counters: make(map[string]int),
		mutex:    sync.RWMutex{},
	}
}

// Run executes the counter action
func (a *CounterAction) Run(ctx context.Context, params types.ActionParams) (types.ActionResult, error) {
	// Parse parameters
	request := struct {
		Name       string `json:"name"`
		Adjustment int    `json:"adjustment"`
	}{}

	if err := params.Unmarshal(&request); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if request.Name == "" {
		return types.ActionResult{}, fmt.Errorf("counter name cannot be empty")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Get current value or initialize if it doesn't exist
	currentValue, exists := a.counters[request.Name]

	// Update the counter
	newValue := currentValue + request.Adjustment
	a.counters[request.Name] = newValue

	// Prepare the response message
	var message string
	if !exists && request.Adjustment == 0 {
		message = fmt.Sprintf("Created counter '%s' with initial value 0", request.Name)
	} else if !exists {
		message = fmt.Sprintf("Created counter '%s' with initial value %d", request.Name, newValue)
	} else if request.Adjustment > 0 {
		message = fmt.Sprintf("Increased counter '%s' by %d to %d", request.Name, request.Adjustment, newValue)
	} else if request.Adjustment < 0 {
		message = fmt.Sprintf("Decreased counter '%s' by %d to %d", request.Name, -request.Adjustment, newValue)
	} else {
		message = fmt.Sprintf("Current value of counter '%s' is %d", request.Name, newValue)
	}

	return types.ActionResult{
		Result: message,
		Metadata: map[string]any{
			"counter_name":  request.Name,
			"counter_value": newValue,
			"adjustment":    request.Adjustment,
			"is_new":        !exists,
		},
	}, nil
}

// Definition returns the action definition
func (a *CounterAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "counter",
		Description: "Create, update, or query named counters. Specify a name and an adjustment value (positive to increase, negative to decrease, zero to query).",
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type:        jsonschema.String,
				Description: "The name of the counter to create, update, or query.",
			},
			"adjustment": {
				Type:        jsonschema.Integer,
				Description: "The value to adjust the counter by. Positive to increase, negative to decrease, zero to query the current value.",
			},
		},
		Required: []string{"name", "adjustment"},
	}
}

func (a *CounterAction) Plannable() bool {
	return true
}
