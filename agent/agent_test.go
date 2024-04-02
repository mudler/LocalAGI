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
				WithReasoningCallback(func(state ActionCurrentState) bool {
					fmt.Println("Reasoning", state)
					return true
				}),
				WithText("can you get the weather in boston, and afterward of Milano, Italy?"),
			)
			reasons := []string{}
			for _, r := range res {
				reasons = append(reasons, r.Result)
			}
			Expect(reasons).To(ContainElement(testActionResult), fmt.Sprint(res))
			Expect(reasons).To(ContainElement(testActionResult2), fmt.Sprint(res))
			reasons = []string{}

			res = agent.Ask(WithText("Now I want to know the weather in Paris"))
			conversation := agent.CurrentConversation()
			Expect(len(conversation)).To(Equal(10), fmt.Sprint(conversation))
			for _, r := range res {
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
				WithText("can you get the weather in boston?"),
			)
			reasons := []string{}
			for _, r := range res {
				reasons = append(reasons, r.Result)
			}
			Expect(reasons).To(ContainElement(testActionResult), fmt.Sprint(res))
		})
	})
})
