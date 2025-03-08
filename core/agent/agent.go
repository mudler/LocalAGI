package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mudler/LocalAgent/pkg/xlog"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/pkg/llm"
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
	currentState             *action.AgentInternalState
	nextAction               Action
	nextActionParams         *action.ActionParams
	currentConversation      Messages
	selfEvaluationInProgress bool
	pause                    bool

	newConversations chan openai.ChatCompletionMessage
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
		jobQueue:     make(chan *Job),
		options:      options,
		client:       client,
		Character:    options.character,
		currentState: &action.AgentInternalState{},
		context:      action.NewContext(ctx, cancel),
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

	xlog.Info(
		"Agent created",
		"agent", a.Character.Name,
		"character", a.Character.String(),
		"state", a.State().String(),
		"goal", a.options.permanentGoal,
	)

	return a, nil
}

// StopAction stops the current action
// if any. Can be called before adding a new job.
func (a *Agent) StopAction() {
	a.Lock()
	defer a.Unlock()
	if a.actionContext != nil {
		xlog.Debug("Stopping current action", "agent", a.Character.Name)
		a.actionContext.Cancel()
	}
}

func (a *Agent) Context() context.Context {
	return a.context.Context
}

func (a *Agent) ActionContext() context.Context {
	return a.actionContext.Context
}

func (a *Agent) ConversationChannel() chan openai.ChatCompletionMessage {
	return a.newConversations
}

// Ask is a pre-emptive, blocking call that returns the response as soon as it's ready.
// It discards any other computation.
func (a *Agent) Ask(opts ...JobOption) *JobResult {
	xlog.Debug("Agent Ask()", "agent", a.Character.Name)
	defer func() {
		xlog.Debug("Agent has finished being asked", "agent", a.Character.Name)
	}()

	//a.StopAction()
	j := NewJob(
		append(
			opts,
			WithReasoningCallback(a.options.reasoningCallback),
			WithResultCallback(a.options.resultCallback),
		)...,
	)
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

func (a *Agent) askLLM(ctx context.Context, conversation []openai.ChatCompletionMessage) (openai.ChatCompletionMessage, error) {
	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    a.options.LLMAPI.Model,
			Messages: conversation,
		},
	)
	if err != nil {
		return openai.ChatCompletionMessage{}, err
	}

	if len(resp.Choices) != 1 {
		return openai.ChatCompletionMessage{}, fmt.Errorf("no enough choices: %w", err)
	}

	return resp.Choices[0].Message, nil
}

func (a *Agent) ResetConversation() {
	a.Lock()
	defer a.Unlock()

	xlog.Info("Resetting conversation", "agent", a.Character.Name)

	// store into memory the conversation before pruning it
	// TODO: Shall we summarize the conversation into a bullet list of highlights
	// using the LLM instead?
	a.saveCurrentConversation()

	a.currentConversation = []openai.ChatCompletionMessage{}
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	a.Lock()
	defer a.Unlock()
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

func (a *Agent) runAction(chosenAction Action, params action.ActionParams) (result action.ActionResult, err error) {
	for _, act := range a.systemInternalActions() {
		if act.Definition().Name == chosenAction.Definition().Name {
			res, err := act.Run(a.actionContext, params)
			if err != nil {
				return action.ActionResult{}, fmt.Errorf("error running action: %w", err)
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
			return action.ActionResult{}, fmt.Errorf("error unmarshalling state of the agent: %w", err)
		}
		// update the current state with the one we just got from the action
		a.currentState = &state

		// update the state file
		if a.options.statefile != "" {
			if err := a.SaveState(a.options.statefile); err != nil {
				return action.ActionResult{}, err
			}
		}
	}

	return result, nil
}

func (a *Agent) processPrompts() {
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
}

func (a *Agent) describeImage(ctx context.Context, model, imageURL string) (string, error) {
	resp, err := a.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model: model, Messages: []openai.ChatCompletionMessage{
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

func (a *Agent) processUserInputs(job *Job, role string) {

	noNewMessage := job.Text == "" && job.Image == ""
	onlyText := job.Text != "" && job.Image == ""

	// walk conversation history, and check if last message from user contains image.
	// If it does, we need to describe the image first with a model that supports image understanding (if the current model doesn't support it)
	// and add it to the conversation context
	if a.options.SeparatedMultimodalModel() && noNewMessage {
		lastUserMessage := a.currentConversation.GetLatestUserMessage()
		if lastUserMessage != nil {
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
					lastUserMessage.Content = fmt.Sprintf("The user shared an image which can be described as: %s", imageDescription)
					lastUserMessage.MultiContent = nil
					lastUserMessage.Role = "system"
					a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
						Role:    role,
						Content: text,
					})
				}
			}
		}

	}

	if onlyText {
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role:    role,
			Content: job.Text,
		})
	}

	if job.Image != "" {
		// If an image is present with the text
		// we have two cases: if the model supports both images and text, we can send both
		// if the model supports only text, we can send the text only and we need to describe the image first with a model that support image understanding and add it to the conversation context
		if a.options.SeparatedMultimodalModel() {
			// We need to describe the image first
			imageDescription, err := a.describeImage(a.context.Context, a.options.LLMAPI.Model, job.Image)
			if err != nil {
				xlog.Error("Error describing image", "error", err)
			} else {
				a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
					Role:    "system",
					Content: fmt.Sprintf("The user shared an image which can be described as: %s", imageDescription),
				})
				a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
					Role:    role,
					Content: job.Text,
				})
			}
		} else {
			// Just append to the message both the image and the text
			a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
				Role: role,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: job.Text,
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: job.Image,
						},
					},
				},
			})
		}
	}
}

