package agent

type AgentState struct {
    ID         string           `json:"id"`
    Memories   []MemoryEntry    `json:"memories"`
    CurrentContext map[string]interface{} `json:"current_context"`
    // ...existing fields
}