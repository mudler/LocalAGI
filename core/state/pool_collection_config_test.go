package state

import "testing"

func TestGetCollectionConfig(t *testing.T) {
	pool := &AgentPool{pool: AgentPoolData{
		"Research": {EmbeddingModel: "research-embed", RerankerModel: "research-rank"},
	}}
	for _, name := range []string{"Research", "research", " RESEARCH "} {
		config, err := pool.GetCollectionConfig(name)
		if err != nil || config == nil || config.EmbeddingModel != "research-embed" || config.RerankerModel != "research-rank" {
			t.Fatalf("lookup %q: config=%+v error=%v", name, config, err)
		}
	}
	if config, err := pool.GetCollectionConfig("missing"); config != nil || err != nil {
		t.Fatalf("missing lookup: config=%+v error=%v", config, err)
	}
	pool.pool["RESEARCH"] = AgentConfig{EmbeddingModel: "other"}
	if config, err := pool.GetCollectionConfig("research"); err == nil || config != nil {
		t.Fatalf("ambiguous lookup: config=%+v error=%v", config, err)
	}
	config, err := pool.GetCollectionConfig("Research")
	if err != nil || config == nil || config.EmbeddingModel != "research-embed" {
		t.Fatalf("exact match must win: config=%+v error=%v", config, err)
	}
}
