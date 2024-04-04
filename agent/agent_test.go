package agent_test

import (
	"fmt"

	"github.com/mudler/local-agent-framework/action"
	. "github.com/mudler/local-agent-framework/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const testActionResult = "In Boston it's 30C today, it's sunny, and humidity is at 98%"
const testActionResult2 = "In milan it's very hot today, it is 45C and the humidity is at 200%"
const testActionResult3 = "In paris it's very cold today, it is 2C and the humidity is at 10%"

var _ Action = &TestAction{}

var debugOptions = []JobOption{
	WithReasoningCallback(func(state ActionCurrentState) bool {
		fmt.Println("Reasoning", state)
		return true
	}),
	WithResultCallback(func(state ActionState) {
		fmt.Println("Reasoning", state.Reasoning)
		fmt.Println("Action", state.Action)
		fmt.Println("Result", state.Result)
	}),
}

type TestAction struct {
	response  []string
	responseN int
}

func (a *TestAction) Run(action.ActionParams) (string, error) {
	res := a.response[a.responseN]
	a.responseN++

	if len(a.response) == a.responseN {
		a.responseN = 0
	}

	return res, nil
}

func (a *TestAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
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

func (a *FakeStoreResultAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
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

func (a *FakeInternetAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
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
				WithLLMAPIURL(apiModel),
				WithModel(testModel),
				//	WithRandomIdentity(),
				WithActions(&TestAction{response: []string{testActionResult, testActionResult2, testActionResult3}}),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()
			res := agent.Ask(
				append(debugOptions,
					WithText("can you get the weather in boston, and afterward of Milano, Italy?"),
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
					WithText("Now I want to know the weather in Paris"),
				)...)
			conversation := agent.CurrentConversation()
			Expect(len(conversation)).To(Equal(10), fmt.Sprint(conversation))
			for _, r := range res.State {
				reasons = append(reasons, r.Result)
			}
			Expect(reasons).ToNot(ContainElement(testActionResult), fmt.Sprint(res))
			Expect(reasons).ToNot(ContainElement(testActionResult2), fmt.Sprint(res))
			Expect(reasons).To(ContainElement(testActionResult3), fmt.Sprint(res))

		})
		It("pick the correct action", func() {
			agent, err := New(
				WithLLMAPIURL(apiModel),
				WithModel(testModel),

				//	WithRandomIdentity(),
				WithActions(&TestAction{response: []string{testActionResult}}),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()
			res := agent.Ask(
				append(debugOptions,
					WithText("can you get the weather in boston?"))...,
			)
			reasons := []string{}
			for _, r := range res.State {
				reasons = append(reasons, r.Result)
			}
			Expect(reasons).To(ContainElement(testActionResult), fmt.Sprint(res))
		})

		It("updates the state with internal actions", func() {
			agent, err := New(
				WithLLMAPIURL(apiModel),
				WithModel(testModel),
				EnableHUD,
				DebugMode,
				//	EnableStandaloneJob,
				WithRandomIdentity(),
				WithPermanentGoal("I want to learn to play music"),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			result := agent.Ask(
				WithText("Update your goals such as you want to learn to play the guitar"),
			)
			fmt.Printf("%+v\n", result)
			Expect(result.Error).ToNot(HaveOccurred())
			Expect(agent.State().Goal).To(ContainSubstring("guitar"), fmt.Sprint(agent.State()))
		})

		FIt("it automatically performs things in the background", func() {
			agent, err := New(
				WithLLMAPIURL(apiModel),
				WithModel(testModel),
				EnableHUD,
				DebugMode,
				EnableStandaloneJob,
				WithAgentReasoningCallback(func(state ActionCurrentState) bool {
					fmt.Println("Reasoning", state)
					return true
				}),
				WithAgentResultCallback(func(state ActionState) {
					fmt.Println("Reasoning", state.Reasoning)
					fmt.Println("Action", state.Action)
					fmt.Println("Result", state.Result)
				}),
				WithActions(
					&FakeInternetAction{
						TestAction{
							response: []string{
								"Major cities in italy: Roma, Venice, Milan",
								"In rome it's 30C today, it's sunny, and humidity is at 98%",
								"In venice it's very hot today, it is 45C and the humidity is at 200%",
								"In milan it's very cold today, it is 2C and the humidity is at 10%",
							},
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
				WithRandomIdentity(),
				WithPermanentGoal("get the weather of all the cities in italy and store the results"),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()

			Eventually(func() string {
				fmt.Println(agent.State())
				return agent.State().Goal
			}, "10m", "10s").Should(ContainSubstring("weather"), fmt.Sprint(agent.State()))

			Eventually(func() string {
				fmt.Println(agent.State())
				return agent.State().String()
			}, "10m", "10s").Should(ContainSubstring("store"), fmt.Sprint(agent.State()))

			// result := agent.Ask(
			// 	WithText("Update your goals such as you want to learn to play the guitar"),
			// )
			// fmt.Printf("%+v\n", result)
			// Expect(result.Error).ToNot(HaveOccurred())
			// Expect(agent.State().Goal).To(ContainSubstring("guitar"), fmt.Sprint(agent.State()))
		})
	})
})
