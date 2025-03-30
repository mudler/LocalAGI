package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mudler/LocalAgent/pkg/xlog"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/core/types"
	"github.com/mudler/LocalAgent/pkg/llm"
	"github.com/sashabaranov/go-openai"
)

const (
	UserRole      = "user"
	AssistantRole = "assistant"
	SystemRole    = "system"
	maxRetries    = 5
)

type Agent struct {
	sync.Mutex
	options   *options
	Character Character
	client    *openai.Client
	jobQueue  chan *types.Job
	context   *types.ActionContext

	currentState *action.AgentInternalState

	selfEvaluationInProgress bool
	pause                    bool

	newConversations chan openai.ChatCompletionMessage

	mcpActions types.Actions

	subscriberMutex        sync.Mutex
	newMessagesSubscribers []func(openai.ChatCompletionMessage)
}

type RAGDB interface {
	Store(s string) error
	Reset() error
	Search(s string, similarEntries int) ([]string, error)
	Count() int
}

func New(opts ...Option) (*Agent, error) {
	options, err := newOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to set options: %v", err)
	}

	client := llm.NewClient(options.LLMAPI.APIKey, options.LLMAPI.APIURL, options.timeout)

	c := context.Background()
	if options.context != nil {
		c = options.context
	}

	ctx, cancel := context.WithCancel(c)
	a := &Agent{
		jobQueue:               make(chan *types.Job),
		options:                options,
		client:                 client,
		Character:              options.character,
		currentState:           &action.AgentInternalState{},
		context:                types.NewActionContext(ctx, cancel),
		newConversations:       make(chan openai.ChatCompletionMessage),
		newMessagesSubscribers: options.newConversationsSubscribers,
	}

	if a.options.statefile != "" {
		if _, err := os.Stat(a.options.statefile); err == nil {
			if err = a.LoadState(a.options.statefile); err != nil {
				return a, fmt.Errorf("failed to load state: %v", err)
			}
		}
	}

	// var programLevel = new(xlog.LevelVar) // Info by default
	// h := xlog.NewTextHandler(os.Stdout, &xlog.HandlerOptions{Level: programLevel})
	// xlog = xlog.New(h)
	//programLevel.Set(a.options.logLevel)

	if err := a.prepareIdentity(); err != nil {
		return nil, fmt.Errorf("failed to prepare identity: %v", err)
	}

	xlog.Info("Populating actions from MCP Servers (if any)")
	a.initMCPActions()
	xlog.Info("Done populating actions from MCP Servers")

	xlog.Info(
		"Agent created",
		"agent", a.Character.Name,
		"character", a.Character.String(),
		"state", a.State().String(),
		"goal", a.options.permanentGoal,
		"model", a.options.LLMAPI.Model,
	)

	return a, nil
}

func (a *Agent) startNewConversationsConsumer() {
	go func() {
		for {
			select {
			case <-a.context.Done():
				return

			case msg := <-a.newConversations:
				xlog.Debug("New conversation", "agent", a.Character.Name, "message", msg.Content)
				a.subscriberMutex.Lock()
				subs := a.newMessagesSubscribers
				a.subscriberMutex.Unlock()
				for _, s := range subs {
					s(msg)
				}
			}
		}
	}()
}

func (a *Agent) AddSubscriber(f func(openai.ChatCompletionMessage)) {
	a.subscriberMutex.Lock()
	defer a.subscriberMutex.Unlock()
	a.newMessagesSubscribers = append(a.newMessagesSubscribers, f)
}

func (a *Agent) Context() context.Context {
	return a.context.Context
}

// Ask is a blocking call that returns the response as soon as it's ready.
// It discards any other computation.
func (a *Agent) Ask(opts ...types.JobOption) *types.JobResult {
	xlog.Debug("Agent Ask()", "agent", a.Character.Name, "model", a.options.LLMAPI.Model)
	defer func() {
		xlog.Debug("Agent has finished being asked", "agent", a.Character.Name)
	}()

	return a.Execute(types.NewJob(
		append(
			opts,
			types.WithReasoningCallback(a.options.reasoningCallback),
			types.WithResultCallback(a.options.resultCallback),
		)...,
	))
}

