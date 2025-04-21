package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client represents a client for interacting with the LocalOperator API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// AgentRequest represents the request body for running an agent
type AgentRequest struct {
	Goal                string `json:"goal"`
	MaxAttempts         int    `json:"max_attempts,omitempty"`
	MaxNoActionAttempts int    `json:"max_no_action_attempts,omitempty"`
}

// StateDescription represents a single state in the agent's history
type StateDescription struct {
	CurrentURL             string `json:"current_url"`
	PageTitle              string `json:"page_title"`
	PageContentDescription string `json:"page_content_description"`
	Screenshot             string `json:"screenshot"`
	ScreenshotMimeType     string `json:"screenshot_mime_type"` // MIME type of the screenshot (e.g., "image/png")
}

// StateHistory represents the complete history of states during agent execution
type StateHistory struct {
	States []StateDescription `json:"states"`
}

// RunAgent sends a request to run an agent with the given goal
func (c *Client) RunBrowserAgent(req AgentRequest) (*StateHistory, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/browser/run", c.baseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var state StateHistory
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &state, nil
}
