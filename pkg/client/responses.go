package localagi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sashabaranov/go-openai/jsonschema"
)

// UserLocation represents the user's location for web search
type UserLocation struct {
	Type     string  `json:"type"`
	City     *string `json:"city,omitempty"`
	Country  *string `json:"country,omitempty"`
	Region   *string `json:"region,omitempty"`
	Timezone *string `json:"timezone,omitempty"`
}

type Tool struct {
	Type string `json:"type"`

	// Function tool fields (used when type == "function")
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Parameters  *jsonschema.Definition `json:"parameters,omitempty"`

	// Web search tool fields (used when type == "web_search_preview" etc.)
	SearchContextSize *string       `json:"search_context_size,omitempty"`
	UserLocation      *UserLocation `json:"user_location,omitempty"`
}

type ToolChoice struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// RequestBody represents the message request to the AI model
type RequestBody struct {
	Model       string   `json:"model"`
	Input       any      `json:"input"`
	Temperature *float64 `json:"temperature,omitempty"`
	Tools       []Tool   `json:"tools,omitempty"`
	ToolChoice *ToolChoice `json:"tool_choice"`  
	MaxTokens   *int     `json:"max_output_tokens,omitempty"`
}

type InputFunctionToolCallOutput struct {
	CallID string `json:"call_id"`
	Output string `json:"output"`
	Type   string `json:"type"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

// InputMessage represents a user input message
type InputMessage struct {
	Type    string `json:"type"`
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
	CreatedAt int64          `json:"created_at"`
	Status    string         `json:"status"`
	Error     any            `json:"error,omitempty"`
	Output    []ResponseBase `json:"output"`
	Tools     []Tool         `json:"tools"`
}

type ResponseType string

const (
	ResponseTypeFunctionToolCall ResponseType = "function_call"
	ResponseTypeMessage          ResponseType = "message"
)

type ResponseBase json.RawMessage

func (r *ResponseBase) UnmarshalJSON(data []byte) error {
	return (*json.RawMessage)(r).UnmarshalJSON(data)
}

func (r *ResponseBase) ToMessage() (msg ResponseMessage, err error) {
	err = json.Unmarshal(*r, &msg)
	if msg.Type != string(ResponseTypeMessage) {
		return ResponseMessage{}, fmt.Errorf("Expected %s, not %s", ResponseTypeMessage, msg.Type)
	}
	return
}

func (r *ResponseBase) ToFunctionToolCall() (msg ResponseFunctionToolCall, err error) {
	err = json.Unmarshal(*r, &msg)
	if msg.Type != string(ResponseTypeFunctionToolCall) {
		return ResponseFunctionToolCall{}, fmt.Errorf("Expected %s, not %s", ResponseTypeFunctionToolCall, msg.Type)
	}
	return
}

type ResponseFunctionToolCall struct {
	Arguments string `json:"arguments"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	Status    string `json:"status"`
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
	for _, out := range response.Output {
		msg, err := out.ToMessage()
		if err != nil {
			return "", fmt.Errorf("out.ToMessage: %w", err)
		}
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
	for _, out := range response.Output {
		msg, err := out.ToMessage()
		if err != nil {
			return "", fmt.Errorf("out.ToMessage: %w", err)
		}

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
