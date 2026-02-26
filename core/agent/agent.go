package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mudler/cogito"
	"github.com/mudler/cogito/clients"

	"github.com/mudler/xlog"

	"github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/scheduler"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/sashabaranov/go-openai"
)

const (
	UserRole      = "user"
	AssistantRole = "assistant"
	SystemRole    = "system"
)

// NoToolToCallArgs defines the arguments for the no_tool_to_call sink state tool
type NoToolToCallArgs struct {
	Reasoning string `json:"reasoning" description:"The reasoning for why no tool is being called"`
}

// NoToolToCallTool is a custom sink state tool that logs when no other tool is needed
type NoToolToCallTool struct{}

// Run executes the no_tool_to_call tool and logs a message
func (t NoToolToCallTool) Run(args NoToolToCallArgs) (string, any, error) {
	xlog.Info("No tool to call - agent decided no action was needed", "reasoning", args.Reasoning)
	return fmt.Sprintf("No action needed: %s", args.Reasoning), nil, nil
}

type Agent struct {
	sync.Mutex
	options   *options
	Character Character
	client    *openai.Client
	jobQueue  chan *types.Job
	context   *types.ActionContext

	currentState *types.AgentInternalState

	selfEvaluationInProgress bool
	pause                    bool

	newConversations chan *types.ConversationMessage

	mcpSessions []*mcp.ClientSession
	// only contains the MCP action definitions for observables
	mcpActionDefinitions types.Actions

	subscriberMutex        sync.Mutex
	newMessagesSubscribers []func(*types.ConversationMessage)

	observer Observer

	llm         cogito.LLM
	sharedState *types.AgentSharedState

	// Task scheduler for managing reminders
	taskScheduler *scheduler.Scheduler

	// currentJobByConversation tracks the running job per conversation_id for cancel-previous-on-new-message
	currentJobByConversation map[string]*types.Job
	currentJobMu             sync.Mutex
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
	llmClient := clients.NewLocalAILLM(options.LLMAPI.Model, options.LLMAPI.APIKey, options.LLMAPI.APIURL)
	c := context.Background()
	if options.context != nil {
		c = options.context
	}

	ctx, cancel := context.WithCancel(c)
	a := &Agent{
		jobQueue:                 make(chan *types.Job),
		options:                  options,
		client:                   client,
		Character:                options.character,
		currentState:             &types.AgentInternalState{},
		llm:                      llmClient,
		context:                  types.NewActionContext(ctx, cancel),
		newConversations:         make(chan *types.ConversationMessage),
		newMessagesSubscribers:   options.newConversationsSubscribers,
		sharedState:              types.NewAgentSharedState(options.lastMessageDuration),
		currentJobByConversation: make(map[string]*types.Job),
	}

	// Initialize observer if provided
	if options.observer != nil {
		a.observer = options.observer
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

	// Initialize task scheduler for reminders
	schedulerPath := options.schedulerStorePath
	if schedulerPath == "" {
		schedulerPath = "scheduled_tasks.json"
	}

	store, err := scheduler.NewJSONStore(schedulerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler store: %v", err)
	}

	executor := &agentSchedulerExecutor{agent: a}
	pollInterval := options.schedulerPollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	a.taskScheduler = scheduler.NewScheduler(store, executor, pollInterval)
	a.sharedState.Scheduler = a.taskScheduler
	a.sharedState.AgentName = a.Character.Name
	xlog.Info("Task scheduler initialized", "store_path", schedulerPath, "poll_interval", pollInterval)

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

func (a *Agent) SharedState() *types.AgentSharedState {
	return a.sharedState
}

func (a *Agent) startNewConversationsConsumer() {
	go func() {
		for {
			select {
			case <-a.context.Done():
				return

			case msg := <-a.newConversations:
				xlog.Debug("New conversation", "agent", a.Character.Name, "message", msg.Message.Content)
				a.subscriberMutex.Lock()
				subs := a.newMessagesSubscribers
				a.subscriberMutex.Unlock()
				for _, s := range subs {
					if s != nil && msg != nil {
						s(msg)
					}
				}
			}
		}
	}()
}

func (a *Agent) AddSubscriber(f func(*types.ConversationMessage)) {
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

	if a.observer != nil {
		obs := a.observer.NewObservable()
		obs.Name = "job"
		obs.Icon = "plug"
		a.observer.Update(*obs)
		opts = append(opts, types.WithObservable(obs))
	}

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

	if j.Obs != nil && a.observer != nil {
		if len(j.ConversationHistory) > 0 {
			m := j.ConversationHistory[len(j.ConversationHistory)-1]
			j.Obs.Creation = &types.Creation{ChatCompletionMessage: &m}
			a.observer.Update(*j.Obs)
		}

		j.Result.AddFinalizer(func(ccm []openai.ChatCompletionMessage) {
			if a.observer == nil {
				return
			}
			// Merge into existing Completion so last-progress completion data is preserved
			if j.Obs.Completion == nil {
				j.Obs.Completion = &types.Completion{}
			}
			j.Obs.Completion.Conversation = ccm
			if j.Result.Error != nil {
				j.Obs.Completion.Error = j.Result.Error.Error()
			}
			a.observer.Update(*j.Obs)
		})
	}

	a.Enqueue(j)
	result, err := j.Result.WaitResult(a.context.Context)
	if err != nil {
		return nil
	}
	return result
}

func (a *Agent) Enqueue(j *types.Job) {
	j.ReasoningCallback = a.options.reasoningCallback
	j.ResultCallback = a.options.resultCallback

	// Cancel previous running job for this conversation if option is enabled
	cancelPrevious := a.options.cancelPreviousOnNewMessage == nil || *a.options.cancelPreviousOnNewMessage
	if cancelPrevious && j.Metadata != nil {
		if convID, ok := j.Metadata[types.MetadataKeyConversationID].(string); ok && convID != "" {
			a.currentJobMu.Lock()
			existing := a.currentJobByConversation[convID]
			a.currentJobMu.Unlock()
			if existing != nil {
				existing.Cancel()
			}
		}
	}

	a.jobQueue <- j
}

func (a *Agent) Transcribe(ctx context.Context, file string) (string, error) {
	resp, err := a.client.CreateTranscription(ctx,
		openai.AudioRequest{
			Model:    a.options.LLMAPI.TranscriptionModel,
			Language: a.options.LLMAPI.TranscriptionLanguage,
			FilePath: file,
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

func (a *Agent) TTS(ctx context.Context, text string) ([]byte, error) {
	if a.options.LLMAPI.TTSModel == "" {
		return nil, fmt.Errorf("TTS model is not set")
	}
	resp, err := a.client.CreateSpeech(ctx,
		openai.CreateSpeechRequest{
			Model:          openai.SpeechModel(a.options.LLMAPI.TTSModel),
			Input:          text,
			ResponseFormat: openai.SpeechResponseFormatMp3,
		},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, resp)

	return buf.Bytes(), nil
}

var ErrContextCanceled = fmt.Errorf("context canceled")

func (a *Agent) Stop() {
	xlog.Debug("Stopping agent", "agent", a.Character.Name)

	// Stop the scheduler
	a.taskScheduler.Stop()
	xlog.Info("Task scheduler stopped")

	a.Lock()
	defer a.Unlock()

	a.closeMCPServers()
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

func (a *Agent) processPrompts(ctx context.Context, conversation Messages) Messages {
	// Add custom prompts
	for _, prompt := range a.options.prompts {
		message, err := prompt.Render(a)
		if err != nil {
			xlog.Error("Error rendering prompt", "error", err)
			continue
		}
		if message.Content == "" && message.ImageBase64 == "" {
			xlog.Debug("Prompt is empty, skipping", "agent", a.Character.Name)
			continue
		}

		content := message.Content

		if strings.Contains(content, "{{") {
			promptTemplate, err := templateBase("template", content)
			if err != nil {
				xlog.Error("Error rendering template", "error", err)
			}

			content, err = templateExecute(promptTemplate, CommonTemplateData{AgentName: a.Character.Name})
			if err != nil {
				xlog.Error("Error executing template", "error", err)
				content = message.Content
			}
		}

		if message.ImageBase64 != "" {
			// iF model support both images and text, process it as a single multicontent message and return
			if !a.options.SeparatedMultimodalModel() {
				conversation = append([]openai.ChatCompletionMessage{
					{
						Role: prompt.Role(),
						MultiContent: []openai.ChatMessagePart{
							{
								Type: openai.ChatMessagePartTypeText,
								Text: content,
							},
							{
								Type: openai.ChatMessagePartTypeImageURL,
								ImageURL: &openai.ChatMessageImageURL{
									URL: message.ImageBase64,
								},
							},
						},
					}}, conversation...)

			} else {
				// We need to describe the image first, and we will process the text separately (we do not return here)
				imageDescription, err := a.describeImage(ctx, a.options.LLMAPI.MultimodalModel, message.ImageBase64)
				if err != nil {
					xlog.Error("Error describing image", "error", err)
				} else {
					conversation = append([]openai.ChatCompletionMessage{
						{
							Role:    prompt.Role(),
							Content: fmt.Sprintf("%s\n\nImage description: %s", content, imageDescription),
						}}, conversation...)
				}
			}
		} else {
			conversation = append([]openai.ChatCompletionMessage{
				{
					Role:    prompt.Role(),
					Content: content,
				}}, conversation...)
		}
	}

	// TODO: move to a Promptblock?
	if a.options.systemPrompt != "" {
		content := a.options.systemPrompt

		if strings.Contains(content, "{{") {
			promptTemplate, err := templateBase("template", a.options.systemPrompt)
			if err != nil {
				xlog.Error("Error rendering template", "error", err)
			}

			content, err = templateExecute(promptTemplate, CommonTemplateData{AgentName: a.Character.Name})
			if err != nil {
				xlog.Error("Error executing template", "error", err)
				content = a.options.systemPrompt
			}
		}

		if !conversation.Exist(content) {
			conversation = append([]openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: content,
				}}, conversation...)
		}
	}

	return conversation
}

func (a *Agent) describeImage(ctx context.Context, model, imageURL string) (string, error) {
	xlog.Debug("Describing image", "model", model)
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

// extractAllImageContent extracts all images from a message
func extractAllImageContent(message openai.ChatCompletionMessage) (images []string, text string, e error) {
	e = fmt.Errorf("no image found")
	if message.MultiContent != nil {
		for _, content := range message.MultiContent {
			if content.Type == openai.ChatMessagePartTypeImageURL {
				images = append(images, content.ImageURL.URL)
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

func (a *Agent) processUserInputs(conv Messages) Messages {

	// walk conversation history, and check if any message contains images.
	// If they do, we need to describe the images first with a model that supports image understanding (if the current model doesn't support it)
	// and add them to the conversation context
	if !a.options.SeparatedMultimodalModel() {
		return conv
	}

	xlog.Debug("Processing user inputs", "agent", a.Character.Name, "conversation", conv)

	// Process all messages in the conversation to extract and describe images
	var processedMessages Messages
	var messagesToRemove []int

	for i, message := range conv {
		images, text, err := extractAllImageContent(message)
		if err == nil && len(images) > 0 {
			xlog.Debug("Found images in message", "messageIndex", i, "imageCount", len(images), "role", message.Role)

			// Mark this message for removal
			messagesToRemove = append(messagesToRemove, i)

			// Process each image in the message
			var imageDescriptions []string
			for j, image := range images {
				imageDescription, err := a.describeImage(a.context.Context, a.options.LLMAPI.MultimodalModel, image)
				if err != nil {
					xlog.Error("Error describing image", "error", err, "messageIndex", i, "imageIndex", j)
					imageDescriptions = append(imageDescriptions, fmt.Sprintf("Image %d: [Error describing image: %v]", j+1, err))
				} else {
					imageDescriptions = append(imageDescriptions, fmt.Sprintf("Image %d: %s", j+1, imageDescription))
				}
			}

			// Add the text content as a new message with the same role first
			if text != "" {
				textMessage := openai.ChatCompletionMessage{
					Role:    message.Role,
					Content: text,
				}
				processedMessages = append(processedMessages, textMessage)

				// Add the image descriptions as a system message after the text
				explainerMessage := openai.ChatCompletionMessage{
					Role: "system",
					Content: fmt.Sprintf("The above message also contains %d image(s) which can be described as: %s",
						len(images), strings.Join(imageDescriptions, "; ")),
				}
				processedMessages = append(processedMessages, explainerMessage)
			} else {
				// If there's no text, just add the image descriptions as a system message
				explainerMessage := openai.ChatCompletionMessage{
					Role: "system",
					Content: fmt.Sprintf("Message contains %d image(s) which can be described as: %s",
						len(images), strings.Join(imageDescriptions, "; ")),
				}
				processedMessages = append(processedMessages, explainerMessage)
			}
		} else {
			// No image found, keep the original message
			processedMessages = append(processedMessages, message)
		}
	}

	// If we found and processed any images, replace the conversation
	if len(messagesToRemove) > 0 {
		xlog.Info("Processed images in conversation", "messagesWithImages", len(messagesToRemove), "agent", a.Character.Name)
		return processedMessages
	}

	return conv
}

func (a *Agent) filterJob(job *types.Job) (ok bool, err error) {
	hasTriggers := false
	triggeredBy := ""
	failedBy := ""

	if job.DoneFilter {
		return true, nil
	}
	job.DoneFilter = true

	if len(a.options.jobFilters) < 1 {
		xlog.Debug("No filters")
		return true, nil
	}

	for _, filter := range a.options.jobFilters {
		name := filter.Name()
		if triggeredBy != "" && filter.IsTrigger() {
			continue
		}

		ok, err = filter.Apply(job)
		if err != nil {
			xlog.Error("Error in job filter", "filter", name, "error", err)
			failedBy = name
			break
		}

		if filter.IsTrigger() {
			hasTriggers = true
			if ok {
				triggeredBy = name
				xlog.Info("Job triggered by filter", "filter", name)
			}
		} else if !ok {
			failedBy = name
			xlog.Info("Job failed filter", "filter", name)
			break
		} else {
			xlog.Debug("Job passed filter", "filter", name)
		}
	}

	if a.Observer() != nil && job.Obs != nil {
		obs := a.Observer().NewObservable()
		obs.Name = "filter"
		obs.Icon = "shield"
		obs.ParentID = job.Obs.ID
		if err == nil {
			obs.Completion = &types.Completion{
				FilterResult: &types.FilterResult{
					HasTriggers: hasTriggers,
					TriggeredBy: triggeredBy,
					FailedBy:    failedBy,
				},
			}
		} else {
			obs.Completion = &types.Completion{
				Error: err.Error(),
			}
		}
		a.Observer().Update(*obs)
	}

	return failedBy == "" && (!hasTriggers || triggeredBy != ""), nil
}

// replyWithToolCall handles user-defined actions by recording the action state without setting Response
func (a *Agent) replyWithToolCall(job *types.Job, conv []openai.ChatCompletionMessage, params types.ActionParams, chosenAction types.Action, reasoning string) {
	// Record the action state so the webui can detect this is a user-defined action
	stateResult := types.ActionState{
		ActionCurrentState: types.ActionCurrentState{
			Job:       job,
			Action:    chosenAction,
			Params:    params,
			Reasoning: reasoning,
		},
		ActionResult: types.ActionResult{
			Result: reasoning, // The reasoning/message to show to user
		},
	}

	// Add the action state to the job result
	job.Result.SetResult(stateResult)

	// Used by the observer
	conv = append(conv, openai.ChatCompletionMessage{
		Role: "assistant",
		ToolCalls: []openai.ToolCall{
			{
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      chosenAction.Definition().ToFunctionDefinition().Name,
					Arguments: params.String(),
				},
			},
		},
	})

	// Set conversation but leave Response empty
	// The webui will detect the user-defined action and generate the proper tool call response
	job.Result.Conversation = conv
	// job.Result.Response remains empty - this signals to webui that it should check State
	job.Result.Finish(nil)
}

// validateBuiltinTools checks that builtin tools specified by the user can be matched to available actions
func (a *Agent) validateBuiltinTools(job *types.Job) {
	builtinTools := job.GetBuiltinTools()
	if len(builtinTools) == 0 {
		return
	}

	// Get available actions
	availableActions := a.availableActions(job)

	for _, tool := range builtinTools {
		functionName := tool.Name

		// Check if this is a web search builtin tool
		if strings.HasPrefix(string(functionName), "web_search_") {
			// Look for a search action
			searchAction := availableActions.Find("search")
			if searchAction == nil {
				xlog.Warn("Web search builtin tool specified but no 'search' action available",
					"function_name", functionName,
					"agent", a.Character.Name)
			} else {
				xlog.Debug("Web search builtin tool matched to search action",
					"function_name", functionName,
					"agent", a.Character.Name)
			}
		} else {
			// For future builtin tools, add more matching logic here
			xlog.Warn("Unknown builtin tool specified",
				"function_name", functionName,
				"agent", a.Character.Name)
		}
	}
}

func (a *Agent) addFunctionResultToConversation(ctx context.Context, chosenAction types.Action, actionParams types.ActionParams, result types.ActionResult, conv Messages) Messages {
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

	// If it contains an image, we need to put it in the conversation (if supported by the model)
	if result.ImageBase64Result != "" {
		// iF model support both images and text, process it as a single multicontent message and return
		if !a.options.SeparatedMultimodalModel() {
			conv = append(conv, openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleTool,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: result.Result,
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: result.ImageBase64Result,
						},
					},
				},
				Name:       chosenAction.Definition().Name.String(),
				ToolCallID: chosenAction.Definition().Name.String(),
			})

			return conv
		} else {
			// We need to describe the image first, and we will process the text separately (we do not return here)
			imageDescription, err := a.describeImage(ctx, a.options.LLMAPI.MultimodalModel, result.ImageBase64Result)
			if err != nil {
				xlog.Error("Error describing image", "error", err)
			} else {
				conv = append(conv, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    fmt.Sprintf("Tool generated an image, the description of the image is: %s", imageDescription),
					Name:       chosenAction.Definition().Name.String(),
					ToolCallID: chosenAction.Definition().Name.String(),
				})
				if result.Result != "" {
					conv = append(conv, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    result.Result,
						Name:       chosenAction.Definition().Name.String(),
						ToolCallID: chosenAction.Definition().Name.String(),
					})
				}
			}
		}
	} else {
		conv = append(conv, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result.Result,
			Name:       chosenAction.Definition().Name.String(),
			ToolCallID: chosenAction.Definition().Name.String(),
		})
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

	// Register this job as the current one for its conversation (for cancel-previous-on-new-message)
	var conversationID string
	if job.Metadata != nil {
		if cid, ok := job.Metadata[types.MetadataKeyConversationID].(string); ok && cid != "" {
			conversationID = cid
			a.currentJobMu.Lock()
			a.currentJobByConversation[conversationID] = job
			a.currentJobMu.Unlock()
		}
	}
	if conversationID != "" {
		defer func() {
			a.currentJobMu.Lock()
			if a.currentJobByConversation[conversationID] == job {
				delete(a.currentJobByConversation, conversationID)
			}
			a.currentJobMu.Unlock()
		}()
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

	// Ensure job observable has Creation and Completion for jobs that bypass Execute() (e.g. periodic, scheduler)
	if job.Obs != nil && a.observer != nil {
		if job.Obs.Creation == nil && len(job.ConversationHistory) > 0 {
			m := job.ConversationHistory[len(job.ConversationHistory)-1]
			job.Obs.Creation = &types.Creation{ChatCompletionMessage: &m}
			a.observer.Update(*job.Obs)
		}
		job.Result.AddFinalizer(func(ccm []openai.ChatCompletionMessage) {
			if a.observer == nil {
				return
			}
			if job.Obs.Completion == nil {
				job.Obs.Completion = &types.Completion{}
			}
			job.Obs.Completion.Conversation = ccm
			if job.Result.Error != nil {
				job.Obs.Completion.Error = job.Result.Error.Error()
			}
			a.observer.Update(*job.Obs)
		})
	}

	conv = a.processPrompts(job.GetContext(), conv)
	if ok, err := a.filterJob(job); !ok || err != nil {
		if err != nil {
			job.Result.Finish(fmt.Errorf("Error in job filter: %w", err))
		} else {
			job.Result.Finish(nil)
		}
		return
	}
	conv = a.processUserInputs(conv)

	// RAG
	conv = a.knowledgeBaseLookup(job, conv)

	// Validate builtin tools against available actions
	a.validateBuiltinTools(job)

	fragment := cogito.NewFragment(conv...)

	if selfEvaluation {
		fragment = fragment.AddStartMessage("system", pickSelfTemplate)
	}

	if a.options.enableHUD {
		prompt, err := renderTemplate(hudTemplate, a.prepareHUD(), a.availableActions(job), "")
		if err != nil {
			job.Result.Finish(fmt.Errorf("error renderTemplate: %w", err))
			return
		}
		fragment = fragment.AddStartMessage("system", prompt)
	}

	availableActions := a.getAvailableActionsForJob(job)
	cogitoTools := availableActions.ToCogitoTools(job.GetContext(), a.sharedState)
	allActions := append(availableActions, a.mcpActionDefinitions...)

	obs := job.Obs

	defer func() {
		if obs != nil && a.observer != nil {
			obs.MakeLastProgressCompletion()
			a.observer.Update(*obs)
		}
	}()

	var err error
	var userTool bool
	// Set by tool callback when it decides the job outcome; Finish is then called once after ExecuteTools.
	var finishedByCallback bool
	var finishErr error

	var observables = make(map[string]*types.Observable)

	cogitoOpts := []cogito.Option{
		cogito.WithMCPs(a.mcpSessions...),
		cogito.WithTools(
			cogitoTools...,
		),
		cogito.WithSinkState(
			cogito.NewToolDefinition(
				NoToolToCallTool{},
				NoToolToCallArgs{},
				"no_tool_to_call",
				"Called when no other tool is needed to respond to the user",
			),
		),
		cogito.WithReasoningCallback(func(s string) {
			xlog.Debug("Cogito reasoning callback", "status", s)
			if s == "" {
				return
			}
			if a.observer != nil && job.Obs != nil {
				job.Obs.AddProgress(
					types.Progress{
						ChatCompletionResponse: &openai.ChatCompletionResponse{
							Choices: []openai.ChatCompletionChoice{
								{
									Message: openai.ChatCompletionMessage{
										Role:    "assistant",
										Content: s,
									},
								},
							},
						},
					})
				a.observer.Update(*job.Obs)
			}
			job.Callback(types.ActionCurrentState{
				Job:       job,
				Action:    nil,
				Params:    types.ActionParams{},
				Reasoning: s,
			})
		}),
		cogito.WithToolCallResultCallback(func(t cogito.ToolStatus) {
			toolObs := observables[t.ToolArguments.ID]
			if a.observer != nil && toolObs != nil {
				toolObs.Progress = append(toolObs.Progress, types.Progress{
					ActionResult: t.Result,
				})
				toolObs.Name = "action"
				toolObs.Icon = "bolt"
				toolObs.MakeLastProgressCompletion()
				a.observer.Update(*toolObs)
			}

			// Use full ActionResult (including Metadata) from action result,
			// so connectors receive e.g. songs_paths, images_url for sending files.
			actionResult := &types.ActionResult{
				Result: t.Result,
			}
			if t.ResultData != nil {
				switch res := t.ResultData.(type) {
				case types.ActionResult:
					actionResult = &res
				}
			}

			// Merge action metadata into job metadata so it accumulates across actions
			// and is available when ConversationAction runs
			if actionResult.Metadata != nil {
				if job.Metadata == nil {
					job.Metadata = make(map[string]interface{})
				}
				for key, value := range actionResult.Metadata {
					job.Metadata[key] = value
				}
			}

			aa := allActions.Find(t.Name)
			state := types.ActionState{
				ActionCurrentState: types.ActionCurrentState{
					Job:       job,
					Action:    aa,
					Params:    types.ActionParams(t.ToolArguments.Arguments),
					Reasoning: t.ToolArguments.Reasoning,
				},
				ActionResult: *actionResult,
			}
			job.Result.SetResult(state)
			job.CallbackWithResult(state)
			conv = a.addFunctionResultToConversation(job.GetContext(), aa, types.ActionParams(t.ToolArguments.Arguments), *actionResult, conv)
		}),
		cogito.WithToolCallBack(
			func(tc *cogito.ToolChoice, _ *cogito.SessionState) cogito.ToolCallDecision {

				xlog.Debug("Tool call back", "tool_call", tc)

				// Check if this is a user-defined action
				chosenAction := allActions.Find(tc.Name)

				xlog.Debug("Action found", "action", chosenAction)

				if chosenAction != nil && types.IsActionUserDefined(chosenAction) {
					xlog.Debug("User-defined action chosen, returning tool call", "action", chosenAction.Definition().Name)
					a.replyWithToolCall(job, conv, tc.Arguments, chosenAction, tc.Reasoning)
					userTool = true
					return cogito.ToolCallDecision{
						Approved: false,
					}
				}

				if a.observer != nil && job.Obs != nil {
					obs := a.observer.NewObservable()
					obs.Name = "decision"
					obs.ParentID = job.Obs.ID
					obs.Icon = "brain"
					obs.Creation = &types.Creation{
						ChatCompletionRequest: &openai.ChatCompletionRequest{
							Model:    a.options.LLMAPI.Model,
							Messages: conv,
						},
						FunctionDefinition: chosenAction.Definition().ToFunctionDefinition(),
						FunctionParams:     types.ActionParams(tc.Arguments),
					}

					a.observer.Update(*obs)
					observables[tc.ID] = obs
				}

				switch tc.Name {
				case action.StopActionName:
					return cogito.ToolCallDecision{
						Approved: false,
					}
				case action.ConversationActionName:
					message := action.ConversationActionResponse{}
					toolArgs, _ := json.Marshal(tc.Arguments)
					if err := json.Unmarshal([]byte(toolArgs), &message); err != nil {
						xlog.Error("Error unmarshalling conversation response", "error", err)
						finishedByCallback = true
						finishErr = fmt.Errorf("error unmarshalling conversation response: %w", err)
						return cogito.ToolCallDecision{
							Approved: false,
						}
					}

					msg := openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: message.Message,
					}

					// Get accumulated metadata from job (e.g., images, files generated by previous actions in this job)
					// This is per-job metadata, so parallel jobs won't interfere with each other
					metadata := job.Metadata

					go func(agent *Agent) {
						xlog.Info("Sending new conversation to channel", "agent", agent.Character.Name, "message", msg.Content, "metadata_keys", len(metadata))
						// Send ConversationMessage with both the message and accumulated metadata
						agent.newConversations <- types.NewConversationMessage(msg).WithMetadata(metadata)
						// Job metadata is automatically cleared when job finishes, no need to manually clear
					}(a)

					job.Result.Conversation = []openai.ChatCompletionMessage{
						msg,
					}
					job.Result.SetResponse("decided to initiate a new conversation")
					finishedByCallback = true
					finishErr = nil
					return cogito.ToolCallDecision{
						Approved: false,
					}
				case action.StateActionName:
					// We need to store the result in the state
					state := types.AgentInternalState{}
					dat, _ := json.Marshal(tc.Arguments)
					err = json.Unmarshal(dat, &state)
					stateObs := observables[tc.ID]
					if err != nil {
						werr := fmt.Errorf("error unmarshalling state of the agent: %w", err)
						if stateObs != nil && a.observer != nil {
							stateObs.Completion = &types.Completion{
								Error: werr.Error(),
							}
							a.observer.Update(*stateObs)
						}
						return cogito.ToolCallDecision{
							Approved: false,
						}
					}
					// update the current state with the one we just got from the action
					a.currentState = &state
					if stateObs != nil && a.observer != nil {
						stateObs.Progress = append(stateObs.Progress, types.Progress{
							AgentState: &state,
						})
						a.observer.Update(*stateObs)
					}

					// update the state file
					if a.options.statefile != "" {
						if err := a.SaveState(a.options.statefile); err != nil {
							if stateObs != nil && a.observer != nil {
								stateObs.Completion = &types.Completion{
									Error: err.Error(),
								}
								a.observer.Update(*stateObs)
							}

							return cogito.ToolCallDecision{
								Approved: false,
							}
						}
					}
					// Mark state tool-call observable as completed successfully
					if stateObs != nil && a.observer != nil {
						stateObs.MakeLastProgressCompletion()
						a.observer.Update(*stateObs)
					}

				}

				cont := job.Callback(types.ActionCurrentState{
					Job:       job,
					Action:    chosenAction,
					Params:    types.ActionParams(tc.Arguments),
					Reasoning: tc.Reasoning})

				if !cont {
					job.Result.SetResult(
						types.ActionState{
							ActionCurrentState: types.ActionCurrentState{
								Job:       job,
								Action:    chosenAction,
								Params:    types.ActionParams(tc.Arguments),
								Reasoning: tc.Reasoning,
							},
							ActionResult: types.ActionResult{Result: "stopped by callback"},
						})

					job.Result.Conversation = conv
					finishedByCallback = true
					finishErr = nil
				}
				return cogito.ToolCallDecision{
					Approved: cont,
				}
			},
		),
	}

	if a.options.canPlan {
		cogitoOpts = append(cogitoOpts, cogito.EnableAutoPlan)
		if a.options.enableEvaluation {
			cogitoOpts = append(cogitoOpts, cogito.EnableAutoPlanReEvaluator)
		}
		if a.options.LLMAPI.ReviewerModel != "" {
			llmClient := clients.NewLocalAILLM(a.options.LLMAPI.ReviewerModel, a.options.LLMAPI.APIKey, a.options.LLMAPI.APIURL)
			cogitoOpts = append(cogitoOpts, cogito.WithReviewerLLM(llmClient))
		}
	}

	// Important: DisableSinkState must be before WithForceReasoning()
	if a.options.disableSinkState {
		cogitoOpts = append(cogitoOpts, cogito.DisableSinkState)
	}

	if a.options.forceReasoning {
		cogitoOpts = append(cogitoOpts, cogito.WithForceReasoning())
	}

	if a.options.enableGuidedTools {
		cogitoOpts = append(cogitoOpts, cogito.EnableGuidedTools)
	}

	if a.options.maxEvaluationLoops > 0 {
		cogitoOpts = append(cogitoOpts,
			cogito.WithIterations(a.options.maxEvaluationLoops),
		)
	}

	if a.options.loopDetection > 0 {
		cogitoOpts = append(cogitoOpts, cogito.WithLoopDetection(a.options.loopDetection))
	}

	if a.options.forceReasoningTool {
		cogitoOpts = append(cogitoOpts,
			cogito.WithForceReasoningTool())
	}

	if a.options.enableAutoCompaction {
		cogitoOpts = append(cogitoOpts,
			cogito.WithCompactionThreshold(a.options.autoCompactionThreshold))
	}

	if a.options.maxAttempts > 1 {
		cogitoOpts = append(cogitoOpts, cogito.WithMaxAttempts(a.options.maxAttempts))
		cogitoOpts = append(cogitoOpts, cogito.WithMaxRetries(a.options.maxAttempts))
	}

	fragment, err = cogito.ExecuteTools(
		a.llm, fragment,
		cogitoOpts...,
	)

	if err != nil && !errors.Is(err, cogito.ErrNoToolSelected) && !errors.Is(err, cogito.ErrGoalNotAchieved) && !userTool {
		if obs != nil {
			obs.Completion = &types.Completion{
				Error: err.Error(),
			}
			a.observer.Update(*obs)
		}
		xlog.Error("Error executing cogito", "error", err)
		job.Result.Finish(err)
		return
	}

	if finishedByCallback {
		job.Result.Finish(finishErr)
		return
	}

	if userTool {
		return
	}

	if len(fragment.Messages) > 0 &&
		fragment.LastMessage().Role == "tool" {
		toolToCall := fragment.Messages[len(fragment.Messages)-2].ToolCalls[0].Function.Name
		switch toolToCall {
		case action.StopActionName:
			job.Result.Finish(nil)
			return
		}
	}

	if len(fragment.Messages) == 0 {
		job.Result.Finish(fmt.Errorf("no messages in fragment"))
		return
	}

	result := a.cleanupLLMResponse(fragment.LastMessage().Content)

	conv = append(fragment.Messages, openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: result,
	})

	job.Result.Plans = fragment.Status.Plans
	job.Result.Conversation = conv
	job.ConversationHistory = conv
	job.Result.AddFinalizer(func(conv []openai.ChatCompletionMessage) {
		a.saveCurrentConversation(conv)
	})
	job.Result.SetResponse(result)
	job.Result.Finish(nil)
}

func stripThinkingTags(content string) string {
	// Remove content between <thinking> and </thinking> (including multi-line)
	content = regexp.MustCompile(`(?s)<thinking>.*?</thinking>`).ReplaceAllString(content, "")
	// Remove content between <think> and </think> (including multi-line)
	content = regexp.MustCompile(`(?s)<think>.*?</think>`).ReplaceAllString(content, "")
	// Clean up any extra whitespace
	content = strings.TrimSpace(content)
	return content
}

func (a *Agent) cleanupLLMResponse(content string) string {
	if a.options.stripThinkingTags {
		content = stripThinkingTags(content)
	}
	// Future post-processing options can be added here
	return content
}

// This is running in the background.
func (a *Agent) periodicallyRun(timer *time.Timer) {
	// Remember always to reset the timer - if we don't the agent will stop..
	defer timer.Reset(a.options.periodicRuns)

	xlog.Debug("Agent is running periodically", "agent", a.Character.Name)

	if !a.options.standaloneJob {
		return
	}
	xlog.Info("Periodically running", "agent", a.Character.Name)

	// Here we go in a loop of
	// - asking the agent to do something
	// - evaluating the result
	// - asking the agent to do something else based on the result

	innerMonologue := a.options.innerMonologueTemplate
	if innerMonologue == "" {
		innerMonologue = innerMonologueTemplate
	}
	whatNext := types.NewJob(
		types.WithText(innerMonologue),
		types.WithReasoningCallback(a.options.reasoningCallback),
		types.WithResultCallback(a.options.resultCallback),
	)

	// Attach observable so UI can show standalone job progress (decisions, actions, reasoning)
	if a.observer != nil {
		obs := a.observer.NewObservable()
		obs.Name = "standalone"
		obs.Icon = "clock"
		a.observer.Update(*obs)
		whatNext.Obs = obs
	}

	a.consumeJob(whatNext, SystemRole)

	xlog.Info("STOP -- Periodically run is done", "agent", a.Character.Name)
}

func (a *Agent) Run() error {
	// Start the scheduler
	a.taskScheduler.Start()
	xlog.Info("Task scheduler started")

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

	timer := time.NewTimer(a.options.periodicRuns)

	// we fire the periodicalRunner only once.
	go a.periodicalRunRunner(timer)
	var errs []error
	var muErr sync.Mutex
	var wg sync.WaitGroup

	parallelJobs := a.options.parallelJobs
	if a.options.parallelJobs == 0 {
		parallelJobs = 1
	}

	for i := 0; i < parallelJobs; i++ {
		xlog.Debug("Starting agent worker", "worker", i)
		wg.Add(1)
		go func() {
			e := a.run(timer)
			muErr.Lock()
			errs = append(errs, e)
			muErr.Unlock()
			wg.Done()
		}()
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (a *Agent) run(timer *time.Timer) error {
	for {
		xlog.Debug("Agent is now waiting for a new job", "agent", a.Character.Name)
		select {
		case job := <-a.jobQueue:
			if !timer.Stop() {
				<-timer.C
			}
			xlog.Debug("Agent is consuming a job", "agent", a.Character.Name, "job", job)
			a.consumeJob(job, UserRole)
			timer.Reset(a.options.periodicRuns)
		case <-a.context.Done():
			// Agent has been canceled, return error
			xlog.Warn("Agent has been canceled", "agent", a.Character.Name)
			return ErrContextCanceled
		}
	}
}

func (a *Agent) periodicalRunRunner(timer *time.Timer) {
	for {
		select {
		case <-a.context.Done():
			// Agent has been canceled, return error
			xlog.Warn("periodicalRunner has been canceled", "agent", a.Character.Name)
			return
		case <-timer.C:
			a.periodicallyRun(timer)
		}
	}
}

func (a *Agent) Observer() Observer {
	return a.observer
}
