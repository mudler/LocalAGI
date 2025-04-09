package e2e_test

import (
	"net/http"
	"time"

	localagi "github.com/mudler/LocalAGI/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent test", func() {
	Context("Creates an agent and it answers", func() {
		BeforeEach(func() {
			Eventually(func() error {
				// test apiURL is working and available
				_, err := http.Get(apiURL + "/readyz")
				return err
			}, "10m", "10s").ShouldNot(HaveOccurred())
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
	})
})
