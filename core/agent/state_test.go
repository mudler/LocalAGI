package agent_test

import (
	. "github.com/mudler/LocalAgent/core/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent test", func() {

	Context("identity", func() {
		It("generates all the fields with random data", func() {
			agent, err := New(
				WithLLMAPIURL(apiModel),
				WithModel(testModel),
				WithRandomIdentity(),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(agent.Character.Name).ToNot(BeEmpty())
			Expect(agent.Character.Age).ToNot(BeZero())
			Expect(agent.Character.Occupation).ToNot(BeEmpty())
			Expect(agent.Character.Hobbies).ToNot(BeEmpty())
			Expect(agent.Character.MusicTaste).ToNot(BeEmpty())
		})
		It("detect an invalid character", func() {
			_, err := New(WithRandomIdentity())
			Expect(err).To(HaveOccurred())
		})
		It("generates all the fields", func() {
			agent, err := New(
				WithLLMAPIURL(apiModel),
				WithModel(testModel),
				WithRandomIdentity("An old man with a long beard, a wizard, who lives in a tower."),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(agent.Character.Name).ToNot(BeEmpty())
			Expect(agent.Character.Age).ToNot(BeZero())
			Expect(agent.Character.Occupation).ToNot(BeEmpty())
			Expect(agent.Character.Hobbies).ToNot(BeEmpty())
			Expect(agent.Character.MusicTaste).ToNot(BeEmpty())
		})
	})
})
