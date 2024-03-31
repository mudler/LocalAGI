package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/mudler/local-agent-framework/action"
	"github.com/sashabaranov/go-openai"
)

// Job is a request to the agent to do something
type Job struct {
	// The job is a request to the agent to do something
	// It can be a question, a command, or a request to do something
	// The agent will try to do it, and return a response
	Text   string
	Image  string // base64 encoded image
	Result *JobResult
}

// JobResult is the result of a job
type JobResult struct {
	sync.Mutex
	// The result of a job
	Data  []string
	ready chan bool
}

// NewJobResult creates a new job result
func NewJobResult() *JobResult {
	return &JobResult{
		ready: make(chan bool),
	}
}

// NewJob creates a new job
// It is a request to the agent to do something
// It has a JobResult to get the result asynchronously
// To wait for a Job result, use JobResult.WaitResult()
func NewJob(text, image string) *Job {
	return &Job{
		Text:   text,
		Image:  image,
		Result: NewJobResult(),
	}
}

// SetResult sets the result of a job
func (j *JobResult) SetResult(text string) {
	j.Lock()
	defer j.Unlock()

	j.Data = append(j.Data, text)
}

// SetResult sets the result of a job
func (j *JobResult) Finish() {
	j.Lock()
	defer j.Unlock()

	close(j.ready)
}

// WaitResult waits for the result of a job
func (j *JobResult) WaitResult() []string {
	<-j.ready
	j.Lock()
	defer j.Unlock()
	return j.Data
}

func (a *Agent) consumeJob(job *Job) {

	// Consume the job and generate a response
	a.Lock()
	// Set the action context
	ctx, cancel := context.WithCancel(context.Background())
	a.actionContext = action.NewContext(ctx, cancel)
	a.Unlock()

	if job.Image != "" {
		// TODO: Use llava to explain the image content
	}

	if job.Text == "" {
		fmt.Println("no text!")
		return
	}

	messages := a.currentConversation
	if job.Text != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: job.Text,
		})
	}

	// choose an action first
	chosenAction, reasoning, err := a.pickAction(ctx, pickActionTemplate, messages)
	if err != nil {
		fmt.Printf("error picking action: %v\n", err)
		return
	}

	if chosenAction == nil || chosenAction.Definition().Name.Is(action.ReplyActionName) {
		fmt.Println("No action to do, just reply")
		job.Result.SetResult(reasoning)
		job.Result.Finish()
		return
	}

	params, err := a.generateParameters(ctx, chosenAction, messages)
	if err != nil {
		fmt.Printf("error generating parameters: %v\n", err)
		return
	}

	if params.actionParams == nil {
		fmt.Println("no parameters")
		return
	}

	var result string
	for _, action := range a.options.actions {
		fmt.Println("Checking action: ", action.Definition().Name, chosenAction.Definition().Name)
		if action.Definition().Name == chosenAction.Definition().Name {
			fmt.Printf("Running action: %v\n", action.Definition().Name)
			if result, err = action.Run(params.actionParams); err != nil {
				fmt.Printf("error running action: %v\n", err)
				return
			}
		}
	}
	fmt.Printf("Action run result: %v\n", result)

	// calling the function
	messages = append(messages, openai.ChatCompletionMessage{
		Role: "assistant",
		FunctionCall: &openai.FunctionCall{
			Name:      chosenAction.Definition().Name.String(),
			Arguments: params.actionParams.String(),
		},
	})

	// result of calling the function
	messages = append(messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result,
		Name:       chosenAction.Definition().Name.String(),
		ToolCallID: chosenAction.Definition().Name.String(),
	})

	// given the result, we can now ask OpenAI to complete the conversation or
	// to continue using another tool given the result
	followingAction, reasoning, err := a.pickAction(ctx, reEvalTemplate, messages)
	if err != nil {
		fmt.Printf("error picking action: %v\n", err)
		return
	}

	if followingAction == nil || followingAction.Definition().Name.Is(action.ReplyActionName) {
		fmt.Println("No action to do, just reply")
	} else if !chosenAction.Definition().Name.Is(action.ReplyActionName) {
		// We need to do another action (?)
		// The agent decided to do another action
		fmt.Println("Another action to do: ", followingAction.Definition().Name)
		fmt.Println("Reasoning: ", reasoning)
		return
	}

	// Generate a human-readable response
	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: messages,
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		fmt.Printf("2nd completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
		return
	}

	// display OpenAI's response to the original question utilizing our function
	msg := resp.Choices[0].Message
	fmt.Printf("OpenAI answered the original request with: %v\n",
		msg.Content)

	messages = append(messages, msg)
	a.currentConversation = append(a.currentConversation, messages...)

	if len(msg.ToolCalls) != 0 {
		fmt.Printf("OpenAI wants to call again functions: %v\n", msg)
		// wants to call again an action (?)
		job.Text = "" // Call the job with the current conversation
		job.Result.SetResult(result)
		a.jobQueue <- job
		return
	}

	// perform the action (if any)
	// or reply with a result
	// if there is an action...
	job.Result.SetResult(result)
	job.Result.Finish()
}
