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
const testActionResult2 = "In milan it's very hot today"

var _ Action = &TestAction{}

type TestAction struct {
	response  []string
	responseN int
}

func (a *TestAction) Run(action.ActionParams) (string, error) {
	res := a.response[a.responseN]
	if len(a.response) == a.responseN {
		a.responseN = 0
	} else {
		a.responseN++
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
		FIt("pick the correct action", func() {
			agent, err := New(
				WithLLMAPIURL(apiModel),
				WithModel(testModel),
				//	WithRandomIdentity(),
				WithActions(&TestAction{response: []string{testActionResult, testActionResult2}}),
			)
			Expect(err).ToNot(HaveOccurred())
			go agent.Run()
			defer agent.Stop()
			res := agent.Ask("can you get the weather in boston, and afterward of Milano, Italy?", "")
			Expect(res).To(ContainElement(testActionResult), fmt.Sprint(res))
			Expect(res).To(ContainElement(testActionResult2), fmt.Sprint(res))
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
			res := agent.Ask("can you get the weather in boston?", "")
			Expect(res).To(ContainElement(testActionResult), fmt.Sprint(res))
		})
	})
})
