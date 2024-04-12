package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mudler/local-agent-framework/action"
	"github.com/mudler/local-agent-framework/llm"
	"github.com/sashabaranov/go-openai"
)

const (
	UserRole      = "user"
	AssistantRole = "assistant"
	SystemRole    = "system"
)

type Agent struct {
	sync.Mutex
	options       *options
	Character     Character
	client        *openai.Client
	jobQueue      chan *Job
	actionContext *action.ActionContext
	context       *action.ActionContext

	currentReasoning         string
	currentState             *action.StateResult
	nextAction               Action
	currentConversation      Messages
	selfEvaluationInProgress bool

	newConversations chan openai.ChatCompletionMessage
}

type RAGDB interface {
	Store(s string) error
	Reset() error
	Search(s string, similarEntries int) ([]string, error)
}

func New(opts ...Option) (*Agent, error) {
	options, err := newOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to set options: %v", err)
	}

	client := llm.NewClient(options.LLMAPI.APIKey, options.LLMAPI.APIURL)

	c := context.Background()
	if options.context != nil {
		c = options.context
	}

	ctx, cancel := context.WithCancel(c)
	a := &Agent{
		jobQueue:     make(chan *Job),
		options:      options,
		client:       client,
		Character:    options.character,
		currentState: &action.StateResult{},
		context:      action.NewContext(ctx, cancel),
	}

	if a.options.statefile != "" {
		if _, err := os.Stat(a.options.statefile); err == nil {
			if err = a.LoadState(a.options.statefile); err != nil {
				return a, fmt.Errorf("failed to load state: %v", err)
			}
		}
	}

	if a.options.debugMode {
		fmt.Println("=== Agent in Debug mode ===")
		fmt.Println(a.Character.String())
		fmt.Println(a.State().String())
		fmt.Println("Permanent goal: ", a.options.permanentGoal)
	}

	return a, nil
}

// StopAction stops the current action
// if any. Can be called before adding a new job.
func (a *Agent) StopAction() {
	a.Lock()
	defer a.Unlock()
	if a.actionContext != nil {
		a.actionContext.Cancel()
	}
}

func (a *Agent) Context() context.Context {
	return a.context.Context
}

func (a *Agent) ConversationChannel() chan openai.ChatCompletionMessage {
	return a.newConversations
}

// Ask is a pre-emptive, blocking call that returns the response as soon as it's ready.
// It discards any other computation.
func (a *Agent) Ask(opts ...JobOption) *JobResult {
	a.StopAction()
	j := NewJob(append(opts, WithReasoningCallback(a.options.reasoningCallback), WithResultCallback(a.options.resultCallback))...)
	//	fmt.Println("Job created", text)
	a.jobQueue <- j
	return j.Result.WaitResult()
}

func (a *Agent) CurrentConversation() []openai.ChatCompletionMessage {
	a.Lock()
	defer a.Unlock()
	return a.currentConversation
}

func (a *Agent) SetConversation(conv []openai.ChatCompletionMessage) {
	a.Lock()
	defer a.Unlock()
	a.currentConversation = conv
}

func (a *Agent) ResetConversation() {
	a.Lock()
	defer a.Unlock()
	a.currentConversation = []openai.ChatCompletionMessage{}
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	a.Lock()
	defer a.Unlock()
	a.context.Cancel()
}

func (a *Agent) runAction(chosenAction Action, decisionResult *decisionResult) (result string, err error) {
	for _, action := range a.systemInternalActions() {
		if action.Definition().Name == chosenAction.Definition().Name {
			if result, err = action.Run(decisionResult.actionParams); err != nil {
				return "", fmt.Errorf("error running action: %w", err)
			}
		}
	}

	if a.options.debugMode {
		fmt.Println("Action", chosenAction.Definition().Name)
		fmt.Println("Result", result)
	}

	if chosenAction.Definition().Name.Is(action.StateActionName) {
		// We need to store the result in the state
		state := action.StateResult{}

		err = decisionResult.actionParams.Unmarshal(&state)
		if err != nil {
			return "", fmt.Errorf("error unmarshalling state of the agent: %w", err)
		}
		// update the current state with the one we just got from the action
		a.currentState = &state

		// update the state file
		if a.options.statefile != "" {
			if err := a.SaveState(a.options.statefile); err != nil {
				return "", err
			}
		}
	}

	return result, nil
}

