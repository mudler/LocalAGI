package types

import (
	"encoding/json"

	coreTypes "github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// Input represents either a string or a slice of Message
type Input struct {
	Text     *string    `json:"-"`
	Messages *[]Message `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Input
func (i *Input) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		i.Text = &text
		return nil
	}

	// Try to unmarshal as []Message
	var messages []Message
	if err := json.Unmarshal(data, &messages); err == nil {
		i.Messages = &messages
		return nil
	}

	return json.Unmarshal(data, &struct{}{}) // fallback to empty struct
}

// MarshalJSON implements custom JSON marshaling for Input
func (i *Input) MarshalJSON() ([]byte, error) {
	if i.Text != nil {
		return json.Marshal(*i.Text)
	}
	if i.Messages != nil {
		return json.Marshal(*i.Messages)
	}
	return json.Marshal(nil)
}

// IsText returns true if the input contains text
func (i *Input) IsText() bool {
	return i.Text != nil
}

// IsMessages returns true if the input contains messages
func (i *Input) IsMessages() bool {
	return i.Messages != nil
}

// GetText returns the text value or empty string
func (i *Input) GetText() string {
	if i.Text != nil {
		return *i.Text
	}
	return ""
}

// GetMessages returns the messages value or empty slice
func (i *Input) GetMessages() []Message {
	if i.Messages != nil {
		return *i.Messages
	}
	return nil
}

// Message represents different types of messages in the input
type Message struct {
	// Common fields
	Type string `json:"type,omitempty"`

	// InputMessage fields (when this is a regular chat message)
	Role    *string  `json:"role,omitempty"`
	Content *Content `json:"content,omitempty"`

	// WebSearchToolCall fields (when type == "web_search_call")
	ID     *string `json:"id,omitempty"`
	Status *string `json:"status,omitempty"`

	// Function call and function call output
	Arguments *string `json:"arguments,omitempty"`
	CallId    *string `json:"call_id,omitempty"`
	Name      *string `json:"name,omitempty"`
	Output    *string `json:"output,omitempty"`
}

// IsInputMessage returns true if this is a regular chat message
func (m *Message) IsInputMessage() bool {
	return m.Role != nil
}

// IsWebSearchCall returns true if this is a web search tool call
func (m *Message) IsWebSearchCall() bool {
	return m.Type == "web_search_call"
}

func (m *Message) IsFunctionCall() bool {
	return m.Type == "function_call"
}

func (m *Message) IsFunctionCallOutput() bool {
	return m.Type == "function_call_output"
}

// ToInputMessage converts to InputMessage if this is a regular message
func (m *Message) ToInputMessage() *InputMessage {
	if m.IsInputMessage() && m.Role != nil && m.Content != nil {
		content := *m.Content
		return &InputMessage{
			Role:    *m.Role,
			Content: content,
		}
	}
	return nil
}

type ToolChoice struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// RequestBody represents the request body structure for the OpenAI API
type RequestBody struct {
	Model              string            `json:"model"`
	Input              Input             `json:"input"`
	Include            []string          `json:"include,omitempty"`
	Instructions       *string           `json:"instructions,omitempty"`
	MaxOutputTokens    *int              `json:"max_output_tokens,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	ParallelToolCalls  *bool             `json:"parallel_tool_calls,omitempty"`
	PreviousResponseID *string           `json:"previous_response_id,omitempty"`
	Reasoning          *ReasoningConfig  `json:"reasoning,omitempty"`
	Store              *bool             `json:"store,omitempty"`
	Stream             *bool             `json:"stream,omitempty"`
	Temperature        *float64          `json:"temperature,omitempty"`
	Text               *TextConfig       `json:"text,omitempty"`
	ToolChoice         json.RawMessage   `json:"tool_choice,omitempty"`
	Tools              []Tool            `json:"tools,omitempty"`
	TopP               *float64          `json:"top_p,omitempty"`
	Truncation         *string           `json:"truncation,omitempty"`
}

func (r *RequestBody) SetInputByType() {
	// This method is no longer needed as Input handles unmarshaling automatically
	if r.Input.IsText() {
		xlog.Debug("[Parse Request] Set input type as text", "input", r.Input.GetText())
	} else if r.Input.IsMessages() {
		xlog.Debug("[Parse Request] Input messages parsed", "messages", r.Input.GetMessages())
	}
}

func (r *RequestBody) ToChatCompletionMessages() []openai.ChatCompletionMessage {
	result := []openai.ChatCompletionMessage{}

	if r.Input.IsMessages() {
		for _, m := range r.Input.GetMessages() {

			if m.IsFunctionCall() {
				result = append(result, openai.ChatCompletionMessage{
					Role: "assistant",
					ToolCalls: []openai.ToolCall{
						{
							Type: "function",
							ID:   *m.CallId,
							Function: openai.FunctionCall{
								Arguments: *m.Arguments,
								Name:      *m.Name,
							},
						},
					},
				})
			}

			if m.IsFunctionCallOutput() {
				result = append(result, openai.ChatCompletionMessage{
					Role:       "tool",
					Content:    *m.Output,
					ToolCallID: *m.CallId,
				})
			}

			if !m.IsInputMessage() {
				continue
			}

			content := []openai.ChatMessagePart{}
			oneImageWasFound := false

			if m.Content != nil && m.Content.IsText() && m.Content.GetText() != "" {
				content = append(content, openai.ChatMessagePart{
					Type: "text",
					Text: m.Content.GetText(),
				})
			}

			if m.Content != nil && m.Content.IsItems() {
				for _, c := range m.Content.GetItems() {
					switch c.Type {
					case "text":
						content = append(content, openai.ChatMessagePart{
							Type: "text",
							Text: c.Text,
						})
					case "image":
						oneImageWasFound = true
						content = append(content, openai.ChatMessagePart{
							Type:     "image",
							ImageURL: &openai.ChatMessageImageURL{URL: c.ImageURL},
						})
					}
				}
			}

			if oneImageWasFound {
				result = append(result, openai.ChatCompletionMessage{
					Role:         *m.Role,
					MultiContent: content,
				})
			} else {
				for _, c := range content {
					result = append(result, openai.ChatCompletionMessage{
						Role:    *m.Role,
						Content: c.Text,
					})
				}
			}
		}
	}

	if r.Input.IsText() && r.Input.GetText() != "" {
		result = append(result, openai.ChatCompletionMessage{
			Role:    "user",
			Content: r.Input.GetText(),
		})
	}

	return result
}

// ReasoningConfig represents reasoning configuration options
type ReasoningConfig struct {
	Effort  *string `json:"effort,omitempty"`
	Summary *string `json:"summary,omitempty"`
}

// TextConfig represents text configuration options
type TextConfig struct {
	Format *FormatConfig `json:"format,omitempty"`
}

// FormatConfig represents format configuration options
type FormatConfig struct {
	Type string `json:"type"`
}

// ResponseMessage represents a message in the response
type ResponseMessage struct {
	Type    string               `json:"type"`
	ID      string               `json:"id"`
	Status  string               `json:"status"`
	Role    string               `json:"role"`
	Content []MessageContentItem `json:"content"`
}

// MessageContentItem represents a content item in a message
type MessageContentItem struct {
	Type        string        `json:"type"`
	Text        string        `json:"text"`
	Annotations []interface{} `json:"annotations"`
}

// FunctionToolCall represents a function tool call as a top-level object in the output array
type FunctionToolCall struct {
	Arguments string `json:"arguments"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	Status    string `json:"status"`
}

// UsageInfo represents token usage information
type UsageInfo struct {
	InputTokens         int          `json:"input_tokens"`
	InputTokensDetails  TokenDetails `json:"input_tokens_details"`
	OutputTokens        int          `json:"output_tokens"`
	OutputTokensDetails TokenDetails `json:"output_tokens_details"`
	TotalTokens         int          `json:"total_tokens"`
}

// TokenDetails represents details about token usage
type TokenDetails struct {
	CachedTokens    int `json:"cached_tokens"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// ResponseBody represents the structure of the OpenAI API response
type ResponseBody struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	CreatedAt          int64                  `json:"created_at"`
	Status             string                 `json:"status"`
	Error              interface{}            `json:"error"`
	IncompleteDetails  interface{}            `json:"incomplete_details"`
	Instructions       interface{}            `json:"instructions"`
	MaxOutputTokens    interface{}            `json:"max_output_tokens"`
	Model              string                 `json:"model"`
	Output             []interface{}          `json:"output"`
	ParallelToolCalls  bool                   `json:"parallel_tool_calls"`
	PreviousResponseID interface{}            `json:"previous_response_id"`
	Reasoning          ReasoningConfig        `json:"reasoning"`
	Store              bool                   `json:"store"`
	Temperature        float64                `json:"temperature"`
	Text               TextConfig             `json:"text"`
	ToolChoice         string                 `json:"tool_choice"`
	Tools              []Tool                 `json:"tools"`
	TopP               float64                `json:"top_p"`
	Truncation         string                 `json:"truncation"`
	Usage              UsageInfo              `json:"usage"`
	User               interface{}            `json:"user"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// Content represents either a string or a slice of ContentItem
type Content struct {
	Text  *string        `json:"-"`
	Items *[]ContentItem `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Content
func (c *Content) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		c.Text = &text
		return nil
	}

	// Try to unmarshal as []ContentItem
	var items []ContentItem
	if err := json.Unmarshal(data, &items); err == nil {
		c.Items = &items
		return nil
	}

	return json.Unmarshal(data, &struct{}{}) // fallback to empty struct
}

