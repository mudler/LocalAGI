// Package actions contains action implementations used by LocalAGI.
// This file implements the "webhook" action which can send an HTTP request
// to an external service with a configurable method, content type, and payload.
package actions

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// NewWebhook constructs a WebhookAction using provided configuration values:
//   - url: Destination endpoint for the HTTP request (required).
//   - method: HTTP method to use (GET, POST, PUT, DELETE, ...). Defaults to POST.
//   - contentType: Value for the Content-Type header (e.g., application/json).
//   - payloadTemplate: Optional template for the request body; the runtime parameter
//     "payload" (if provided) will replace the "{{payload}}" placeholder inside this template.
func NewWebhook(cfg map[string]string) *WebhookAction {
	wa := &WebhookAction{
		url:             strings.TrimSpace(cfg["url"]),
		method:          strings.ToUpper(strings.TrimSpace(cfg["method"])),
		contentType:     strings.TrimSpace(cfg["contentType"]),
		payloadTemplate: cfg["payloadTemplate"],
	}
	if wa.method == "" {
		wa.method = http.MethodPost
	}
	return wa
}

// WebhookAction holds the static configuration for the webhook.
// These values come from the action configuration (UI/agent config),
// while the runtime parameter only carries the dynamic payload.
//   - url: Target endpoint for the request.
//   - method: HTTP method to use. Defaults to POST if not provided.
//   - contentType: Sets the Content-Type header when a body is sent.
//   - payloadTemplate: Optional template used to build the request body; occurrences
//     of "{{payload}}" get replaced with the runtime payload string.
//     If no placeholder is present, the template is used as-is.
//     For GET requests the body is omitted regardless of payload.
//
// Note: This action does not follow redirects!
type WebhookAction struct {
	url             string
	method          string
	contentType     string
	payloadTemplate string
}

// Run executes the webhook call.
// It reads the runtime parameter "payload" (optional), merges it into the
// configured payloadTemplate (if any), constructs an HTTP request using the
// configured URL, method and content type, and then returns a summary with the
// response status and body (truncated to 4KiB for safety).
func (a *WebhookAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	// Runtime parameters: only payload
	type input struct {
		Payload string `json:"payload"`
	}
	var in input
	if err := params.Unmarshal(&in); err != nil {
		return types.ActionResult{}, err
	}

	// Validate essential configuration. The URL must be provided via the
	// action configuration (not via runtime parameters).
	if a.url == "" {
		return types.ActionResult{}, fmt.Errorf("configuration.url is required")
	}

	method := a.method

	// Build the request body based on template and payload:
	// - If a payloadTemplate is provided, replace occurrences of "{{payload}}"
	//   with the runtime payload value.
	// - If the template does not contain the placeholder but is provided, we use
	//   the template as-is (common for static JSON bodies prepared at config time).
	// - If no template is configured, we send the runtime payload as-is.
	// - For GET requests the body is omitted regardless of payload.
	var payload string
	if a.payloadTemplate != "" {
		payload = strings.ReplaceAll(a.payloadTemplate, "{{payload}}", in.Payload)
		if payload == a.payloadTemplate && in.Payload != "" {
			// If no placeholder found, fallback to template or payload alone
			payload = a.payloadTemplate
		}
	} else {
		payload = in.Payload
	}

	var body io.Reader
	if method != http.MethodGet && payload != "" {
		body = bytes.NewBufferString(payload)
	}

	// Create the HTTP request bound to the provided context so that cancellation
	// or timeouts from the caller propagate to the outbound call.
	req, err := http.NewRequestWithContext(ctx, method, a.url, body)
	if err != nil {
		return types.ActionResult{}, err
	}

	// Set Content-Type header if configured. For GET requests this header is
	// typically ignored by servers as there is no body.
	if a.contentType != "" {
		req.Header.Set("Content-Type", a.contentType)
	}

	// Use a new http.Client with default settings. Consider configuring timeouts
	// at the caller level via the context, or wiring a custom client if needed.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return types.ActionResult{}, err
	}
	defer resp.Body.Close()

	// Read and safely truncate the response body to avoid flooding the agent's
	// context with very large payloads. Errors on ReadAll are ignored here as
	// we already have the status code.
	respBytes, _ := io.ReadAll(resp.Body)
	respBody := string(respBytes)
	if len(respBody) > 4096 {
		respBody = respBody[:4096] + "... (truncated)"
	}

	return types.ActionResult{
		// Return the response body as the result.
		// If the response body is empty, use the status text as the result (e.g. "OK" for status code 200).
		Result: func() string {
			if respBody == "" {
				return http.StatusText(resp.StatusCode)
			}
			return respBody
		}(),
		// Include the response status code in the metadata.
		Metadata: map[string]interface{}{
			"statusCode": resp.StatusCode,
		},
	}, nil
}

// Definition returns the action schema exposed to the planner/runtime.
// Only the runtime parameter "payload" is accepted; all connection details
// are configured statically via the action configuration UI.
func (a *WebhookAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "webhook",
		Description: "Send an HTTP request to a configured URL/method/content-type. Accepts a runtime payload parameter optionally inserted into the configured payload template.",
		Properties: map[string]jsonschema.Definition{
			"payload": {
				Type:        jsonschema.String,
				Description: "Payload/body to send with the request at runtime. If a payloadTemplate is configured, '{{payload}}' will be replaced by this value.",
			},
		},
	}
}

// Plannable indicates the action can be suggested/used by planners without
// requiring hidden context; inputs are straightforward and safe.
func (a *WebhookAction) Plannable() bool { return true }

// WebhookConfigMeta returns the metadata for Webhook action configuration fields:
//   - url: The endpoint to send requests to (required).
//   - method: One of GET/POST/PUT/DELETE. Defaults to POST.
//   - contentType: Common content types selectable from a dropdown.
//   - payloadTemplate: Optional body template. At runtime, "{{payload}}" is
//     replaced by the provided payload parameter. If missing, the template is used
//     as-is; for GET, no body is sent regardless.
func WebhookConfigMeta() []config.Field {
	return []config.Field{
		{
			Name:     "url",
			Label:    "URL",
			Type:     config.FieldTypeText,
			Required: true,
			HelpText: "Destination URL for the webhook",
		},
		{
			Name:         "method",
			Label:        "HTTP Method",
			Type:         config.FieldTypeSelect,
			Options:      []config.FieldOption{{Value: http.MethodGet, Label: "GET"}, {Value: http.MethodPost, Label: "POST"}, {Value: http.MethodPut, Label: "PUT"}, {Value: http.MethodDelete, Label: "DELETE"}},
			DefaultValue: http.MethodPost,
			Required:     true,
			HelpText:     "HTTP method to use",
		},
		{
			Name:  "contentType",
			Label: "Content Type",
			Type:  config.FieldTypeSelect,
			Options: []config.FieldOption{
				{Value: "application/json", Label: "application/json"},
				{Value: "text/plain", Label: "text/plain"},
				{Value: "application/x-www-form-urlencoded", Label: "application/x-www-form-urlencoded"},
			},
			Required: true,
			HelpText: "Content-Type header to send",
		},
		{
			Name:     "payloadTemplate",
			Label:    "Payload Template",
			Type:     config.FieldTypeTextarea,
			HelpText: "Optional template used to craft the request body. Use '{{payload}}' as placeholder for the runtime payload.",
		},
	}
}