func (a *Agent) consumeJob(job *Job, role string) {
	// We are self evaluating if we consume the job as a system role
	selfEvaluation := role == SystemRole

	memory := a.options.enableKB && a.options.ragdb != nil

	a.Lock()
	// Set the action context
	ctx, cancel := context.WithCancel(context.Background())
	a.actionContext = action.NewContext(ctx, cancel)
	a.selfEvaluationInProgress = selfEvaluation
	if len(job.conversationHistory) != 0 {
		a.currentConversation = job.conversationHistory
	}
	a.Unlock()

	if selfEvaluation {
		defer func() {
			a.Lock()
			a.selfEvaluationInProgress = false
			a.Unlock()
		}()
	}

	//if job.Image != "" {
	// TODO: Use llava to explain the image content
	//}
	// Add custom prompts
	for _, prompt := range a.options.prompts {
		message := prompt.Render(a)
		if !Messages(a.currentConversation).Exist(a.options.systemPrompt) {
			a.currentConversation = append([]openai.ChatCompletionMessage{
				{
					Role:    prompt.Role(),
					Content: message,
				}}, a.currentConversation...)
		}
	}

	// TODO: move to a Promptblock?
	if a.options.systemPrompt != "" {
		if !Messages(a.currentConversation).Exist(a.options.systemPrompt) {
			a.currentConversation = append([]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: a.options.systemPrompt,
				}}, a.currentConversation...)
		}
	}

	// TODO: move to a promptblock?
	// RAG
	if memory {
		// Walk conversation from bottom to top, and find the first message of the user
		// to use it as a query to the KB
		var userMessage string
		for i := len(a.currentConversation) - 1; i >= 0; i-- {
			if a.currentConversation[i].Role == "user" {
				userMessage = a.currentConversation[i].Content
				break
			}
		}

		if userMessage != "" {
			results, err := a.options.ragdb.Search(userMessage, a.options.kbResults)
			if err != nil {
				if a.options.debugMode {
					fmt.Println("Error finding similar strings inside KB:", err)
				}
				//	job.Result.Finish(fmt.Errorf("error finding similar strings inside KB: %w", err))
				//	return
			}

			if len(results) != 0 {

				formatResults := ""
				for _, r := range results {
					formatResults += fmt.Sprintf("- %s \n", r)
				}
				if a.options.debugMode {
					fmt.Println("Found similar strings in KB:")
					fmt.Println(formatResults)
				}
				// a.currentConversation = append(a.currentConversation,
				// 	openai.ChatCompletionMessage{
				// 		Role:    "system",
				// 		Content: fmt.Sprintf("Given the user input you have the following in memory:\n%s", formatResults),
				// 	},
				// )
				a.currentConversation = append([]openai.ChatCompletionMessage{
					{
						Role:    "system",
						Content: fmt.Sprintf("Given the user input you have the following in memory:\n%s", formatResults),
					}}, a.currentConversation...)
			}
		}
	}

	if job.Text != "" {
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role:    role,
			Content: job.Text,
		})
	}

	var pickTemplate string
	var reEvaluationTemplate string

	if selfEvaluation {
		pickTemplate = pickSelfTemplate
		reEvaluationTemplate = reSelfEvalTemplate
	} else {
		pickTemplate = pickActionTemplate
		reEvaluationTemplate = reEvalTemplate
	}

	// choose an action first
	var chosenAction Action
	var reasoning string

	if a.currentReasoning != "" && a.nextAction != nil {
		// if we are being re-evaluated, we already have the action
		// and the reasoning. Consume it here and reset it
		chosenAction = a.nextAction
		reasoning = a.currentReasoning
		a.currentReasoning = ""
		a.nextAction = nil
	} else {
		var err error
		chosenAction, reasoning, err = a.pickAction(ctx, pickTemplate, a.currentConversation)
		if err != nil {
			job.Result.Finish(err)
			return
		}
	}

	if chosenAction == nil {
		// If no action was picked up, the reasoning is the message returned by the assistant
		// so we can consume it as if it was a reply.
		//job.Result.SetResult(ActionState{ActionCurrentState{nil, nil, "No action to do, just reply"}, ""})
		//job.Result.Finish(fmt.Errorf("no action to do"))\
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: reasoning,
		})
		job.Result.SetResponse(reasoning)
		job.Result.Finish(nil)
		return
	}

	if a.options.debugMode {
		fmt.Println("===> Generating parameters for", chosenAction.Definition().Name)
	}

	params, err := a.generateParameters(ctx, pickTemplate, chosenAction, a.currentConversation, reasoning)
	if err != nil {
		job.Result.Finish(fmt.Errorf("error generating action's parameters: %w", err))
		return
	}

	if a.options.debugMode {
		fmt.Println("===> Generated parameters for", chosenAction.Definition().Name)
		fmt.Println(params.actionParams.String())
	}

	if params.actionParams == nil {
		job.Result.Finish(fmt.Errorf("no parameters"))
		return
	}

	if !job.Callback(ActionCurrentState{chosenAction, params.actionParams, reasoning}) {
		job.Result.SetResult(ActionState{ActionCurrentState{chosenAction, params.actionParams, reasoning}, "stopped by callback"})
		job.Result.Finish(nil)
		return
	}

	if selfEvaluation && a.options.initiateConversations &&
		chosenAction.Definition().Name.Is(action.ConversationActionName) {

		message := action.ConversationActionResponse{}
		if err := params.actionParams.Unmarshal(&message); err != nil {
			job.Result.Finish(fmt.Errorf("error unmarshalling conversation response: %w", err))
			return
		}

		a.currentConversation = []openai.ChatCompletionMessage{
			{
				Role:    "assistant",
				Content: message.Message,
			},
		}
		go func() {
			a.newConversations <- openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: message.Message,
			}
		}()
		job.Result.SetResponse("decided to initiate a new conversation")
		job.Result.Finish(nil)
		return
	}

	// If we don't have to reply , run the action!
	if !chosenAction.Definition().Name.Is(action.ReplyActionName) {
		result, err := a.runAction(chosenAction, params)
		if err != nil {
			job.Result.Finish(fmt.Errorf("error running action: %w", err))
			return
		}

		stateResult := ActionState{ActionCurrentState{chosenAction, params.actionParams, reasoning}, result}
		job.Result.SetResult(stateResult)
		job.CallbackWithResult(stateResult)

		// calling the function
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role: "assistant",
			FunctionCall: &openai.FunctionCall{
				Name:      chosenAction.Definition().Name.String(),
				Arguments: params.actionParams.String(),
			},
		})

		// result of calling the function
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result,
			Name:       chosenAction.Definition().Name.String(),
			ToolCallID: chosenAction.Definition().Name.String(),
		})

		//a.currentConversation = append(a.currentConversation, messages...)
		//a.currentConversation = messages

		// given the result, we can now ask OpenAI to complete the conversation or
		// to continue using another tool given the result
		followingAction, reasoning, err := a.pickAction(ctx, reEvaluationTemplate, a.currentConversation)
		if err != nil {
			job.Result.Finish(fmt.Errorf("error picking action: %w", err))
			return
		}

		if followingAction != nil &&
			!followingAction.Definition().Name.Is(action.ReplyActionName) &&
			!chosenAction.Definition().Name.Is(action.ReplyActionName) {
			// We need to do another action (?)
			// The agent decided to do another action
			// call ourselves again
			a.currentReasoning = reasoning
			a.nextAction = followingAction
			job.Text = ""
			a.consumeJob(job, role)
			return
		}
	}

	// decode the response
	replyResponse := action.ReplyResponse{}

	if err := params.actionParams.Unmarshal(&replyResponse); err != nil {
		job.Result.Finish(fmt.Errorf("error unmarshalling reply response: %w", err))
		return
	}

	// If we have already a reply from the action, just return it.
	// Otherwise generate a full conversation to get a proper message response
	// if chosenAction.Definition().Name.Is(action.ReplyActionName) {
	// 	replyResponse := action.ReplyResponse{}
	// 	if err := params.actionParams.Unmarshal(&replyResponse); err != nil {
	// 		job.Result.Finish(fmt.Errorf("error unmarshalling reply response: %w", err))
	// 		return
	// 	}
	// 	if replyResponse.Message != "" {
	// 		job.Result.SetResponse(replyResponse.Message)
	// 		job.Result.Finish(nil)
	// 		return
	// 	}
	// }

	// If we have a hud, display it
	if a.options.enableHUD {
		var promptHUD *PromptHUD
		if a.options.enableHUD {
			h := a.prepareHUD()
			promptHUD = &h
		}

		prompt, err := renderTemplate(hudTemplate, promptHUD, a.systemInternalActions(), reasoning)
		if err != nil {
			job.Result.Finish(fmt.Errorf("error renderTemplate: %w", err))
			return
		}
		if !a.currentConversation.Exist(prompt) {
			a.currentConversation = append([]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: prompt,
				},
			}, a.currentConversation...)
		}
	}

	// Generate a human-readable response
	// resp, err := a.client.CreateChatCompletion(ctx,
	// 	openai.ChatCompletionRequest{
	// 		Model: a.options.LLMAPI.Model,
	// 		Messages: append(a.currentConversation,
	// 			openai.ChatCompletionMessage{
	// 				Role:    "system",
	// 				Content: "Assistant thought: " + replyResponse.Message,
	// 			},
	// 		),
	// 	},
	// )
	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: a.currentConversation,
		},
	)
	if err != nil {
		job.Result.Finish(err)
		return
	}

	if len(resp.Choices) != 1 {
		job.Result.Finish(fmt.Errorf("no enough choices: %w", err))
		return
	}

	// display OpenAI's response to the original question utilizing our function
	msg := resp.Choices[0].Message

	// If we didn't got any message, we can use the response from the action
	if chosenAction.Definition().Name.Is(action.ReplyActionName) && msg.Content == "" {
		if a.options.debugMode {
			fmt.Println("No output returned from conversation, using the action response as a reply.")
		}

		msg.Content = replyResponse.Message
	}

	a.currentConversation = append(a.currentConversation, msg)
	job.Result.SetResponse(msg.Content)
	job.Result.Finish(nil)
}

