func GenerateResponse(agent *agent.Agent, input string) string {
	// Retrieve relevant memories
	memories, _ := memory.LoadMemories(agent.ID, input)
	context := buildContext(memories, input)
	
	// Use context to inform response
	response := llm.Generate(context)
	return response
}

func buildContext(memories []memory.MemoryEntry, input string) string {
	var context strings.Builder
	context.WriteString("User Input: " + input + "\n\n")
	context.WriteString("Relevant Memories:\n")
	for _, m := range memories {
		context.WriteString(fmt.Sprintf("- %s\n", m.Content))
	}
	return context.String()
}