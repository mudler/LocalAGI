package agent

import (
	"fmt"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type EvaluationResult struct {
	Satisfied bool     `json:"satisfied"`
	Gaps      []string `json:"gaps"`
	Reasoning string   `json:"reasoning"`
}

type GoalExtraction struct {
	Goal        string   `json:"goal"`
	Constraints []string `json:"constraints"`
	Context     string   `json:"context"`
}

func (a *Agent) extractGoal(job *types.Job, conv []openai.ChatCompletionMessage) (*GoalExtraction, error) {
	// Create the goal extraction schema
	schema := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"goal": {
				Type:        jsonschema.String,
				Description: "The main goal or request from the user",
			},
			"constraints": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
				Description: "Any constraints or requirements specified by the user",
			},
			"context": {
				Type:        jsonschema.String,
				Description: "Additional context that might be relevant for understanding the goal",
			},
		},
		Required: []string{"goal", "constraints", "context"},
	}

	// Create the goal extraction prompt
	prompt := `Analyze the conversation and extract the user's main goal, any constraints, and relevant context.
Consider the entire conversation history to understand the complete context and requirements.
Focus on identifying the primary objective and any specific requirements or limitations mentioned.`

	var result GoalExtraction
	err := llm.GenerateTypedJSONWithConversation(job.GetContext(), a.client,
		append(
			[]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: prompt,
				},
			},
			conv...), a.options.LLMAPI.Model, schema, &result)
	if err != nil {
		return nil, fmt.Errorf("error extracting goal: %w", err)
	}

	return &result, nil
}

func (a *Agent) evaluateJob(job *types.Job, conv []openai.ChatCompletionMessage) (*EvaluationResult, error) {
	if !a.options.enableEvaluation {
		return &EvaluationResult{Satisfied: true}, nil
	}

	// Extract the goal first
	goal, err := a.extractGoal(job, conv)
	if err != nil {
		return nil, fmt.Errorf("error extracting goal: %w", err)
	}

	// Create the evaluation schema
	schema := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"satisfied": {
				Type: jsonschema.Boolean,
			},
			"gaps": {
				Type: jsonschema.Array,
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"reasoning": {
				Type: jsonschema.String,
			},
		},
		Required: []string{"satisfied", "gaps", "reasoning"},
	}

	// Create the evaluation prompt
	prompt := fmt.Sprintf(`Evaluate if the assistant has satisfied the user's request. Consider:
1. The identified goal: %s
2. Constraints and requirements: %v
3. Context: %s
4. The conversation history
5. Any gaps or missing information
6. Whether the response fully addresses the user's needs

Provide a detailed evaluation with specific gaps if any are found.`,
		goal.Goal,
		goal.Constraints,
		goal.Context)

	var result EvaluationResult
	err = llm.GenerateTypedJSONWithConversation(job.GetContext(), a.client,
		append(
			[]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: prompt,
				},
			},
			conv...),
		a.options.LLMAPI.Model, schema, &result)
	if err != nil {
		return nil, fmt.Errorf("error generating evaluation: %w", err)
	}

	return &result, nil
}

func (a *Agent) handleEvaluation(job *types.Job, conv []openai.ChatCompletionMessage, currentLoop int) (bool, []openai.ChatCompletionMessage, error) {
	if !a.options.enableEvaluation || currentLoop >= a.options.maxEvaluationLoops {
		return true, conv, nil
	}

	result, err := a.evaluateJob(job, conv)
	if err != nil {
		return false, conv, err
	}

	if result.Satisfied {
		return true, conv, nil
	}

	// If there are gaps, we need to address them
	if len(result.Gaps) > 0 {
		// Add the evaluation result to the conversation
		conv = append(conv, openai.ChatCompletionMessage{
			Role: "system",
			Content: fmt.Sprintf("Evaluation found gaps that need to be addressed:\n%s\nReasoning: %s",
				result.Gaps, result.Reasoning),
		})

		xlog.Debug("Evaluation found gaps, incrementing loop count", "loop", currentLoop+1)
		return false, conv, nil
	}

	return true, conv, nil
}