func (a *Agent) periodicallyRun() {
	// This is running in the background.

	// TODO: Would be nice if we have a special action to
	// contact the user. This would actually make sure that
	// if the agent wants to initiate a conversation, it can do so.
	// This would be a special action that would be picked up by the agent
	// and would be used to contact the user.

	if a.options.debugMode {
		fmt.Println("START -- Periodically run is starting")
	}

	if len(a.CurrentConversation()) != 0 {
		// Here the LLM could decide to store some part of the conversation too in the memory
		evaluateMemory := NewJob(
			WithText(
				`Evaluate the current conversation and decide if we need to store some relevant informations from it`,
			),
			WithReasoningCallback(a.options.reasoningCallback),
			WithResultCallback(a.options.resultCallback),
		)
		a.consumeJob(evaluateMemory, SystemRole)

		a.ResetConversation()
	}

	// Here we go in a loop of
	// - asking the agent to do something
	// - evaluating the result
	// - asking the agent to do something else based on the result

	//	whatNext := NewJob(WithText("Decide what to do based on the state"))
	whatNext := NewJob(
		WithText("Decide what to based on the goal and the persistent goal."),
		WithReasoningCallback(a.options.reasoningCallback),
		WithResultCallback(a.options.resultCallback),
	)
	a.consumeJob(whatNext, SystemRole)
	a.ResetConversation()

	if a.options.debugMode {
		fmt.Println("STOP -- Periodically run is done")
	}
	// Save results from state

	// a.ResetConversation()

	// doWork := NewJob(WithText("Select the tool to use based on your goal and the current state."))
	// a.consumeJob(doWork, SystemRole)

	// results := []string{}
	// for _, v := range doWork.Result.State {
	// 	results = append(results, v.Result)
	// }

	// a.ResetConversation()

	// // Here the LLM could decide to do something based on the result of our automatic action
	// evaluateAction := NewJob(
	// 	WithText(
	// 		`Evaluate the current situation and decide if we need to execute other tools (for instance to store results into permanent, or short memory).
	// 		We have done the following actions:
	// 		` + strings.Join(results, "\n"),
	// 	))
	// a.consumeJob(evaluateAction, SystemRole)

	// a.ResetConversation()
}