// Ask is a pre-emptive, blocking call that returns the response as soon as it's ready.
// It discards any other computation.
func (a *Agent) Execute(j *types.Job) *types.JobResult {
	xlog.Debug("Agent Execute()", "agent", a.Character.Name, "model", a.options.LLMAPI.Model)
	defer func() {
		xlog.Debug("Agent has finished", "agent", a.Character.Name)
	}()

	a.Enqueue(j)
	return j.Result.WaitResult()
}

func (a *Agent) Enqueue(j *types.Job) {
	j.ReasoningCallback = a.options.reasoningCallback
	j.ResultCallback = a.options.resultCallback

	a.jobQueue <- j
}

func (a *Agent) askLLM(ctx context.Context, conversation []openai.ChatCompletionMessage, maxRetries int) (openai.ChatCompletionMessage, error) {
	var resp openai.ChatCompletionResponse
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = a.client.CreateChatCompletion(ctx,
			openai.ChatCompletionRequest{
				Model:    a.options.LLMAPI.Model,
				Messages: conversation,
			},
		)
		if err == nil && len(resp.Choices) == 1 && resp.Choices[0].Message.Content != "" {
			break
		}
		xlog.Warn("Error asking LLM, retrying", "attempt", attempt+1, "error", err)
		if attempt < maxRetries {
			time.Sleep(2 * time.Second) // Optional: Add a delay between retries
		}
	}

	if err != nil {
		return openai.ChatCompletionMessage{}, err
	}

	if len(resp.Choices) != 1 {
		return openai.ChatCompletionMessage{}, fmt.Errorf("not enough choices: %w", err)
	}

	return resp.Choices[0].Message, nil
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	a.Lock()
	defer a.Unlock()
	xlog.Debug("Stopping agent", "agent", a.Character.Name)
	a.context.Cancel()
}

func (a *Agent) Pause() {
	a.Lock()
	defer a.Unlock()
	a.pause = true

}

func (a *Agent) Resume() {
	a.Lock()
	defer a.Unlock()
	a.pause = false
}

func (a *Agent) Paused() bool {
	a.Lock()
	defer a.Unlock()
	return a.pause
}

func (a *Agent) Memory() RAGDB {
	return a.options.ragdb
}

func (a *Agent) runAction(ctx context.Context, chosenAction types.Action, params types.ActionParams) (result types.ActionResult, err error) {
	for _, act := range a.availableActions() {
		if act.Definition().Name == chosenAction.Definition().Name {
			res, err := act.Run(ctx, params)
			if err != nil {
				return types.ActionResult{}, fmt.Errorf("error running action: %w", err)
			}

			result = res
		}
	}

	xlog.Info("Running action", "action", chosenAction.Definition().Name, "agent", a.Character.Name)

	if chosenAction.Definition().Name.Is(action.StateActionName) {
		// We need to store the result in the state
		state := action.AgentInternalState{}

		err = params.Unmarshal(&state)
		if err != nil {
			return types.ActionResult{}, fmt.Errorf("error unmarshalling state of the agent: %w", err)
		}
		// update the current state with the one we just got from the action
		a.currentState = &state

		// update the state file
		if a.options.statefile != "" {
			if err := a.SaveState(a.options.statefile); err != nil {
				return types.ActionResult{}, err
			}
		}
	}

	return result, nil
}

func (a *Agent) processPrompts(conversation Messages) Messages {
	//if job.Image != "" {
	// TODO: Use llava to explain the image content
	//}
	// Add custom prompts
	for _, prompt := range a.options.prompts {
		message, err := prompt.Render(a)
		if err != nil {
			xlog.Error("Error rendering prompt", "error", err)
			continue
		}
		if message == "" {
			xlog.Debug("Prompt is empty, skipping", "agent", a.Character.Name)
			continue
		}
		if !conversation.Exist(a.options.systemPrompt) {
			conversation = append([]openai.ChatCompletionMessage{
				{
					Role:    prompt.Role(),
					Content: message,
				}}, conversation...)
		}
	}

	// TODO: move to a Promptblock?
	if a.options.systemPrompt != "" {
		if !conversation.Exist(a.options.systemPrompt) {
			conversation = append([]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: a.options.systemPrompt,
				}}, conversation...)
		}
	}

	return conversation
}

func (a *Agent) describeImage(ctx context.Context, model, imageURL string) (string, error) {
	xlog.Debug("Describing image", "model", model, "image", imageURL)
	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{

					Role: "user",
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: "What is in the image?",
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{

								URL: imageURL,
							},
						},
					},
				},
			}})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}

	xlog.Debug("Described image", "description", resp.Choices[0].Message.Content)
	return resp.Choices[0].Message.Content, nil
}

