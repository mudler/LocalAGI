package memory

import (
    "time"
    "github.com/mudler/LocalAGI/core/agent"
    "github.com/mudler/LocalAGI/pkg/localrecall"
)

type MemoryEntry struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
    Context   map[string]interface{} `json:"context"`
}

func SaveMemory(agentID string, entry MemoryEntry) error {
    return localrecall.Store(agentID, entry.ID, entry.Content, entry.Context)
}

func LoadMemories(agentID, query string) ([]MemoryEntry, error) {
    results, err := localrecall.Query(agentID, query)
    if err != nil {
        return nil, err
    }
    memories := make([]MemoryEntry, len(results))
    for i, r := range results {
        memories[i] = MemoryEntry{
            ID:        r.ID,
            Content:   r.Content,
            Timestamp: r.Timestamp,
            Context:   r.Metadata,
        }
    }
    return memories, nil
}