func (a *Agent) prepareIdentity() error {

	if a.options.characterfile != "" {
		if _, err := os.Stat(a.options.characterfile); err == nil {
			// if there is a file, load the character back
			if err = a.LoadCharacter(a.options.characterfile); err != nil {
				return fmt.Errorf("failed to load character: %v", err)
			}
		} else {
			if a.options.randomIdentity {
				if err = a.generateIdentity(a.options.randomIdentityGuidance); err != nil {
					return fmt.Errorf("failed to generate identity: %v", err)
				}
			}

			// otherwise save it for next time
			if err = a.SaveCharacter(a.options.characterfile); err != nil {
				return fmt.Errorf("failed to save character: %v", err)
			}
		}
	} else {
		if err := a.generateIdentity(a.options.randomIdentityGuidance); err != nil {
			return fmt.Errorf("failed to generate identity: %v", err)
		}
	}

	return nil
}

func (a *Agent) Run() error {
	// The agent run does two things:
	// picks up requests from a queue
	// and generates a response/perform actions

	if err := a.prepareIdentity(); err != nil {
		return fmt.Errorf("failed to prepare identity: %v", err)
	}

	// It is also preemptive.
	// That is, it can interrupt the current action
	// if another one comes in.

	// If there is no action, periodically evaluate if it has to do something on its own.

	// Expose a REST API to interact with the agent to ask it things

	//todoTimer := time.NewTicker(a.options.periodicRuns)
	timer := time.NewTimer(a.options.periodicRuns)
	for {
		select {
		case job := <-a.jobQueue:
			// Consume the job and generate a response
			// TODO: Give a short-term memory to the agent
			a.consumeJob(job, UserRole)
			timer.Reset(a.options.periodicRuns)
		case <-a.context.Done():
			// Agent has been canceled, return error
			return ErrContextCanceled
		case <-timer.C:
			if !a.options.standaloneJob {
				continue
			}
			// TODO: We should also do not immediately fire this timer but
			// instead have a cool-off timer starting after we received the last job
			a.periodicallyRun()
			timer.Reset(a.options.periodicRuns)
		}
	}
}
