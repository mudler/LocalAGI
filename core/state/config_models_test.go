package state

import (
	"encoding/json"
	"testing"
)

func TestAgentConfigKnowledgeModelsRoundTrip(t *testing.T) {
	var config AgentConfig
	if err := json.Unmarshal([]byte(`{"embedding_model":"embed","reranker_model":"rank"}`), &config); err != nil {
		t.Fatal(err)
	}
	if config.EmbeddingModel != "embed" || config.RerankerModel != "rank" {
		t.Fatalf("lost models: %+v", config)
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip map[string]any
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip["embedding_model"] != "embed" || roundTrip["reranker_model"] != "rank" {
		t.Fatalf("models not serialized: %s", data)
	}
}
