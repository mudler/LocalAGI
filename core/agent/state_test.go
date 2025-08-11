package agent_test

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAGI/pkg/llm"
	"github.com/sashabaranov/go-openai"

	. "github.com/mudler/LocalAGI/core/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

)

var _ = Describe("Agent test", func() {
	Context("identity", func() {
		var agent *Agent

		// BeforeEach(func() {
		// 	Eventually(func() error {
		// 		// test apiURL is working and available
		// 		_, err := http.Get(apiURL + "/readyz")
		// 		return err
		// 	}, "10m", "10s").ShouldNot(HaveOccurred())
		// })

		It("generates all the fields with random data", func() {
			var llmClient llm.LLMClient
			if useRealLocalAI {
				llmClient = llm.NewClient(apiKey, apiURL, testModel)
			} else {
				llmClient = &llm.MockClient{
					CreateChatCompletionFunc: func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
						return openai.ChatCompletionResponse{
							Choices: []openai.ChatCompletionChoice{{
								Message: openai.ChatCompletionMessage{
									ToolCalls: []openai.ToolCall{{
										ID:   "tool_call_id_1",
										Type: "function",
										Function: openai.FunctionCall{
											Name:      "generate_identity",
											Arguments: `{"name":"John Doe","age":"42","job_occupation":"Engineer","hobbies":["reading","hiking"],"favorites_music_genres":["Jazz"]}`,
										},
									}},
								},
							}},
						}, nil
					},
				}
			}
			var err error
			agent, err = New(
				WithLLMClient(llmClient),
				WithModel(testModel),
				WithTimeout("10m"),
				WithRandomIdentity(),
			)
			Expect(err).ToNot(HaveOccurred())
			By("generating random identity")
			Expect(agent.Character.Name).ToNot(BeEmpty())
			Expect(agent.Character.Age).ToNot(BeZero())
			Expect(agent.Character.Occupation).ToNot(BeEmpty())
			Expect(agent.Character.Hobbies).ToNot(BeEmpty())
			Expect(agent.Character.MusicTaste).ToNot(BeEmpty())
		})
		It("detect an invalid character", func() {
			mock := &llm.MockClient{
				CreateChatCompletionFunc: func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					return openai.ChatCompletionResponse{}, fmt.Errorf("invalid character")
				},
			}
			var err error
			agent, err = New(
				WithLLMClient(mock),
				WithRandomIdentity(),
			)
			Expect(err).To(HaveOccurred())
		})
		It("generates all the fields", func() {
			mock := &llm.MockClient{
				CreateChatCompletionFunc: func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					return openai.ChatCompletionResponse{
						Choices: []openai.ChatCompletionChoice{{
							Message: openai.ChatCompletionMessage{
								ToolCalls: []openai.ToolCall{{
									ID:   "tool_call_id_2",
									Type: "function",
									Function: openai.FunctionCall{
										Name:      "generate_identity",
										Arguments: `{"name":"Gandalf","age":"90","job_occupation":"Wizard","hobbies":["magic","reading"],"favorites_music_genres":["Classical"]}`,
									},
								}},
							},
						}},
					}, nil
				},
			}
			var err error
			agent, err := New(
				WithLLMClient(mock),
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				WithRandomIdentity("An 90-year old man with a long beard, a wizard, who lives in a tower."),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(agent.Character.Name).ToNot(BeEmpty())
		})
	})
})