func (a *Agent) consumeJob(job *Job, role string) {
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

	a.Lock()
	// Set the action context
	ctx, cancel := context.WithCancel(context.Background())
	a.actionContext = action.NewContext(ctx, cancel)
	a.selfEvaluationInProgress = selfEvaluation
	if len(job.conversationHistory) != 0 {
		a.currentConversation = job.conversationHistory
	}
	a.Unlock()

	defer func() {
		a.Lock()
		if a.actionContext != nil {
			a.actionContext.Cancel()
			a.actionContext = nil
		}
		a.Unlock()
	}()

	if selfEvaluation {
		defer func() {
			a.Lock()
			a.selfEvaluationInProgress = false
			a.Unlock()
		}()
	}

	a.processPrompts()
	a.processUserInputs(job, role)

	// RAG
	a.knowledgeBaseLookup()

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
	var actionParams action.ActionParams

	if a.nextAction != nil {
		// if we are being re-evaluated, we already have the action
		// and the reasoning. Consume it here and reset it
		chosenAction = a.nextAction
		reasoning = a.currentReasoning
		actionParams = *a.nextActionParams
		a.currentReasoning = ""
		a.nextActionParams = nil
		a.nextAction = nil
	} else {
		var err error
		chosenAction, actionParams, reasoning, err = a.pickAction(ctx, pickTemplate, a.currentConversation)
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

		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: reasoning,
		})
		job.Result.Conversation = a.currentConversation
		a.saveCurrentConversation()
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

		params, err := a.generateParameters(ctx, pickTemplate, chosenAction, a.currentConversation, reasoning)
		if err != nil {
			job.Result.Finish(fmt.Errorf("error generating action's parameters: %w", err))
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
		return
	}

	if !job.Callback(ActionCurrentState{chosenAction, actionParams, reasoning}) {
		job.Result.SetResult(ActionState{ActionCurrentState{chosenAction, actionParams, reasoning}, action.ActionResult{Result: "stopped by callback"}})
		job.Result.Conversation = a.currentConversation
		job.Result.Finish(nil)
		return
	}

	if selfEvaluation && a.options.initiateConversations &&
		chosenAction.Definition().Name.Is(action.ConversationActionName) {

		message := action.ConversationActionResponse{}
		if err := actionParams.Unmarshal(&message); err != nil {
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
		job.Result.Conversation = a.currentConversation
		job.Result.SetResponse("decided to initiate a new conversation")
		job.Result.Finish(nil)
		return
	}

	// If we don't have to reply , run the action!
	if !chosenAction.Definition().Name.Is(action.ReplyActionName) {
		result, err := a.runAction(chosenAction, actionParams)
		if err != nil {
			//job.Result.Finish(fmt.Errorf("error running action: %w", err))
			//return
			// make the LLM aware of the error of running the action instead of stopping the job here
			result.Result = fmt.Sprintf("Error running tool: %v", err)
		}

		stateResult := ActionState{ActionCurrentState{chosenAction, actionParams, reasoning}, result}
		job.Result.SetResult(stateResult)
		job.CallbackWithResult(stateResult)
		xlog.Debug("Action executed", "agent", a.Character.Name, "action", chosenAction.Definition().Name, "result", result)

		// calling the function
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
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
		a.currentConversation = append(a.currentConversation, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result.Result,
			Name:       chosenAction.Definition().Name.String(),
			ToolCallID: chosenAction.Definition().Name.String(),
		})

		//a.currentConversation = append(a.currentConversation, messages...)
		//a.currentConversation = messages

		// given the result, we can now ask OpenAI to complete the conversation or
		// to continue using another tool given the result
		followingAction, followingParams, reasoning, err := a.pickAction(ctx, reEvaluationTemplate, a.currentConversation)
		if err != nil {
			job.Result.Conversation = a.currentConversation
			job.Result.Finish(fmt.Errorf("error picking action: %w", err))
			return
		}

		if followingAction != nil &&
			!followingAction.Definition().Name.Is(action.ReplyActionName) &&
			!chosenAction.Definition().Name.Is(action.ReplyActionName) {
			xlog.Info("Following action", "action", followingAction.Definition().Name, "agent", a.Character.Name)

			// We need to do another action (?)
			// The agent decided to do another action
			// call ourselves again
			a.currentReasoning = reasoning
			a.nextAction = followingAction
			a.nextActionParams = &followingParams
			job.Text = ""
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

				a.currentConversation = append(a.currentConversation, msg)
				a.saveCurrentConversation()
				job.Result.SetResponse(msg.Content)
				job.Result.Conversation = a.currentConversation
				job.Result.Finish(nil)
				return
			}
		}
	}

	job.Result.Conversation = a.currentConversation

	// At this point can only be a reply action
	xlog.Info("Computing reply", "agent", a.Character.Name)

	// decode the response
	replyResponse := action.ReplyResponse{}

	if err := actionParams.Unmarshal(&replyResponse); err != nil {
		job.Result.Conversation = a.currentConversation
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
		prompt, err := renderTemplate(hudTemplate, a.prepareHUD(), a.systemInternalActions(), reasoning)
		if err != nil {
			job.Result.Conversation = a.currentConversation
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

	if !a.options.forceReasoning {
		xlog.Info("No reasoning, return reply message", "reply", replyResponse.Message, "agent", a.Character.Name)

		msg := openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: replyResponse.Message,
		}

		a.currentConversation = append(a.currentConversation, msg)
		job.Result.Conversation = a.currentConversation
		job.Result.SetResponse(msg.Content)
		a.saveCurrentConversation()
		job.Result.Finish(nil)
		return
	}

	xlog.Info("Reasoning, ask LLM for a reply", "agent", a.Character.Name)
	xlog.Debug("Conversation", "conversation", fmt.Sprintf("%+v", a.currentConversation))
	msg, err := a.askLLM(ctx, a.currentConversation)
	if err != nil {
		job.Result.Conversation = a.currentConversation
		job.Result.Finish(err)
		xlog.Error("Error asking LLM for a reply", "error", err)
		return
	}

	// If we didn't got any message, we can use the response from the action
	if chosenAction.Definition().Name.Is(action.ReplyActionName) && msg.Content == "" ||
		strings.Contains(msg.Content, "<tool_call>") {
		xlog.Info("No output returned from conversation, using the action response as a reply " + replyResponse.Message)

		msg = openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: replyResponse.Message,
		}
	}

	a.currentConversation = append(a.currentConversation, msg)
	job.Result.SetResponse(msg.Content)
	xlog.Info("Response from LLM", "response", msg.Content, "agent", a.Character.Name)
	job.Result.Conversation = a.currentConversation
	a.saveCurrentConversation()
	job.Result.Finish(nil)
}

