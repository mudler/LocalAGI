package actions

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewPiKVMAction(config map[string]string) *PiKVMAction {
	return &PiKVMAction{
		hostname:          config["hostname"],
		username:          config["username"],
		password:          config["password"],
		customName:        config["custom_name"],
		customDescription: config["custom_description"],
		insecure:          config["insecure"] == "true",
	}
}

type PiKVMAction struct {
	hostname          string
	username          string
	password          string
	customName        string
	customDescription string
	insecure          bool
}

type pikvmPowerParams struct {
	Action string `json:"action"`
}

func (a *PiKVMAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	var req pikvmPowerParams
	if err := params.Unmarshal(&req); err != nil {
		return types.ActionResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate action parameter
	validActions := map[string]bool{
		"on":         true,
		"off":        true,
		"off_hard":   true,
		"reset_hard": true,
	}
	if !validActions[req.Action] {
		return types.ActionResult{}, fmt.Errorf("invalid action: %s. Valid actions are: on, off, off_hard, reset_hard", req.Action)
	}

	// Check if required config is provided
	if a.hostname == "" {
		return types.ActionResult{}, fmt.Errorf("hostname is required in action configuration")
	}
	if a.username == "" {
		return types.ActionResult{}, fmt.Errorf("username is required in action configuration")
	}
	if a.password == "" {
		return types.ActionResult{}, fmt.Errorf("password is required in action configuration")
	}

	// Build the API URL
	apiURL := fmt.Sprintf("https://%s/api/atx/power", a.hostname)

	insecure := false
	if a.insecure {
		insecure = true
	}
	// Create HTTP client with basic auth
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	client := &http.Client{Transport: tr}

	reqHTTP, err := http.NewRequestWithContext(ctx, "POST", apiURL, nil)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set basic authentication
	reqHTTP.SetBasicAuth(a.username, a.password)

	// Add query parameters
	q := reqHTTP.URL.Query()
	q.Add("action", req.Action)
	reqHTTP.URL.RawQuery = q.Encode()

	// Make the request
	resp, err := client.Do(reqHTTP)
	if err != nil {
		return types.ActionResult{}, fmt.Errorf("failed to make HTTP request to PiKVM: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return types.ActionResult{}, fmt.Errorf("PiKVM API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Determine action description for user-friendly response
	actionDesc := map[string]string{
		"on":         "power on",
		"off":        "power off",
		"off_hard":   "hard power off",
		"reset_hard": "hard reset",
	}

	result := fmt.Sprintf("Successfully sent %s command to PiKVM at %s", actionDesc[req.Action], a.hostname)

	return types.ActionResult{
		Result: result,
		Metadata: map[string]any{
			"action":   req.Action,
			"hostname": a.hostname,
			"status":   "success",
		},
	}, nil
}

func (a *PiKVMAction) Definition() types.ActionDefinition {
	name := "pikvm_power_control"
	description := "Control power state of a PiKVM device using ATX power management."

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
			"action": {
				Type:        jsonschema.String,
				Description: "The power action to perform on the PiKVM device.",
				Enum:        []string{"on", "off", "off_hard", "reset_hard"},
			},
		},
		Required: []string{"action"},
	}
}

func (a *PiKVMAction) Plannable() bool {
	return true
}

// PiKVMConfigMeta returns the metadata for PiKVM action configuration fields
func PiKVMConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "hostname",
			Label:    "PiKVM Hostname",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "The hostname or IP address of the PiKVM device (e.g., pikvm.local or 192.168.1.100)",
		},
		{
			Name:     "username",
			Label:    "Username",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Username for PiKVM authentication (usually 'admin')",
		},
		{
			Name:     "password",
			Label:    "Password",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Password for PiKVM authentication",
		},
		{
			Name:     "custom_name",
			Label:    "Custom Action Name",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom name for this action (optional, defaults to 'pikvm_power_control')",
		},
		{
			Name:     "custom_description",
			Label:    "Custom Description",
			Type:     config.FieldTypeText,
			Required: false,
			HelpText: "Custom description for this action (optional)",
		},
		{
			Name:     "insecure",
			Label:    "Insecure",
			Type:     config.FieldTypeCheckbox,
			Required: false,
			HelpText: "Skip certificate verification (optional)",
		},
	}
}
