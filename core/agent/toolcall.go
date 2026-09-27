package agent

// lastToolCallName returns the name of the tool call that produced the
// conversation's closing tool result, reporting false when the conversation
// does not end in one or when the call that produced it cannot be recovered.
//
// Both guards are load-bearing. A tool result needs a message before it to read
// the call from, and that message can carry an empty ToolCalls slice: when the
// model returns no tool selection — after a context-window overflow, say — the
// fragment still ends in a tool role. Reading ToolCalls[0] there panics, and
// because consumeJob runs on its own goroutine with no recover, that panic ends
// the process rather than the job.
func lastToolCallName(messages Messages) (string, bool) {
	if len(messages) < 2 {
		return "", false
	}

	if messages[len(messages)-1].Role != "tool" {
		return "", false
	}

	calls := messages[len(messages)-2].ToolCalls
	if len(calls) == 0 {
		return "", false
	}

	return calls[0].Function.Name, true
}
