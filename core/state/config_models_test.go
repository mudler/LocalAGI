package state

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgentConfig knowledge models", func() {
	It("round-trips embedding_model and reranker_model through JSON", func() {
		var config AgentConfig
		Expect(json.Unmarshal([]byte(`{"embedding_model":"embed","reranker_model":"rank"}`), &config)).To(Succeed())
		Expect(config.EmbeddingModel).To(Equal("embed"))
		Expect(config.RerankerModel).To(Equal("rank"))

		data, err := json.Marshal(config)
		Expect(err).ToNot(HaveOccurred())
		var roundTrip map[string]any
		Expect(json.Unmarshal(data, &roundTrip)).To(Succeed())
		Expect(roundTrip).To(HaveKeyWithValue("embedding_model", "embed"))
		Expect(roundTrip).To(HaveKeyWithValue("reranker_model", "rank"))
	})
})