func extractImageContent(message openai.ChatCompletionMessage) (imageURL, text string, e error) {
	e = fmt.Errorf("no image found")
	if message.MultiContent != nil {
		for _, content := range message.MultiContent {
			if content.Type == openai.ChatMessagePartTypeImageURL {
				imageURL = content.ImageURL.URL
				e = nil
			}
			if content.Type == openai.ChatMessagePartTypeText {
				text = content.Text
				e = nil
			}
		}
	}
	return
}

func (a *Agent) processUserInputs(job *types.Job, role string, conv Messages) Messages {

	// walk conversation history, and check if last message from user contains image.
	// If it does, we need to describe the image first with a model that supports image understanding (if the current model doesn't support it)
	// and add it to the conversation context
	if !a.options.SeparatedMultimodalModel() {
		return conv
	}
	lastUserMessage := conv.GetLatestUserMessage()
	if lastUserMessage != nil && conv.IsLastMessageFromRole(UserRole) {
		imageURL, text, err := extractImageContent(*lastUserMessage)
		if err == nil {
			// We have an image, we need to describe it first
			// and add it to the conversation context
			imageDescription, err := a.describeImage(a.context.Context, a.options.LLMAPI.MultimodalModel, imageURL)
			if err != nil {
				xlog.Error("Error describing image", "error", err)
			} else {
				// We replace the user message with the image description
				// and add the user text to the conversation
				explainerMessage := openai.ChatCompletionMessage{
					Role:    "system",
					Content: fmt.Sprintf("The user shared an image which can be described as: %s", imageDescription),
				}

				// remove lastUserMessage from the conversation
				conv = conv.RemoveLastUserMessage()
				conv = append(conv, explainerMessage)
				conv = append(conv, openai.ChatCompletionMessage{
					Role:    role,
					Content: text,
				})
			}
		}
	}

	return conv
}