// This is running in the background.
func (a *Agent) periodicallyRun(timer *time.Timer) {
	// Remember always to reset the timer - if we don't the agent will stop..
	defer timer.Reset(a.options.periodicRuns)

	a.StopAction()
	xlog.Debug("Agent is running periodically", "agent", a.Character.Name)

	// TODO: Would be nice if we have a special action to
	// contact the user. This would actually make sure that
	// if the agent wants to initiate a conversation, it can do so.
	// This would be a special action that would be picked up by the agent
	// and would be used to contact the user.

	xlog.Info("START -- Periodically run is starting")

	// if len(a.CurrentConversation()) != 0 {
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
		a.ResetConversation()

		return
	}

	// Here we go in a loop of
	// - asking the agent to do something
	// - evaluating the result
	// - asking the agent to do something else based on the result

	//	whatNext := NewJob(WithText("Decide what to do based on the state"))
	whatNext := NewJob(
		WithText(innerMonologueTemplate),
		WithReasoningCallback(a.options.reasoningCallback),
		WithResultCallback(a.options.resultCallback),
	)
	a.consumeJob(whatNext, SystemRole)
	a.ResetConversation()

	xlog.Info("STOP -- Periodically run is done")

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
		xlog.Debug("Agent is waiting for a job", "agent", a.Character.Name)
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

func (a *Agent) loop(timer *time.Timer, job *Job) {
	// Remember always to reset the timer - if we don't the agent will stop..
	defer timer.Reset(a.options.periodicRuns)
	// Consume the job and generate a response
	// TODO: Give a short-term memory to the agent
	// stop and drain the timer
	if !timer.Stop() {
		<-timer.C
	}
	xlog.Debug("Agent is consuming a job", "agent", a.Character.Name, "job", job)
	a.StopAction()
	a.consumeJob(job, UserRole)
}
