package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// generateAnswer generates an answer for the given text using the OpenAI API
func GenerateJSON(client *openai.Client, model, text string, i interface{}) error {
	req := openai.ChatCompletionRequest{
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
		Model:          model,
		Messages: []openai.ChatCompletionMessage{
			{

				Role:    "user",
				Content: text,
			},
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to generate answer: %v", err)
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("no response from OpenAI API")
	}

	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), i)
	if err != nil {
		return err
	}
	return nil
}

func GenerateJSONFromStruct(client *openai.Client, guidance, model string, i interface{}) error {
	// TODO: use functions?
	exampleJSON, err := json.Marshal(i)
	if err != nil {
		return err
	}
	return GenerateJSON(client, model, "Generate a character as JSON data. "+guidance+". This is the JSON fields that should contain: "+string(exampleJSON), i)
}