// MarshalJSON implements custom JSON marshaling for Content
func (c *Content) MarshalJSON() ([]byte, error) {
	if c.Text != nil {
		return json.Marshal(*c.Text)
	}
	if c.Items != nil {
		return json.Marshal(*c.Items)
	}
	return json.Marshal(nil)
}

// IsText returns true if the content contains text
func (c *Content) IsText() bool {
	return c.Text != nil
}

// IsItems returns true if the content contains items
func (c *Content) IsItems() bool {
	return c.Items != nil
}

// GetText returns the text value or empty string
func (c *Content) GetText() string {
	if c.Text != nil {
		return *c.Text
	}
	return ""
}

// GetItems returns the items value or empty slice
func (c *Content) GetItems() []ContentItem {
	if c.Items != nil {
		return *c.Items
	}
	return nil
}

// InputMessage represents a user input message
type InputMessage struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

// ContentItem represents an item in a content array
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// Tool represents a tool that can be called by the assistant
type Tool struct {
	Type string `json:"type"`

	// Function tool fields (used when type == "function")
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Parameters  *jsonschema.Definition `json:"parameters,omitempty"`
	Strict      *bool                  `json:"strict,omitempty"`

	// Web search tool fields (used when type == "web_search_preview" etc.)
	SearchContextSize *string       `json:"search_context_size,omitempty"`
	UserLocation      *UserLocation `json:"user_location,omitempty"`
}

