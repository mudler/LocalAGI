package agent_test

import (
	"net/http"

	. "github.com/mudler/LocalAGI/core/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent test", func() {
	Context("identity", func() {
		var agent *Agent

		BeforeEach(func() {
			Eventually(func() error {
				// test apiURL is working and available
				_, err := http.Get(apiURL + "/readyz")
				return err
			}, "10m", "10s").ShouldNot(HaveOccurred())
		})

		It("generates all the fields with random data", func() {
			var err error
			agent, err = New(
				WithLLMAPIURL(apiURL),
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
			var err error
			agent, err = New(WithRandomIdentity())
			Expect(err).To(HaveOccurred())
		})
		It("generates all the fields", func() {
			var err error

			agent, err := New(
				WithLLMAPIURL(apiURL),
				WithModel(testModel),
				WithRandomIdentity("An 90-year old man with a long beard, a wizard, who lives in a tower."),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(agent.Character.Name).ToNot(BeEmpty())
		})
	})
})
