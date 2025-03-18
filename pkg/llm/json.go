package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// generateAnswer generates an answer for the given text using the OpenAI API
func GenerateJSON(ctx context.Context, client *openai.Client, model, text string, i interface{}) error {
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

	resp, err := client.CreateChatCompletion(ctx, req)
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

func GenerateJSONFromStruct(ctx context.Context, client *openai.Client, guidance, model string, i interface{}) error {
	// TODO: use functions?
	exampleJSON, err := json.Marshal(i)
	if err != nil {
		return err
	}
	return GenerateJSON(ctx, client, model, "Generate a character as JSON data. "+guidance+". This is the JSON fields that should contain: "+string(exampleJSON), i)
}

func GenerateTypedJSON(ctx context.Context, client *openai.Client, guidance, model string, i jsonschema.Definition, dst interface{}) error {
	decision := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Generate a character as JSON data. " + guidance,
			},
		},
		Tools: []openai.Tool{
			{

				Type: openai.ToolTypeFunction,
				Function: openai.FunctionDefinition{
					Name:       "identity",
					Parameters: i,
				},
			},
		},
		ToolChoice: "identity",
	}

	resp, err := client.CreateChatCompletion(ctx, decision)
	if err != nil {
		return err
	}

	if len(resp.Choices) != 1 {
		return fmt.Errorf("no choices: %d", len(resp.Choices))
	}

	msg := resp.Choices[0].Message

	if len(msg.ToolCalls) == 0 {
		return fmt.Errorf("no tool calls: %d", len(msg.ToolCalls))
	}

	return json.Unmarshal([]byte(msg.ToolCalls[0].Function.Arguments), dst)
}
