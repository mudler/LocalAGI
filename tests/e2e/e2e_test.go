package e2e_test

import (
	"time"

	localagi "github.com/mudler/LocalAGI/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent test", func() {
	Context("Creates an agent and it answers", func() {
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
	})
})
