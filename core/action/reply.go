package action

import (
    "github.com/mudler/LocalAGI/core/agent"
    "github.com/mudler/LocalAGI/core/memory"
)

func GenerateResponse(agent *agent.Agent, input string) string {
    memories, _ := memory.LoadMemories(agent.ID, input)
    // ... rest of implementation
}

func buildContext(memories []memory.MemoryEntry, input string) string {
    // ... implementation here
}