func (a *Agent) consumeJob(job *types.Job, role string) {

	if err := job.GetContext().Err(); err != nil {
		job.Result.Finish(fmt.Errorf("expired"))
		return
	}

	a.Lock()
	paused := a.pause
	a.Unlock()

	if paused {
		xlog.Info("Agent is paused, skipping job", "agent", a.Character.Name)
		job.Result.Finish(fmt.Errorf("agent is paused"))
		return
	}

	// We are self evaluating if we consume the job as a system role
	selfEvaluation := role == SystemRole

	conv := job.ConversationHistory

	a.Lock()
	a.selfEvaluationInProgress = selfEvaluation
	a.Unlock()
	defer job.Cancel()

	if selfEvaluation {
		defer func() {
			a.Lock()
			a.selfEvaluationInProgress = false
			a.Unlock()
		}()
	}

	conv = a.processPrompts(conv)
	conv = a.processUserInputs(job, role, conv)

	// RAG
	a.knowledgeBaseLookup(conv)

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
	var chosenAction types.Action
	var reasoning string
	var actionParams types.ActionParams

	if job.HasNextAction() {
		// if we are being re-evaluated, we already have the action
		// and the reasoning. Consume it here and reset it
		action, params, reason := job.GetNextAction()
		chosenAction = *action
		reasoning = reason
		if params == nil {
			p, err := a.generateParameters(job.GetContext(), pickTemplate, chosenAction, conv, reasoning, maxRetries)
			if err != nil {
				xlog.Error("Error generating parameters, trying again", "error", err)
				// try again
				job.SetNextAction(&chosenAction, nil, reasoning)
				a.consumeJob(job, role)
				return
			}
			actionParams = p.actionParams
		} else {
			actionParams = *params
		}
		job.ResetNextAction()
	} else {
		var err error
		chosenAction, actionParams, reasoning, err = a.pickAction(job.GetContext(), pickTemplate, conv, maxRetries)
		if err != nil {
			xlog.Error("Error picking action", "error", err)
			job.Result.Finish(err)
			return
		}
	}

	//xlog.Debug("Picked action", "agent", a.Character.Name, "action", chosenAction.Definition().Name, "reasoning", reasoning)
	if chosenAction == nil {
		// If no action was picked up, the reasoning is the message returned by the assistant
		// so we can consume it as if it was a reply.
		//job.Result.SetResult(ActionState{ActionCurrentState{nil, nil, "No action to do, just reply"}, ""})
		//job.Result.Finish(fmt.Errorf("no action to do"))\
		xlog.Info("No action to do, just reply", "agent", a.Character.Name, "reasoning", reasoning)

		conv = append(conv, openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: reasoning,
		})

		xlog.Debug("Finish job with reasoning", "reasoning", reasoning, "agent", a.Character.Name, "conversation", fmt.Sprintf("%+v", conv))
		job.Result.Conversation = conv
		job.Result.AddFinalizer(func(conv []openai.ChatCompletionMessage) {
			a.saveCurrentConversation(conv)
		})
		job.Result.SetResponse(reasoning)
		job.Result.Finish(nil)
		return
	}

	if chosenAction.Definition().Name.Is(action.StopActionName) {
		xlog.Info("LLM decided to stop")
		job.Result.Finish(nil)
		return
	}

	// if we force a reasoning, we need to generate the parameters
	if a.options.forceReasoning || actionParams == nil {
		xlog.Info("Generating parameters",
			"agent", a.Character.Name,
			"action", chosenAction.Definition().Name,
			"reasoning", reasoning,
		)

		params, err := a.generateParameters(job.GetContext(), pickTemplate, chosenAction, conv, reasoning, maxRetries)
		if err != nil {
			xlog.Error("Error generating parameters, trying again", "error", err)
			// try again
			job.SetNextAction(&chosenAction, nil, reasoning)
			a.consumeJob(job, role)
			return
		}
		actionParams = params.actionParams
	}

	xlog.Info(
		"Generated parameters",
		"agent", a.Character.Name,
		"action", chosenAction.Definition().Name,
		"reasoning", reasoning,
		"params", actionParams.String(),
	)

	if actionParams == nil {
		job.Result.Finish(fmt.Errorf("no parameters"))
		xlog.Error("No parameters", "agent", a.Character.Name)
		return
	}

	var err error
	conv, err = a.handlePlanning(job.GetContext(), job, chosenAction, actionParams, reasoning, pickTemplate, conv)
	if err != nil {
		job.Result.Finish(fmt.Errorf("error running action: %w", err))
		return
	}

	if !job.Callback(types.ActionCurrentState{
		Job:       job,
		Action:    chosenAction,
		Params:    actionParams,
		Reasoning: reasoning}) {
		job.Result.SetResult(types.ActionState{
			ActionCurrentState: types.ActionCurrentState{
				Job:       job,
				Action:    chosenAction,
				Params:    actionParams,
				Reasoning: reasoning,
			},
			ActionResult: types.ActionResult{Result: "stopped by callback"}})
		job.Result.Conversation = conv
		job.Result.Finish(nil)
		return
	}

	if selfEvaluation && a.options.initiateConversations &&
		chosenAction.Definition().Name.Is(action.ConversationActionName) {

		xlog.Info("LLM decided to initiate a new conversation", "agent", a.Character.Name)

		message := action.ConversationActionResponse{}
		if err := actionParams.Unmarshal(&message); err != nil {
			xlog.Error("Error unmarshalling conversation response", "error", err)
			job.Result.Finish(fmt.Errorf("error unmarshalling conversation response: %w", err))
			return
		}

		msg := openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: message.Message,
		}

		go func(agent *Agent) {
			xlog.Info("Sending new conversation to channel", "agent", agent.Character.Name, "message", msg.Content)
			agent.newConversations <- msg
		}(a)

		job.Result.Conversation = []openai.ChatCompletionMessage{
			msg,
		}
		job.Result.SetResponse("decided to initiate a new conversation")
		job.Result.Finish(nil)
		return
	}

	// If we don't have to reply , run the action!
	if !chosenAction.Definition().Name.Is(action.ReplyActionName) {

		if !chosenAction.Definition().Name.Is(action.PlanActionName) {
			result, err := a.runAction(job.GetContext(), chosenAction, actionParams)
			if err != nil {
				//job.Result.Finish(fmt.Errorf("error running action: %w", err))
				//return
				// make the LLM aware of the error of running the action instead of stopping the job here
				result.Result = fmt.Sprintf("Error running tool: %v", err)
			}

			stateResult := types.ActionState{
				ActionCurrentState: types.ActionCurrentState{
					Job:       job,
					Action:    chosenAction,
					Params:    actionParams,
					Reasoning: reasoning,
				},
				ActionResult: result,
			}
			job.Result.SetResult(stateResult)
			job.CallbackWithResult(stateResult)
			xlog.Debug("Action executed", "agent", a.Character.Name, "action", chosenAction.Definition().Name, "result", result)

			conv = a.addFunctionResultToConversation(chosenAction, actionParams, result, conv)
		}

		//conv = append(conv, messages...)
		//conv = messages

		// given the result, we can now ask OpenAI to complete the conversation or
		// to continue using another tool given the result
		followingAction, followingParams, reasoning, err := a.pickAction(job.GetContext(), reEvaluationTemplate, conv, maxRetries)
		if err != nil {
			job.Result.Conversation = conv
			job.Result.Finish(fmt.Errorf("error picking action: %w", err))
			return
		}

		if followingAction != nil &&
			!followingAction.Definition().Name.Is(action.ReplyActionName) &&
			!chosenAction.Definition().Name.Is(action.ReplyActionName) &&
			!followingAction.Definition().Name.Is(action.PlanActionName) {

			xlog.Info("Following action", "action", followingAction.Definition().Name, "agent", a.Character.Name)

			// We need to do another action (?)
			// The agent decided to do another action
			// call ourselves again
			job.SetNextAction(&followingAction, &followingParams, reasoning)
			a.consumeJob(job, role)
			return
		} else if followingAction == nil {
			xlog.Info("Not following another action", "agent", a.Character.Name)

			if !a.options.forceReasoning {
				xlog.Info("Finish conversation with reasoning", "reasoning", reasoning, "agent", a.Character.Name)

				msg := openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: reasoning,
				}

				conv = append(conv, msg)
				job.Result.SetResponse(msg.Content)
				job.Result.Conversation = conv
				job.Result.AddFinalizer(func(conv []openai.ChatCompletionMessage) {
					a.saveCurrentConversation(conv)
				})
				job.Result.Finish(nil)
				return
			}
		} else if followingAction.Definition().Name.Is(action.PlanActionName) {
			xlog.Debug("Following action is a plan action, skipping", "agent", a.Character.Name)
		}
	}

	job.Result.Conversation = conv

	// At this point can only be a reply action
	xlog.Info("Computing reply", "agent", a.Character.Name)

	// decode the response
	replyResponse := action.ReplyResponse{}

	if err := actionParams.Unmarshal(&replyResponse); err != nil {
		job.Result.Conversation = conv
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

	// If we have a hud, display it when answering normally
	if a.options.enableHUD {
		prompt, err := renderTemplate(hudTemplate, a.prepareHUD(), a.availableActions(), reasoning)
		if err != nil {
			job.Result.Conversation = conv
			job.Result.Finish(fmt.Errorf("error renderTemplate: %w", err))
			return
		}
		if !Messages(conv).Exist(prompt) {
			conv = append([]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: prompt,
				},
			}, conv...)
		}
	}

	// Generate a human-readable response
	// resp, err := a.client.CreateChatCompletion(ctx,
	// 	openai.ChatCompletionRequest{
	// 		Model: a.options.LLMAPI.Model,
	// 		Messages: append(conv,
	// 			openai.ChatCompletionMessage{
	// 				Role:    "system",
	// 				Content: "Assistant thought: " + replyResponse.Message,
	// 			},
	// 		),
	// 	},
	// )

	if !a.options.forceReasoning {
		xlog.Info("No reasoning, return reply message", "reply", replyResponse.Message, "agent", a.Character.Name)

		msg := openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: replyResponse.Message,
		}

		conv = append(conv, msg)
		job.Result.Conversation = conv
		job.Result.SetResponse(msg.Content)
		job.Result.AddFinalizer(func(conv []openai.ChatCompletionMessage) {
			a.saveCurrentConversation(conv)
		})
		job.Result.Finish(nil)
		return
	}

	xlog.Info("Reasoning, ask LLM for a reply", "agent", a.Character.Name)
	xlog.Debug("Conversation", "conversation", fmt.Sprintf("%+v", conv))
	msg, err := a.askLLM(job.GetContext(), conv, maxRetries)
	if err != nil {
		job.Result.Conversation = conv
		job.Result.Finish(err)
		xlog.Error("Error asking LLM for a reply", "error", err)
		return
	}

	// If we didn't got any message, we can use the response from the action
	if chosenAction.Definition().Name.Is(action.ReplyActionName) && msg.Content == "" {
		xlog.Info("No output returned from conversation, using the action response as a reply " + replyResponse.Message)

		msg = openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: replyResponse.Message,
		}
	}

	conv = append(conv, msg)
	job.Result.SetResponse(msg.Content)
	xlog.Info("Response from LLM", "response", msg.Content, "agent", a.Character.Name)
	job.Result.Conversation = conv
	job.Result.AddFinalizer(func(conv []openai.ChatCompletionMessage) {
		a.saveCurrentConversation(conv)
	})
	job.Result.Finish(nil)
}

