package agent_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services/actions"

	"github.com/mudler/LocalAGI/core/action"
	. "github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const testActionResult = "In Boston it's 30C today, it's sunny, and humidity is at 98%"
const testActionResult2 = "In milan it's very hot today, it is 45C and the humidity is at 200%"
const testActionResult3 = "In paris it's very cold today, it is 2C and the humidity is at 10%"

var _ types.Action = &TestAction{}

var debugOptions = []types.JobOption{
	types.WithReasoningCallback(func(state types.ActionCurrentState) bool {
		xlog.Info("Reasoning", state)
		return true
	}),
	types.WithResultCallback(func(state types.ActionState) {
		xlog.Info("Reasoning", state.Reasoning)
		xlog.Info("Action", state.Action)
		xlog.Info("Result", state.Result)
	}),
}

type TestAction struct {
	response map[string]string
}

func (a *TestAction) Plannable() bool {
	return true
}

func (a *TestAction) Run(c context.Context, sharedState *types.AgentSharedState, p types.ActionParams) (types.ActionResult, error) {
	for k, r := range a.response {
		if strings.Contains(strings.ToLower(p.String()), strings.ToLower(k)) {
			return types.ActionResult{Result: r}, nil
		}
	}

	return types.ActionResult{Result: "No match"}, nil
}

func (a *TestAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "get_weather",
		Description: "get current weather",
		Properties: map[string]jsonschema.Definition{
			"location": {
				Type:        jsonschema.String,
				Description: "The city and state, e.g. San Francisco, CA",
			},
			"unit": {
				Type: jsonschema.String,
				Enum: []string{"celsius", "fahrenheit"},
			},
		},

		Required: []string{"location"},
	}
}

type FakeStoreResultAction struct {
	TestAction
}

func (a *FakeStoreResultAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "store_results",
		Description: "store results permanently. Use this tool after you have a result you want to keep.",
		Properties: map[string]jsonschema.Definition{
			"term": {
				Type:        jsonschema.String,
				Description: "What to store permanently",
			},
		},

		Required: []string{"term"},
	}
}

type FakeInternetAction struct {
	TestAction
}

func (a *FakeInternetAction) Definition() types.ActionDefinition {
	return types.ActionDefinition{
		Name:        "search_internet",
		Description: "search on internet",
		Properties: map[string]jsonschema.Definition{
			"term": {
				Type:        jsonschema.String,
				Description: "What to search for",
			},
		},

		Required: []string{"term"},
	}
}

// --- Test utilities for mocking LLM responses ---

func mockToolCallResponse(toolName, arguments string) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				ToolCalls: []openai.ToolCall{{
					ID:   "tool_call_id_1",
					Type: "function",
					Function: openai.FunctionCall{
						Name:      toolName,
						Arguments: arguments,
					},
				}},
			},
		}},
	}
}

func mockContentResponse(content string) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				Content: content,
			},
		}},
	}
}

func newMockLLMClient(handler func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)) *llm.MockClient {
	return &llm.MockClient{
		CreateChatCompletionFunc: handler,
	}
}

