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

// NewWebhook constructs a WebhookAction using provided configuration values.
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

type WebhookAction struct {
	url             string
	method          string
	contentType     string
	payloadTemplate string
}

func (a *WebhookAction) Run(ctx context.Context, sharedState *types.AgentSharedState, params types.ActionParams) (types.ActionResult, error) {
	// Runtime parameters: only payload
	type input struct {
		Payload string `json:"payload"`
	}
	var in input
	if err := params.Unmarshal(&in); err != nil {
		return types.ActionResult{}, err
	}

	if a.url == "" {
		return types.ActionResult{}, fmt.Errorf("configuration.url is required")
	}

	method := a.method

	// Build the request body based on template and payload
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

	req, err := http.NewRequestWithContext(ctx, method, a.url, body)
	if err != nil {
		return types.ActionResult{}, err
	}

	if a.contentType != "" {
		req.Header.Set("Content-Type", a.contentType)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return types.ActionResult{}, err
	}
	defer resp.Body.Close()

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

func (a *WebhookAction) Plannable() bool { return true }

// WebhookConfigMeta returns the metadata for Webhook action configuration fields
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
