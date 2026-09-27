package state

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgentPool.GetCollectionConfig", func() {
	var pool *AgentPool

	BeforeEach(func() {
		pool = &AgentPool{pool: AgentPoolData{
			"Research": {EmbeddingModel: "research-embed", RerankerModel: "research-rank"},
		}}
	})

	DescribeTable("resolves normalized collection names to the agent",
		func(name string) {
			config, err := pool.GetCollectionConfig(name)
			Expect(err).ToNot(HaveOccurred())
			Expect(config).ToNot(BeNil())
			Expect(config.EmbeddingModel).To(Equal("research-embed"))
			Expect(config.RerankerModel).To(Equal("research-rank"))
		},
		Entry("exact", "Research"),
		Entry("lowercase", "research"),
		Entry("padded uppercase", " RESEARCH "),
	)

	It("returns nil without error for an unknown collection", func() {
		config, err := pool.GetCollectionConfig("missing")
		Expect(err).ToNot(HaveOccurred())
		Expect(config).To(BeNil())
	})

	It("fails on an ambiguous normalized name but prefers an exact match", func() {
		pool.pool["RESEARCH"] = AgentConfig{EmbeddingModel: "other"}

		config, err := pool.GetCollectionConfig("research")
		Expect(err).To(HaveOccurred())
		Expect(config).To(BeNil())

		config, err = pool.GetCollectionConfig("Research")
		Expect(err).ToNot(HaveOccurred())
		Expect(config).ToNot(BeNil())
		Expect(config.EmbeddingModel).To(Equal("research-embed"))
	})
})
