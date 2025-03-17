package types

import "github.com/sashabaranov/go-openai"

// RequestBody represents the request body structure for the OpenAI API
type RequestBody struct {
	Model              string            `json:"model"`
	Input              interface{}       `json:"input"`
	InputText          string            `json:"input_text"`
	InputMessages      []InputMessage    `json:"input_messages"`
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
	ToolChoice         interface{}       `json:"tool_choice,omitempty"`
	Tools              []interface{}     `json:"tools,omitempty"`
	TopP               *float64          `json:"top_p,omitempty"`
	Truncation         *string           `json:"truncation,omitempty"`
}

func (r *RequestBody) SetInputByType() {
	switch input := r.Input.(type) {
	case string:
		r.InputText = input
	case []any:
		for _, i := range input {
			switch i := i.(type) {
			case InputMessage:
				r.InputMessages = append(r.InputMessages, i)
			}
		}
	}
}

func (r *RequestBody) ToChatCompletionMessages() []openai.ChatCompletionMessage {
	result := []openai.ChatCompletionMessage{}

	for _, m := range r.InputMessages {
		content := []openai.ChatMessagePart{}
		oneImageWasFound := false
		for _, c := range m.Content {
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

		if oneImageWasFound {
			result = append(result, openai.ChatCompletionMessage{
				Role:         m.Role,
				MultiContent: content,
			})
		} else {
			for _, c := range content {
				result = append(result, openai.ChatCompletionMessage{
					Role:    m.Role,
					Content: c.Text,
				})
			}
		}
	}

	if r.InputText != "" {
		result = append(result, openai.ChatCompletionMessage{
			Role:    "user",
			Content: r.InputText,
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
	Output             []ResponseMessage      `json:"output"`
	ParallelToolCalls  bool                   `json:"parallel_tool_calls"`
	PreviousResponseID interface{}            `json:"previous_response_id"`
	Reasoning          ReasoningConfig        `json:"reasoning"`
	Store              bool                   `json:"store"`
	Temperature        float64                `json:"temperature"`
	Text               TextConfig             `json:"text"`
	ToolChoice         string                 `json:"tool_choice"`
	Tools              []interface{}          `json:"tools"`
	TopP               float64                `json:"top_p"`
	Truncation         string                 `json:"truncation"`
	Usage              UsageInfo              `json:"usage"`
	User               interface{}            `json:"user"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// InputMessage represents a user input message
type InputMessage struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

// ContentItem represents an item in a content array
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}
