package localagent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a client for the LocalAgent API
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new LocalAgent client
func NewClient(baseURL string, apiKey string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = time.Second * 30
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.HTTPClient.Timeout = timeout
}

// doRequest performs an HTTP request and returns the response
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Read the error response
		defer resp.Body.Close()
		errorData, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(errorData))
	}

	return resp, nil
}
