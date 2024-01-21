package agent_test

import (
	"fmt"

	. "github.com/mudler/local-agent-framework/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent test", func() {

	Context("identity", func() {
		It("generates all the fields with random data", func() {
			agent, err := New(
				WithLLMAPIURL("http://192.168.68.113:8080"),
				WithModel("echidna"),
				WithRandomIdentity(),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(agent.Character.Name).ToNot(BeEmpty())
			Expect(agent.Character.Age).ToNot(BeZero())
			Expect(agent.Character.Occupation).ToNot(BeEmpty())
			Expect(agent.Character.NowDoing).ToNot(BeEmpty())
			Expect(agent.Character.DoingNext).ToNot(BeEmpty())
			Expect(agent.Character.DoneHistory).ToNot(BeEmpty())
			Expect(agent.Character.Memories).ToNot(BeEmpty())
			Expect(agent.Character.Hobbies).ToNot(BeEmpty())
			Expect(agent.Character.MusicTaste).ToNot(BeEmpty())
			fmt.Printf("%+v\n", agent.Character)
		})
		It("detect an invalid character", func() {
			_, err := New(WithRandomIdentity())
			Expect(err).To(HaveOccurred())
		})
		It("generates all the fields", func() {
			agent, err := New(
				WithLLMAPIURL("http://192.168.68.113:8080"),
				WithModel("echidna"),
				WithRandomIdentity("An old man with a long beard, a wizard, who lives in a tower."),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(agent.Character.Name).ToNot(BeEmpty())
			Expect(agent.Character.Age).ToNot(BeZero())
			Expect(agent.Character.Occupation).ToNot(BeEmpty())
			Expect(agent.Character.NowDoing).ToNot(BeEmpty())
			Expect(agent.Character.DoingNext).ToNot(BeEmpty())
			Expect(agent.Character.DoneHistory).ToNot(BeEmpty())
			Expect(agent.Character.Memories).ToNot(BeEmpty())
			Expect(agent.Character.Hobbies).ToNot(BeEmpty())
			Expect(agent.Character.MusicTaste).ToNot(BeEmpty())
			fmt.Printf("%+v\n", agent.Character)
		})
	})
})
