package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func GenerateTypedJSONWithGuidance(ctx context.Context, client LLMClient, guidance, model string, i jsonschema.Definition, dst any) error {
	return GenerateTypedJSONWithConversation(ctx, client, []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: guidance,
		},
	}, model, i, dst)
}

func GenerateTypedJSONWithConversation(ctx context.Context, client LLMClient, conv []openai.ChatCompletionMessage, model string, i jsonschema.Definition, dst any) error {
	toolName := "json"
	decision := openai.ChatCompletionRequest{
		Model:    model,
		Messages: conv,
		Tools: []openai.Tool{
			{

				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:       toolName,
					Parameters: i,
				},
			},
		},
		ToolChoice: openai.ToolChoice{
			Type:     openai.ToolTypeFunction,
			Function: openai.ToolFunction{Name: toolName},
		},
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

	xlog.Debug("JSON generated", "Arguments", msg.ToolCalls[0].Function.Arguments)

	return json.Unmarshal([]byte(msg.ToolCalls[0].Function.Arguments), dst)
}
