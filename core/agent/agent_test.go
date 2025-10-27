package agent_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services/actions"

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
	response   map[string]string
	definition *types.ActionDefinition
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
	def := types.ActionDefinition{
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

	if a.definition != nil {
		def = *a.definition
	}
	return def
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

		BeforeEach(func() {
			Eventually(func() error {
				// test apiURL is working and available
				_, err := http.Get(apiURL + "/readyz")
				return err
			}, "10m", "10s").ShouldNot(HaveOccurred())
		})

		It("pick the correct action", func() {
			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				EnableForceReasoning,
				WithTimeout("10m"),
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

			Expect(len(res.Conversation)).To(BeNumerically(">", 1), fmt.Sprint(res.Conversation))

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
				WithTimeout("10m"),
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
				WithTimeout("10m"),
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
				WithTimeout("10m"),
				WithMaxEvaluationLoops(1),
				WithActions(
					&TestAction{
						response: map[string]string{
							"boston": testActionResult,
							"milan":  testActionResult2,
						},
					},
					&TestAction{
						response: map[string]string{
							"flight": "Flight options from Boston to Milan (April 22-26, 2025):\n• Outbound: Boston Logan (BOS) → Milan Malpensa (MXP), April 22, 2025\n  - Economy: $450-650 (Alitalia, Delta, Lufthansa)\n  - Business: $1,200-1,800\n  - Flight time: 8h 15m (1 stop) or 9h 45m (direct)\n• Return: Milan Malpensa (MXP) → Boston Logan (BOS), April 26, 2025\n  - Economy: $420-580\n  - Business: $1,100-1,600\n• Total estimated cost: $870-1,230 per person\n• Best booking window: 2-3 months in advance for optimal prices",
							"hotel":  "Hotel recommendations in Milan for April 22-26, 2025:\n• Luxury (4-5 stars): $200-400/night\n  - Hotel Principe di Savoia: $380/night (central location)\n  - Mandarin Oriental: $420/night (luxury amenities)\n• Mid-range (3-4 stars): $120-200/night\n  - Hotel Spadari al Duomo: $160/night (near cathedral)\n  - Hotel Milano Scala: $140/night (theater district)\n• Budget (2-3 stars): $80-120/night\n  - Hotel Bernina: $95/night (near train station)\n• Total 4-night stay: $320-1,680 depending on category\n• Booking tip: Reserve early for spring season discounts",
							"car":    "Car rental options in Milan for April 22-26, 2025:\n• Economy cars: $35-50/day (Fiat 500, VW Polo)\n• Compact cars: $45-65/day (Ford Focus, Opel Astra)\n• Mid-size cars: $60-85/day (BMW 3 Series, Audi A4)\n• SUV/Luxury: $90-150/day (BMW X3, Mercedes E-Class)\n• Total 4-day rental: $140-600\n• Pickup locations: Milan Malpensa Airport, Milan Central Station, city center\n• Insurance: $15-25/day additional\n• Fuel: ~$60-80 for 4 days of city driving\n• Parking: $20-40/day in city center hotels",
							"food":   "Dining budget and recommendations for Milan (April 22-26, 2025):\n• Fine dining: $80-150/person (Michelin-starred restaurants)\n  - Cracco: $120/person (2 Michelin stars)\n  - Trussardi alla Scala: $100/person\n• Mid-range restaurants: $40-80/person\n  - Luini: $15/person (famous panzerotti)\n  - Piz: $25/person (authentic pizza)\n• Casual dining: $20-40/person\n  - Aperitivo bars: $15-25/person\n  - Street food: $8-15/person\n• Daily food budget: $60-120/person\n• Total 4-day food cost: $240-480/person\n• Must-try: Risotto alla Milanese, Osso Buco, Panettone",
						},
						definition: &types.ActionDefinition{
							Name:        "search_internet",
							Description: "search the internet for information",
							Properties: map[string]jsonschema.Definition{
								"query": {
									Type:        jsonschema.String,
									Description: "The query to search for",
								},
							},
							Required: []string{"query"},
						},
					},
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
				types.WithText("Create a plan for my 4-day trip from Boston to milan in April of this year (2025). I'm not sure about the dates yet, I want you to find out the best dates also according to what you find."),
			)

			Expect(len(result.Conversation)).To(BeNumerically(">", 1), fmt.Sprint(result.Conversation))

			Expect(len(result.Plans)).To(BeNumerically(">=", 1), fmt.Sprintf("%+v", result))
			Expect(len(result.State)).To(BeNumerically(">=", 1))

			actionsExecuted := []string{}
			for _, r := range result.State {
				xlog.Info(r.Result)
				actionsExecuted = append(actionsExecuted, r.Action.Definition().Name.String())
			}
			Expect(actionsExecuted).To(Or(ContainElement("search_internet"), ContainElement("get_weather")), fmt.Sprint(result))
		})

		It("Can initiate conversations", func() {

			message := openai.ChatCompletionMessage{}
			mu := &sync.Mutex{}
			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				WithLLMAPIKey(apiKeyURL),
				WithTimeout("10m"),
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
