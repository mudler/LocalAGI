package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	localagi "github.com/mudler/LocalAGI/pkg/client"
	"github.com/mudler/LocalAGI/pkg/utils/ptr"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/sashabaranov/go-openai/jsonschema"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent test", func() {
	Context("Creates an agent and it answers", func() {
		BeforeEach(func() {
			Eventually(func() error {
				// test apiURL is working and available
				_, err := http.Get(localagiURL + "/readyz")
				return err
			}, "10m", "10s").ShouldNot(HaveOccurred())

			client := localagi.NewClient(localagiURL, "", time.Minute)
			err := client.DeleteAgent("testagent")
			Expect(err).ToNot(HaveOccurred())
		})

		It("create agent", func() {
			client := localagi.NewClient(localagiURL, "", 5*time.Minute)

			err := client.CreateAgent(&localagi.AgentConfig{
				Name: "testagent",
			})
			Expect(err).ToNot(HaveOccurred())

			result, err := client.SimpleAIResponse("testagent", "hello")
			Expect(err).ToNot(HaveOccurred())

			Expect(result).ToNot(BeEmpty())
		})

		It("tool call", func() {
			client := localagi.NewClient(localagiURL, "", 5*time.Minute)

			err := client.CreateAgent(&localagi.AgentConfig{
				Name: "testagent",
			})
			Expect(err).ToNot(HaveOccurred())

			req := localagi.RequestBody{
				Model: "testagent",
				Input: "Create an appointment next week on wednesday at 10:00 am for the whole day. The topic is about AI and you include ABC and DEF to the appointment.",
				Tools: []localagi.Tool{
					{
						Type:        "function",
						Name:        ptr.To("CreateTask"),
						Description: ptr.To("Write the needed details whenever you're asked to create something like an info, appointment, e-mail or when you're asked to remind of anything or create a remainder. Also use this if you're supposed to answer an e-mail."),
						Parameters: ptr.To(jsonschema.Definition{
							Type: "object",
							Properties: map[string]jsonschema.Definition{
								"task": {
									Type:        "string",
									Description: "Look for the name of the task you're supposed to do or create ",
									Enum: []string{
										"appointment",
										"E-mail",
									},
								},
								"subject": {
									Type:        "string",
									Description: "A subject the task is about. Infer this from the given context data and user prompt.",
								},
								"reply": {
									Type:        "string",
									Description: "A sharp and short reply to the contextual data given. Use a friendly and neutral general greeting.",
								},
								"recipient": {
									Type:        "array",
									Description: "A list of names and abbreviations to send our task to. Abbreviations always have to match exactly. If the user gives you first names you can deduce the last name.",
									Items: &jsonschema.Definition{
										Type: "string",
										Enum: []string{
											"ABC",
											"DEF",
										},
									},
								},
								"datestart": {
									Type:        "string",
									Description: "The date and time when the task should start. Discard any older dates than today. Use tomorrow as default. Use the format DD/MM/YYYY HH:MM",
								},
								"dateend": {
									Type:        "string",
									Description: "The date and time when a meeting should end. Default to start date. If the duration of an appointment is given, calculate the end with the start date. Use the format DD/MM/YYYY HH:MM",
								},
								"datedone": {
									Type:        "string",
									Description: "The date and time when the task should be done. Use the format DD/MM/YYYY HH:MM",
								},
								"private": {
									Type:        "boolean",
									Description: "Whether the task should be private or not. Default to false.",
								},
								"includeall": {
									Type:        "boolean",
									Description: "Whether the task should include every mentioned person or not. Default to true. If you find explicitly mentioned people in the prompt whilst ignoring the contextual xml schema you choose false unless it is mentioned that you should include everyone.",
								},
								"wholedayappointment": {
									Type:        "boolean",
									Description: "Whether the appointment should be done for the whole days. Default to false unless mentioned by the user prompt. Ignore the xml schema for this.",
								},
								"remainder": {
									Type:        "boolean",
									Description: "Whether you are explicitly supposed to remind of something or not. Default to false. Ignore the xml schema for this.",
								},
							},
							Required: []string{
								"task",
								"recipient",
								"datestart",
								"dateend",
								"datedone",
								"private",
								"wholedayappointment",
								"remainder",
								"subject",
								"reply",
								"includeall",
							},
						}),
					},
				}}
			result, err := client.GetAIResponse(&req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			var call localagi.ResponseFunctionToolCall
			var args struct {
				Task                string   `json:"task"`
				Subject             string   `json:"subject"`
				Reply               string   `json:"reply"`
				Recipient           []string `json:"recipient"`
				DateStart           string   `json:"datestart"`
				DateEnd             string   `json:"dateend"`
				DateDone            string   `json:"datedone"`
				Private             bool     `json:"private"`
				IncludeAll          bool     `json:"includeall"`
				WholeDayAppointment bool     `json:"wholedayappointment"`
				Remainder           bool     `json:"remainder"`
			}

			for _, out := range result.Output {
				msg, err := out.ToMessage()
				if err == nil && msg.Role == "assistant" {
					xlog.Info("Agent returned message", "message", msg)
					continue
				}
				fnc, err := out.ToFunctionToolCall()
				call = fnc
				Expect(err).ToNot(HaveOccurred())
				Expect(string(fnc.Type)).To(Equal("function_call"))
				Expect(fnc.Name).To(Equal("CreateTask"))

				err = json.Unmarshal([]byte(fnc.Arguments), &args)
				Expect(err).ToNot(HaveOccurred())

				Expect(args.Task).To(Equal("appointment"))
				Expect(args.Subject).ToNot(BeEmpty())
				Expect(args.Reply).ToNot(BeEmpty())
			}

			req = localagi.RequestBody{
				Model: "testagent",
				Input: []any{
					localagi.InputMessage{
						Type:    "message",
						Role:    "user",
						Content: "Create an appointment next week on wednesday at 10:00 am for the whole day. The topic is about AI and you include ABC and DEF to the appointment.",
					},
					call,
					localagi.InputFunctionToolCallOutput{
						Type:   "function_call_output",
						CallID: call.CallID,
						Output: fmt.Sprintf("Successfully created %s: %s", args.Task, args.Subject),
					},
					localagi.InputMessage{
						Type:    "message",
						Role:    "user",
						Content: "Was the appointment created?",
					},
				},
				Tools: []localagi.Tool{
					{
						Type:        "function",
						Name:        ptr.To("ChooseAnswer"),
						Description: ptr.To("Select Yes or No"),
						Parameters: ptr.To(jsonschema.Definition{
							Type: "object",
							Properties: map[string]jsonschema.Definition{
								"answer": {
									Type:        "boolean",
									Description: "Set true for Yes and false for no",
								},
							},
							Required: []string{
								"answer",
							},
						}),
					},
				},
				ToolChoice: &localagi.ToolChoice{
					Type: "function",
					Name: "ChooseAnswer",
				},
			}
			result, err = client.GetAIResponse(&req)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(result.Output)).To(BeNumerically(">", 0))
			fnc, err := result.Output[len(result.Output)-1].ToFunctionToolCall()
			Expect(err).ToNot(HaveOccurred())
			Expect(fnc.Arguments).To(ContainSubstring("true"))
		})
	})
})
