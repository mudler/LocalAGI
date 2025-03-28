package agent_test

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/mudler/LocalAgent/services/actions"

	. "github.com/mudler/LocalAgent/core/agent"
	"github.com/mudler/LocalAgent/core/types"
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

func (a *TestAction) Run(c context.Context, p types.ActionParams) (types.ActionResult, error) {
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

var _ = Describe("Agent test", func() {
	Context("jobs", func() {
		It("pick the correct action", func() {
			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				//	WithRandomIdentity(),
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
			//Expect(reasons).ToNot(ContainElement(testActionResult), fmt.Sprint(res))
			//Expect(reasons).ToNot(ContainElement(testActionResult2), fmt.Sprint(res))
			Expect(reasons).To(ContainElement(testActionResult3), fmt.Sprint(res))
			// conversation := agent.CurrentConversation()
			// for _, r := range res.State {
			// 	reasons = append(reasons, r.Result)
			// }
			//			Expect(len(conversation)).To(Equal(10), fmt.Sprint(conversation))
		})
		It("pick the correct action", func() {
			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),

				//	WithRandomIdentity(),
				WithActions(&TestAction{response: map[string]string{
					"boston": testActionResult,
				},
				}),
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
			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				EnableHUD,
				//	EnableStandaloneJob,
				//	WithRandomIdentity(),
				WithPermanentGoal("I want to learn to play music"),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			result := agent.Ask(
				types.WithText("Update your goals such as you want to learn to play the guitar"),
			)
			fmt.Printf("%+v\n", result)
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(agent.State().Goal).To(ContainSubstring("guitar"), fmt.Sprint(agent.State()))
		})

		It("Can generate a plan", func() {
			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				WithLLMAPIKey(apiKeyURL),
				WithActions(
					actions.NewSearch(map[string]string{}),
				),
				EnablePlanning,
				EnableForceReasoning,
				//	EnableStandaloneJob,
				//	WithRandomIdentity(),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			result := agent.Ask(
				types.WithText("plan a trip to San Francisco from Venice, Italy"),
			)
			Expect(len(result.State)).To(BeNumerically(">", 1))

			actionsExecuted := []string{}
			for _, r := range result.State {
				xlog.Info(r.Result)
				actionsExecuted = append(actionsExecuted, r.Action.Definition().Name.String())
			}
			Expect(actionsExecuted).To(ContainElement("search_internet"), fmt.Sprint(result))
			Expect(actionsExecuted).To(ContainElement("plan"), fmt.Sprint(result))

		})

		It("Can initiate conversations", func() {

			message := openai.ChatCompletionMessage{}
			mu := &sync.Mutex{}
			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				WithLLMAPIKey(apiKeyURL),
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
				WithPermanentGoal("use the new_conversation tool"),
				//	EnableStandaloneJob,
				//	WithRandomIdentity(),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			Eventually(func() string {
				mu.Lock()
				defer mu.Unlock()
				return message.Content
			}, "10m", "10s").ShouldNot(BeEmpty())
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
				// fmt.Printf("%+v\n", result)
				// Expect(result.Error).ToNot(HaveOccurred())
				// Expect(agent.State().Goal).To(ContainSubstring("guitar"), fmt.Sprint(agent.State()))
			})
		*/
	})
})
