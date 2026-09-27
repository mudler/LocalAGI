package state_test

import (
	"encoding/json"

	"github.com/mudler/LocalAGI/core/state"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgentConfig tool lists", func() {
	decode := func(body string) (state.AgentConfig, error) {
		var c state.AgentConfig
		err := json.Unmarshal([]byte(body), &c)
		return c, err
	}

	It("reads JSON arrays", func() {
		c, err := decode(`{"allowed_tools":["a"," b ",""],"excluded_tools":["c"]}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(c.AllowedTools).To(Equal([]string{"a", "b"}))
		Expect(c.ExcludedTools).To(Equal([]string{"c"}))
	})

	It("reads the comma or newline separated string the agent form submits", func() {
		c, err := decode(`{"allowed_tools":"a, b\nc\r\n,","excluded_tools":""}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(c.AllowedTools).To(Equal([]string{"a", "b", "c"}))
		Expect(c.ExcludedTools).To(BeEmpty())
	})

	It("leaves the lists empty when they are absent", func() {
		c, err := decode(`{"name":"x"}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(c.AllowedTools).To(BeEmpty())
		Expect(c.ExcludedTools).To(BeEmpty())
	})

	It("rejects values that are not tool names", func() {
		_, err := decode(`{"allowed_tools":[1]}`)
		Expect(err).To(HaveOccurred())
		_, err = decode(`{"excluded_tools":{"a":true}}`)
		Expect(err).To(HaveOccurred())
	})

	It("round-trips through MarshalJSON", func() {
		in := state.AgentConfig{Name: "x", AllowedTools: []string{"a", "b"}, ExcludedTools: []string{"c"}}
		data, err := json.Marshal(&in)
		Expect(err).ToNot(HaveOccurred())
		out, err := decode(string(data))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.AllowedTools).To(Equal(in.AllowedTools))
		Expect(out.ExcludedTools).To(Equal(in.ExcludedTools))
	})

	It("describes both fields for the agent form", func() {
		meta := state.NewAgentConfigMeta(nil, nil, nil, nil)
		names := map[string]string{}
		for _, f := range meta.Fields {
			names[f.Name] = string(f.Type)
		}
		Expect(names).To(HaveKeyWithValue("allowed_tools", "textarea"))
		Expect(names).To(HaveKeyWithValue("excluded_tools", "textarea"))
	})
})
