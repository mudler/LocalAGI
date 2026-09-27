package common

import (
	"fmt"
	"strings"

	"github.com/mudler/LocalAGI/core/types"
)

const (
	// MaxParamsLen is the maximum length for params in "Calling tool X with parameters: ..."
	MaxParamsLen = 400
	// MaxResultLen is the maximum length for result in "Result of X: ..."
	MaxResultLen = 500
)

// StatusAccumulator holds accumulated status lines for a job's placeholder message.
// Callers (connectors) are responsible for mutex and for clearing when the job ends.
type StatusAccumulator struct {
	lines []string
}

// NewStatusAccumulator returns a new accumulator.
func NewStatusAccumulator() *StatusAccumulator {
	return &StatusAccumulator{lines: nil}
}

// AppendReasoning appends a "Current thought: ..." line when reasoning is non-empty.
func (a *StatusAccumulator) AppendReasoning(reasoning string) {
	if reasoning == "" {
		return
	}
	a.lines = append(a.lines, "Current thought process:\n"+reasoning)
}

// AppendToolCall appends a "Calling tool X with parameters: ..." line (params truncated).
func (a *StatusAccumulator) AppendToolCall(actionName string, params string) {
	if actionName == "" {
		actionName = "Tool"
	}
	truncated := Truncate(params, MaxParamsLen)
	a.lines = append(a.lines, fmt.Sprintf("Calling tool `%s` with parameters: %s", actionName, truncated))
}

// AppendToolResult appends a "Result of X: ..." line (result truncated).
func (a *StatusAccumulator) AppendToolResult(actionName string, result string) {
	if actionName == "" {
		actionName = "Tool"
	}
	truncated := Truncate(result, MaxResultLen)
	a.lines = append(a.lines, fmt.Sprintf("Result of `%s`: %s", actionName, truncated))
}

// BuildMessage returns thinkingPrefix + "\n\n" + joined lines, truncated to maxTotalLen if needed.
// If over the limit, the message is truncated from the start (oldest content dropped) so the latest lines stay visible.
func (a *StatusAccumulator) BuildMessage(thinkingPrefix string, maxTotalLen int) string {
	if len(a.lines) == 0 {
		return thinkingPrefix
	}
	body := strings.Join(a.lines, "\n\n")
	full := thinkingPrefix + "\n\n" + body
	if maxTotalLen <= 0 || len(full) <= maxTotalLen {
		return full
	}
	// Keep prefix and truncate from the start of the body
	available := maxTotalLen - len(thinkingPrefix) - 2 // 2 for "\n\n"
	if available <= 0 {
		return Truncate(full, maxTotalLen)
	}
	if len(body) <= available {
		return full
	}
	// Drop oldest lines until we fit
	for i := 0; i < len(a.lines); i++ {
		trimmed := strings.Join(a.lines[i:], "\n\n")
		if len(trimmed) <= available {
			return thinkingPrefix + "\n\n" + trimmed
		}
	}
	// Single line too long
	return thinkingPrefix + "\n\n" + Truncate(body, available)
}

// Truncate returns s truncated to maxLen with "..." suffix if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ActionDisplayName returns the action's display name for status messages, or "Tool" if nil.
func ActionDisplayName(action types.Action) string {
	if action == nil {
		return "Tool"
	}
	return action.Definition().Name.String()
}
