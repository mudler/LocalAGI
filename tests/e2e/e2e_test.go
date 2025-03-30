package e2e_test

import (
	"time"

	localagent "github.com/mudler/LocalAgent/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent test", func() {
	Context("Creates an agent and it answers", func() {
		It("create agent", func() {
			client := localagent.NewClient(localagentURL, "", 5*time.Minute)

			err := client.CreateAgent(&localagent.AgentConfig{
				Name: "testagent",
			})
			Expect(err).ToNot(HaveOccurred())

			result, err := client.SimpleAIResponse("testagent", "hello")
			Expect(err).ToNot(HaveOccurred())

			Expect(result).ToNot(BeEmpty())
		})
	})
})
