package localoperator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout ...time.Duration) *Client {
	defaultTimeout := 30 * time.Second
	if len(timeout) > 0 {
		defaultTimeout = timeout[0]
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

type AgentRequest struct {
	Goal                string `json:"goal"`
	MaxAttempts         int    `json:"max_attempts,omitempty"`
	MaxNoActionAttempts int    `json:"max_no_action_attempts,omitempty"`
}

type DesktopAgentRequest struct {
	AgentRequest
	DesktopURL string `json:"desktop_url"`
}

type DeepResearchRequest struct {
	Topic               string `json:"topic"`
	MaxCycles           int    `json:"max_cycles,omitempty"`
	MaxNoActionAttempts int    `json:"max_no_action_attempts,omitempty"`
	MaxResults          int    `json:"max_results,omitempty"`
}

// Response types
type StateDescription struct {
	CurrentURL             string `json:"current_url"`
	PageTitle              string `json:"page_title"`
	PageContentDescription string `json:"page_content_description"`
	Screenshot             string `json:"screenshot"`
	ScreenshotMimeType     string `json:"screenshot_mime_type"`
}

type StateHistory struct {
	States []StateDescription `json:"states"`
}

type DesktopStateDescription struct {
	ScreenContent  string `json:"screen_content"`
	ScreenshotPath string `json:"screenshot_path"`
}

type DesktopStateHistory struct {
	States []DesktopStateDescription `json:"states"`
}

type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type ResearchResult struct {
	Topic          string         `json:"topic"`
	Summary        string         `json:"summary"`
	Sources        []SearchResult `json:"sources"`
	KnowledgeGaps  []string       `json:"knowledge_gaps"`
	SearchQueries  []string       `json:"search_queries"`
	ResearchCycles int            `json:"research_cycles"`
	CompletionTime time.Duration  `json:"completion_time"`
}

func (c *Client) RunBrowserAgent(req AgentRequest) (*StateHistory, error) {
	return post[*StateHistory](c.httpClient, c.baseURL+"/api/browser/run", req)
}

func (c *Client) RunDesktopAgent(req DesktopAgentRequest) (*DesktopStateHistory, error) {
	return post[*DesktopStateHistory](c.httpClient, c.baseURL+"/api/desktop/run", req)
}

func (c *Client) RunDeepResearch(req DeepResearchRequest) (*ResearchResult, error) {
	return post[*ResearchResult](c.httpClient, c.baseURL+"/api/deep-research/run", req)
}

func (c *Client) Readyz() (string, error) {
	return c.get("/readyz")
}

func (c *Client) Healthz() (string, error) {
	return c.get("/healthz")
}

func (c *Client) get(path string) (string, error) {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return resp.Status, nil
}

func post[T any](client *http.Client, url string, body interface{}) (T, error) {
	var result T
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return result, fmt.Errorf("failed to marshal request body: %w", err)
	}

	fmt.Println("Sending request", "url", url, "body", string(jsonBody))

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return result, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Println("Response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
