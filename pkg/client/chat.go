package localagi

import (
	"fmt"
	"net/http"
	"strings"
)

// Message represents a chat message
type Message struct {
	Message string `json:"message"`
}

// ChatResponse represents a response from the agent
type ChatResponse struct {
	Response string `json:"response"`
}

// SendMessage sends a message to an agent
func (c *Client) SendMessage(agentName, message string) error {
	path := fmt.Sprintf("/chat/%s", agentName)

	msg := Message{
		Message: message,
	}

	resp, err := c.doRequest(http.MethodPost, path, msg)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// The response is HTML, so it's not easily parseable in this context
	return nil
}

// Notify sends a notification to an agent
func (c *Client) Notify(agentName, message string) error {
	path := fmt.Sprintf("/notify/%s", agentName)

	// URL encoded form data
	form := strings.NewReader(fmt.Sprintf("message=%s", message))

	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, form)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("api error (status %d)", resp.StatusCode)
	}

	return nil
}
