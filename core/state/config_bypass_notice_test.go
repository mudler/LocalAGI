package state_test

import (
	"encoding/json"

	"github.com/mudler/LocalAGI/core/state"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgentConfig required-tool bypass notice", func() {
	It("reads the notice from JSON and writes it back", func() {
		var c state.AgentConfig
		Expect(json.Unmarshal([]byte(`{"required_tool_bypass_notice":"not validated"}`), &c)).To(Succeed())
		Expect(c.RequiredToolBypassNotice).To(Equal("not validated"))
		out, err := json.Marshal(c)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(ContainSubstring(`"required_tool_bypass_notice":"not validated"`))
	})

	It("is empty when absent", func() {
		var c state.AgentConfig
		Expect(json.Unmarshal([]byte(`{"name":"x"}`), &c)).To(Succeed())
		Expect(c.RequiredToolBypassNotice).To(BeEmpty())
	})

	It("is offered in the agent form", func() {
		meta := state.NewAgentConfigMeta(nil, nil, nil, nil)
		names := []string{}
		for _, f := range meta.Fields {
			names = append(names, f.Name)
		}
		Expect(names).To(ContainElement("required_tool_bypass_notice"))
	})
})