func (a *Agent) addFunctionResultToConversation(chosenAction types.Action, actionParams types.ActionParams, result types.ActionResult, conv Messages) Messages {
	// calling the function
	conv = append(conv, openai.ChatCompletionMessage{
		Role: "assistant",
		ToolCalls: []openai.ToolCall{
			{
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      chosenAction.Definition().Name.String(),
					Arguments: actionParams.String(),
				},
			},
		},
	})

	// result of calling the function
	conv = append(conv, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result.Result,
		Name:       chosenAction.Definition().Name.String(),
		ToolCallID: chosenAction.Definition().Name.String(),
	})

	return conv
}

// This is running in the background.
func (a *Agent) periodicallyRun(timer *time.Timer) {
	// Remember always to reset the timer - if we don't the agent will stop..
	defer timer.Reset(a.options.periodicRuns)

	xlog.Debug("Agent is running periodically", "agent", a.Character.Name)

	// TODO: Would be nice if we have a special action to
	// contact the user. This would actually make sure that
	// if the agent wants to initiate a conversation, it can do so.
	// This would be a special action that would be picked up by the agent
	// and would be used to contact the user.

	// if len(conv()) != 0 {
	// 	// Here the LLM could decide to store some part of the conversation too in the memory
	// 	evaluateMemory := NewJob(
	// 		WithText(
	// 			`Evaluate the current conversation and decide if we need to store some relevant informations from it`,
	// 		),
	// 		WithReasoningCallback(a.options.reasoningCallback),
	// 		WithResultCallback(a.options.resultCallback),
	// 	)
	// 	a.consumeJob(evaluateMemory, SystemRole)

	// 	a.ResetConversation()
	// }

	if !a.options.standaloneJob {
		return
	}
	xlog.Info("Periodically running", "agent", a.Character.Name)

	// Here we go in a loop of
	// - asking the agent to do something
	// - evaluating the result
	// - asking the agent to do something else based on the result

	//	whatNext := NewJob(WithText("Decide what to do based on the state"))
	whatNext := types.NewJob(
		types.WithText(innerMonologueTemplate),
		types.WithReasoningCallback(a.options.reasoningCallback),
		types.WithResultCallback(a.options.resultCallback),
	)
	a.consumeJob(whatNext, SystemRole)

	xlog.Info("STOP -- Periodically run is done", "agent", a.Character.Name)

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

func (a *Agent) Run() error {

	a.startNewConversationsConsumer()
	xlog.Debug("Agent is now running", "agent", a.Character.Name)
	// The agent run does two things:
	// picks up requests from a queue
	// and generates a response/perform actions

	// It is also preemptive.
	// That is, it can interrupt the current action
	// if another one comes in.

	// If there is no action, periodically evaluate if it has to do something on its own.

	// Expose a REST API to interact with the agent to ask it things

	//todoTimer := time.NewTicker(a.options.periodicRuns)
	timer := time.NewTimer(a.options.periodicRuns)
	for {
		xlog.Debug("Agent is now waiting for a new job", "agent", a.Character.Name)
		select {
		case job := <-a.jobQueue:
			a.loop(timer, job)
		case <-a.context.Done():
			// Agent has been canceled, return error
			xlog.Warn("Agent has been canceled", "agent", a.Character.Name)
			return ErrContextCanceled
		case <-timer.C:
			a.periodicallyRun(timer)
		}
	}
}

func (a *Agent) loop(timer *time.Timer, job *types.Job) {
	// Remember always to reset the timer - if we don't the agent will stop..
	defer timer.Reset(a.options.periodicRuns)
	// Consume the job and generate a response
	// TODO: Give a short-term memory to the agent
	// stop and drain the timer
	if !timer.Stop() {
		<-timer.C
	}
	xlog.Debug("Agent is consuming a job", "agent", a.Character.Name, "job", job)
	a.consumeJob(job, UserRole)
}