// IsFunction returns true if this is a function tool
func (t *Tool) IsFunction() bool {
	return t.Type == "function"
}

// IsWebSearch returns true if this is a web search tool
func (t *Tool) IsWebSearch() bool {
	return t.Type == "web_search_preview" || t.Type == "web_search_preview_2025_03_11"
}

// ToActionDefinition converts this tool to an ActionDefinition
func (t *Tool) ToActionDefinition() *coreTypes.ActionDefinition {
	if t.IsFunction() && t.Name != nil {
		// Regular function tool
		properties := make(map[string]jsonschema.Definition)
		required := []string{}

		if t.Parameters != nil {
			properties = t.Parameters.Properties
			required = t.Parameters.Required
		}

		desc := ""
		if t.Description != nil {
			desc = *t.Description
		}

		return &coreTypes.ActionDefinition{
			Name:        coreTypes.ActionDefinitionName(*t.Name),
			Description: desc,
			Properties:  properties,
			Required:    required,
		}
	}

	if t.IsWebSearch() {
		// Convert web search builtin to ActionDefinition
		name := "web_search_" + t.Type
		desc := "Web search tool for finding relevant information online"

		// Create parameters schema for web search options
		properties := map[string]jsonschema.Definition{
			"search_context_size": {
				Type:        jsonschema.String,
				Enum:        []string{"low", "medium", "high"},
				Description: "Amount of context window space to use for search",
			},
			"user_location": {
				Type: jsonschema.Object,
				Properties: map[string]jsonschema.Definition{
					"type": {
						Type:        jsonschema.String,
						Enum:        []string{"approximate"},
						Description: "Type of location approximation",
					},
					"city": {
						Type:        jsonschema.String,
						Description: "City of the user",
					},
					"country": {
						Type:        jsonschema.String,
						Description: "Two-letter ISO country code",
					},
					"region": {
						Type:        jsonschema.String,
						Description: "Region of the user",
					},
					"timezone": {
						Type:        jsonschema.String,
						Description: "IANA timezone of the user",
					},
				},
			},
		}

		return &coreTypes.ActionDefinition{
			Name:        coreTypes.ActionDefinitionName(name),
			Description: desc,
			Properties:  properties,
			Required:    []string{},
		}
	}

	return nil
}

// SeparateTools separates a slice of Tools into builtin tools and user tools as ActionDefinitions
func SeparateTools(tools []Tool) (builtinTools []coreTypes.ActionDefinition, userTools []coreTypes.ActionDefinition) {
	for _, tool := range tools {
		if actionDef := tool.ToActionDefinition(); actionDef != nil {
			if tool.IsFunction() {
				// User-defined function tool
				userTools = append(userTools, *actionDef)
			} else if tool.IsWebSearch() {
				// Builtin tool (web search)
				builtinTools = append(builtinTools, *actionDef)
			}
		}
	}
	return builtinTools, userTools
}

// UserLocation represents the user's location for web search
type UserLocation struct {
	Type     string  `json:"type"`
	City     *string `json:"city,omitempty"`
	Country  *string `json:"country,omitempty"`
	Region   *string `json:"region,omitempty"`
	Timezone *string `json:"timezone,omitempty"`
}
