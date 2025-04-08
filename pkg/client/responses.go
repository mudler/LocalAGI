package localagi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// RequestBody represents the message request to the AI model
type RequestBody struct {
	Model       string   `json:"model"`
	Input       any      `json:"input"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_output_tokens,omitempty"`
}

// InputMessage represents a user input message
type InputMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ContentItem represents an item in a content array
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// ResponseBody represents the response from the AI model
type ResponseBody struct {
	CreatedAt int64             `json:"created_at"`
	Status    string            `json:"status"`
	Error     any               `json:"error,omitempty"`
	Output    []ResponseMessage `json:"output"`
}

// ResponseMessage represents a message in the response
type ResponseMessage struct {
	Type    string               `json:"type"`
	Status  string               `json:"status"`
	Role    string               `json:"role"`
	Content []MessageContentItem `json:"content"`
}

// MessageContentItem represents a content item in a message
type MessageContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// GetAIResponse sends a request to the AI model and returns the response
func (c *Client) GetAIResponse(request *RequestBody) (*ResponseBody, error) {
	resp, err := c.doRequest(http.MethodPost, "/v1/responses", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response ResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Check if there was an error in the response
	if response.Error != nil {
		return nil, fmt.Errorf("api error: %v", response.Error)
	}

	return &response, nil
}

// SimpleAIResponse is a helper function to get a simple text response from the AI
func (c *Client) SimpleAIResponse(agentName, input string) (string, error) {
	temperature := 0.7
	request := &RequestBody{
		Model:       agentName,
		Input:       input,
		Temperature: &temperature,
	}

	response, err := c.GetAIResponse(request)
	if err != nil {
		return "", err
	}

	// Extract the text response from the output
	for _, msg := range response.Output {
		if msg.Role == "assistant" {
			for _, content := range msg.Content {
				if content.Type == "output_text" {
					return content.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no text response found")
}

// ChatAIResponse sends chat messages to the AI model
func (c *Client) ChatAIResponse(agentName string, messages []InputMessage) (string, error) {
	temperature := 0.7
	request := &RequestBody{
		Model:       agentName,
		Input:       messages,
		Temperature: &temperature,
	}

	response, err := c.GetAIResponse(request)
	if err != nil {
		return "", err
	}

	// Extract the text response from the output
	for _, msg := range response.Output {
		if msg.Role == "assistant" {
			for _, content := range msg.Content {
				if content.Type == "output_text" {
					return content.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no text response found")
}
