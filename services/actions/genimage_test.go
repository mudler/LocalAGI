package actions_test

import (
	"context"
	"os"

	. "github.com/mudler/LocalAgent/core/action"

	. "github.com/mudler/LocalAgent/services/actions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GenImageAction", func() {
	var (
		ctx    context.Context
		action *GenImageAction
		params ActionParams
		config map[string]string
	)

	BeforeEach(func() {
		ctx = context.Background()
		apiKey := os.Getenv("OPENAI_API_KEY")
		apiURL := os.Getenv("OPENAI_API_URL")
		testModel := os.Getenv("OPENAI_MODEL")
		if apiURL == "" {
			Skip("OPENAI_API_URL must be set")
		}
		config = map[string]string{
			"apiKey": apiKey,
			"apiURL": apiURL,
			"model":  testModel,
		}
		action = NewGenImage(config)
	})

	Describe("Run", func() {
		It("should generate an image with valid prompt and size", func() {
			params = ActionParams{
				"prompt": "test prompt",
				"size":   "256x256",
			}

			url, err := action.Run(ctx, params)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).ToNot(BeEmpty())
		})

		It("should return an error if the prompt is not provided", func() {
			params = ActionParams{
				"size": "256x256",
			}

			_, err := action.Run(ctx, params)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Definition", func() {
		It("should return the correct action definition", func() {
			definition := action.Definition()
			Expect(definition.Name.String()).To(Equal("generate_image"))
			Expect(definition.Description).To(Equal("Generate image with."))
			Expect(definition.Properties).To(HaveKey("prompt"))
			Expect(definition.Properties).To(HaveKey("size"))
			Expect(definition.Required).To(ContainElement("prompt"))
		})
	})
})