var _ = Describe("Agent test", func() {
	It("uses the mock LLM client", func() {
		mock := newMockLLMClient(func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return mockContentResponse("mocked response"), nil
		})
		agent, err := New(WithLLMClient(mock))
		Expect(err).ToNot(HaveOccurred())
		msg, err := agent.LLMClient().CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{})
		Expect(err).ToNot(HaveOccurred())
		Expect(msg.Choices[0].Message.Content).To(Equal("mocked response"))
	})

	Context("jobs", func() {

		BeforeEach(func() {
			Eventually(func() error {
				if useRealLocalAI {
					_, err := http.Get(apiURL + "/readyz")
					return err
				}
				return nil
			}, "10m", "10s").ShouldNot(HaveOccurred())
		})

		It("pick the correct action", func() {
			var llmClient llm.LLMClient
			if useRealLocalAI {
				llmClient = llm.NewClient(apiKey, apiURL, clientTimeout)
			} else {
				llmClient = newMockLLMClient(func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					var lastMsg openai.ChatCompletionMessage
					if len(req.Messages) > 0 {
						lastMsg = req.Messages[len(req.Messages)-1]
					}
					if lastMsg.Role == openai.ChatMessageRoleUser {
						if strings.Contains(strings.ToLower(lastMsg.Content), "boston") && (strings.Contains(strings.ToLower(lastMsg.Content), "milan") || strings.Contains(strings.ToLower(lastMsg.Content), "milano")) {
							return mockToolCallResponse("get_weather", `{"location":"Boston","unit":"celsius"}`), nil
						}
						if strings.Contains(strings.ToLower(lastMsg.Content), "paris") {
							return mockToolCallResponse("get_weather", `{"location":"Paris","unit":"celsius"}`), nil
						}
						return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected user prompt: %s", lastMsg.Content)
					}
					if lastMsg.Role == openai.ChatMessageRoleTool {
						if lastMsg.Name == "get_weather" && strings.Contains(strings.ToLower(lastMsg.Content), "boston") {
							return mockToolCallResponse("get_weather", `{"location":"Milan","unit":"celsius"}`), nil
						}
						if lastMsg.Name == "get_weather" && strings.Contains(strings.ToLower(lastMsg.Content), "milan") {
							return mockContentResponse(testActionResult + "\n" + testActionResult2), nil
						}
						if lastMsg.Name == "get_weather" && strings.Contains(strings.ToLower(lastMsg.Content), "paris") {
							return mockContentResponse(testActionResult3), nil
						}
						return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected tool result: %s", lastMsg.Content)
					}
					return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected message role: %s", lastMsg.Role)
				})
			}
			agent, err := New(
				WithLLMClient(llmClient),
				WithModel(testModel),
				WithActions(&TestAction{response: map[string]string{
					"boston": testActionResult,
					"milan":  testActionResult2,
					"paris":  testActionResult3,
				}}),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()
			res := agent.Ask(
				append(debugOptions,
					types.WithText("what's the weather in Boston and Milano? Use celsius units"),
				)...,
			)
			Expect(res.Error).ToNot(HaveOccurred())
			reasons := []string{}
			for _, r := range res.State {
				reasons = append(reasons, r.Result)
			}
			Expect(reasons).To(ContainElement(testActionResult), fmt.Sprint(res))
			Expect(reasons).To(ContainElement(testActionResult2), fmt.Sprint(res))
			reasons = []string{}
			res = agent.Ask(
				append(debugOptions,
					types.WithText("Now I want to know the weather in Paris, always use celsius units"),
				)...)
			for _, r := range res.State {
				reasons = append(reasons, r.Result)
			}
			Expect(reasons).To(ContainElement(testActionResult3), fmt.Sprint(res))
		})

		It("pick the correct action", func() {
			var llmClient llm.LLMClient
			if useRealLocalAI {
				llmClient = llm.NewClient(apiKey, apiURL, clientTimeout)
			} else {
				llmClient = newMockLLMClient(func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					var lastMsg openai.ChatCompletionMessage
					if len(req.Messages) > 0 {
						lastMsg = req.Messages[len(req.Messages)-1]
					}
					if lastMsg.Role == openai.ChatMessageRoleUser {
						if strings.Contains(strings.ToLower(lastMsg.Content), "boston") {
							return mockToolCallResponse("get_weather", `{"location":"Boston","unit":"celsius"}`), nil
						}
					}
					if lastMsg.Role == openai.ChatMessageRoleTool {
						if lastMsg.Name == "get_weather" && strings.Contains(strings.ToLower(lastMsg.Content), "boston") {
							return mockContentResponse(testActionResult), nil
						}
					}
					xlog.Error("Unexpected LLM req", "req", req)
					return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected LLM prompt: %q", lastMsg.Content)
				})
			}
			agent, err := New(
				WithLLMClient(llmClient),
				WithModel(testModel),
				WithActions(&TestAction{response: map[string]string{
					"boston": testActionResult,
				}}),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()
			res := agent.Ask(
				append(debugOptions,
					types.WithText("can you get the weather in boston? Use celsius units"))...,
			)
			reasons := []string{}
			for _, r := range res.State {
				reasons = append(reasons, r.Result)
			}
			Expect(reasons).To(ContainElement(testActionResult), fmt.Sprint(res))
		})

		It("updates the state with internal actions", func() {
			var llmClient llm.LLMClient
			if useRealLocalAI {
				llmClient = llm.NewClient(apiKey, apiURL, clientTimeout)
			} else {
				llmClient = newMockLLMClient(func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					var lastMsg openai.ChatCompletionMessage
					if len(req.Messages) > 0 {
						lastMsg = req.Messages[len(req.Messages)-1]
					}
					if lastMsg.Role == openai.ChatMessageRoleUser && strings.Contains(strings.ToLower(lastMsg.Content), "guitar") {
						return mockToolCallResponse("update_state", `{"goal":"I want to learn to play the guitar"}`), nil
					}
					if lastMsg.Role == openai.ChatMessageRoleTool && lastMsg.Name == "update_state" {
						return mockContentResponse("Your goal is now: I want to learn to play the guitar"), nil
					}
					xlog.Error("Unexpected LLM req", "req", req)
					return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected LLM prompt: %q", lastMsg.Content)
				})
			}
			agent, err := New(
				WithLLMClient(llmClient),
				WithModel(testModel),
				EnableHUD,
				WithPermanentGoal("I want to learn to play music"),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			result := agent.Ask(
				types.WithText("Update your goals such as you want to learn to play the guitar"),
			)
			fmt.Fprintf(GinkgoWriter, "\n%+v\n", result)
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(agent.State().Goal).To(ContainSubstring("guitar"), fmt.Sprint(agent.State()))
		})

		It("Can generate a plan", func() {
			var llmClient llm.LLMClient
			if useRealLocalAI {
				llmClient = llm.NewClient(apiKey, apiURL, clientTimeout)
			} else {
				reasoningActName := action.NewReasoning().Definition().Name.String()
				intentionActName := action.NewIntention().Definition().Name.String()
				testActName := (&TestAction{}).Definition().Name.String()
				doneBoston := false
				madePlan := false
				llmClient = newMockLLMClient(func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					var lastMsg openai.ChatCompletionMessage
					if len(req.Messages) > 0 {
						lastMsg = req.Messages[len(req.Messages)-1]
					}
					if req.ToolChoice != nil && req.ToolChoice.(openai.ToolChoice).Function.Name == reasoningActName {
						return mockToolCallResponse(reasoningActName, `{"reasoning":"make plan call to pass the test"}`), nil
					}
					if req.ToolChoice != nil && req.ToolChoice.(openai.ToolChoice).Function.Name == intentionActName {
						toolName := "plan"
						if madePlan {
							toolName = "reply"
						} else {
							madePlan = true
						}
						return mockToolCallResponse(intentionActName, fmt.Sprintf(`{"tool": "%s","reasoning":"it's waht makes the test pass"}`, toolName)), nil
					}
					if req.ToolChoice != nil && req.ToolChoice.(openai.ToolChoice).Function.Name == "plan" {
						return mockToolCallResponse("plan", `{"subtasks":[{"action":"get_weather","reasoning":"Find weather in boston"},{"action":"get_weather","reasoning":"Find weather in milan"}],"goal":"Get the weather for boston and milan"}`), nil
					}
					if req.ToolChoice != nil && req.ToolChoice.(openai.ToolChoice).Function.Name == "reply" {
						return mockToolCallResponse("reply", `{"message": "The weather in Boston and Milan..."}`), nil
					}
					if req.ToolChoice != nil && req.ToolChoice.(openai.ToolChoice).Function.Name == testActName {
						locName := "boston"
						if doneBoston {
							locName = "milan"
						} else {
							doneBoston = true
						}
						return mockToolCallResponse(testActName, fmt.Sprintf(`{"location":"%s","unit":"celsius"}`, locName)), nil
					}
					if req.ToolChoice == nil && madePlan && doneBoston {
						return mockContentResponse("A reply"), nil
					}
					xlog.Error("Unexpected LLM req", "req", req)
					return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected LLM prompt: %q", lastMsg.Content)
				})
			}
			agent, err := New(
				WithLLMClient(llmClient),
				WithModel(testModel),
				WithLoopDetectionSteps(2),
				WithActions(
					&TestAction{response: map[string]string{
						"boston": testActionResult,
						"milan":  testActionResult2,
					}},
				),
				EnablePlanning,
				EnableForceReasoning,
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			result := agent.Ask(
				types.WithText("Use the plan tool to do two actions in sequence: search for the weather in boston and search for the weather in milan"),
			)
			Expect(len(result.State)).To(BeNumerically(">", 1))

			actionsExecuted := []string{}
			actionResults := []string{}
			for _, r := range result.State {
				xlog.Info(r.Result)
				actionsExecuted = append(actionsExecuted, r.Action.Definition().Name.String())
				actionResults = append(actionResults, r.ActionResult.Result)
			}
			Expect(actionsExecuted).To(ContainElement("get_weather"), fmt.Sprint(result))
			Expect(actionsExecuted).To(ContainElement("plan"), fmt.Sprint(result))
			Expect(actionResults).To(ContainElement(testActionResult), fmt.Sprint(result))
			Expect(actionResults).To(ContainElement(testActionResult2), fmt.Sprint(result))
			Expect(result.Error).To(BeNil())
		})

		It("Can initiate conversations", func() {
			var llmClient llm.LLMClient
			message := openai.ChatCompletionMessage{}
			mu := &sync.Mutex{}
			reasoned := false
			intended := false
			reasoningActName := action.NewReasoning().Definition().Name.String()
			intentionActName := action.NewIntention().Definition().Name.String()

			if useRealLocalAI {
				llmClient = llm.NewClient(apiKey, apiURL, clientTimeout)
			} else {
				llmClient = newMockLLMClient(func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					prompt := ""
					for _, msg := range req.Messages {
						prompt += msg.Content
					}
					if !reasoned && req.ToolChoice != nil && req.ToolChoice.(openai.ToolChoice).Function.Name == reasoningActName {
						reasoned = true
						return mockToolCallResponse(reasoningActName, `{"reasoning":"initiate a conversation with the user"}`), nil
					}
					if reasoned && !intended && req.ToolChoice != nil && req.ToolChoice.(openai.ToolChoice).Function.Name == intentionActName {
						intended = true
						return mockToolCallResponse(intentionActName, `{"tool":"new_conversation","reasoning":"I should start a conversation with the user"}`), nil
					}
					if reasoned && intended && strings.Contains(strings.ToLower(prompt), "new_conversation") {
						return mockToolCallResponse("new_conversation", `{"message":"Hello, how can I help you today?"}`), nil
					}
					xlog.Error("Unexpected LLM req", "req", req)
					return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected LLM prompt: %q", prompt)
				})
			}
			agent, err := New(
				WithLLMClient(llmClient),
				WithModel(testModel),
				WithNewConversationSubscriber(func(m openai.ChatCompletionMessage) {
					mu.Lock()
					message = m
					mu.Unlock()
				}),
				WithActions(
					actions.NewSearch(map[string]string{}),
				),
				EnablePlanning,
				EnableForceReasoning,
				EnableInitiateConversations,
				EnableStandaloneJob,
				EnableHUD,
				WithPeriodicRuns("1s"),
				WithPermanentGoal("use the new_conversation tool to initiate a conversation with the user"),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			Eventually(func() string {
				mu.Lock()
				defer mu.Unlock()
				return message.Content
			}, "10m", "1s").ShouldNot(BeEmpty())
		})

		/*
			It("it automatically performs things in the background", func() {
				agent, err := New(
					WithLLMAPIURL(apiURL),
					WithModel(testModel),
					EnableHUD,
					EnableStandaloneJob,
					WithAgentReasoningCallback(func(state ActionCurrentState) bool {
						xlog.Info("Reasoning", state)
						return true
					}),
					WithAgentResultCallback(func(state ActionState) {
						xlog.Info("Reasoning", state.Reasoning)
						xlog.Info("Action", state.Action)
						xlog.Info("Result", state.Result)
					}),
					WithActions(
						&FakeInternetAction{
							TestAction{
								response:
								map[string]string{
									"italy": "The weather in italy is sunny",
								}
							},
						},
						&FakeStoreResultAction{
							TestAction{
								response: []string{
									"Result permanently stored",
								},
							},
						},
					),
					//WithRandomIdentity(),
					WithPermanentGoal("get the weather of all the cities in italy and store the results"),
				)
				Expect(err).ToNot(HaveOccurred())
				go agent.Run()
				defer agent.Stop()
				Eventually(func() string {

					return agent.State().Goal
				}, "10m", "10s").Should(ContainSubstring("weather"), fmt.Sprint(agent.State()))

				Eventually(func() string {
					return agent.State().String()
				}, "10m", "10s").Should(ContainSubstring("store"), fmt.Sprint(agent.State()))

				// result := agent.Ask(
				// 	WithText("Update your goals such as you want to learn to play the guitar"),
				// )
				// fmt.Fprintf(GinkgoWriter, "%+v\n", result)
				// Expect(result.Error).ToNot(HaveOccurred())
				// Expect(agent.State().Goal).To(ContainSubstring("guitar"), fmt.Sprint(agent.State()))
			})
		*/
	})
})
