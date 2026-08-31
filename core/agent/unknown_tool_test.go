package agent

import (
	"strings"
	"testing"

	"github.com/mudler/LocalAGI/core/action"
)

// TestCorrectUnknownToolCall covers the self-correction logic used when the
// model invokes a tool name that matches no available action.
func TestCorrectUnknownToolCall(t *testing.T) {
	available := []string{"openregister_search", "openregister_company"}
	const max = 3

	t.Run("built-in control verbs are not treated as unknown", func(t *testing.T) {
		for _, name := range []string{action.StopActionName, action.ConversationActionName, action.StateActionName} {
			attempts := map[string]int{}
			if _, corrected := correctUnknownToolCall(name, available, attempts, max); corrected {
				t.Fatalf("control verb %q should not be corrected", name)
			}
			if len(attempts) != 0 {
				t.Fatalf("control verb %q should not consume a correction attempt", name)
			}
		}
	})

	t.Run("unknown tool yields an adjustment listing the available tools", func(t *testing.T) {
		attempts := map[string]int{}
		decision, corrected := correctUnknownToolCall("list_collections", available, attempts, max)
		if !corrected {
			t.Fatal("unknown tool should be corrected")
		}
		if !decision.Approved {
			t.Fatal("adjustment must keep Approved=true, otherwise cogito aborts the run instead of re-querying")
		}
		if decision.Skip {
			t.Fatal("first attempts should adjust, not skip")
		}
		if decision.Adjustment == "" {
			t.Fatal("expected a non-empty adjustment")
		}
		if !strings.Contains(decision.Adjustment, "list_collections") {
			t.Errorf("adjustment should name the offending tool, got: %q", decision.Adjustment)
		}
		for _, name := range available {
			if !strings.Contains(decision.Adjustment, name) {
				t.Errorf("adjustment should list available tool %q, got: %q", name, decision.Adjustment)
			}
		}
		if attempts["list_collections"] != 1 {
			t.Errorf("expected attempt counter 1, got %d", attempts["list_collections"])
		}
	})

	t.Run("repeated hallucination is skipped after max attempts", func(t *testing.T) {
		attempts := map[string]int{}
		for i := 0; i < max; i++ {
			decision, _ := correctUnknownToolCall("ghost_tool", available, attempts, max)
			if decision.Skip {
				t.Fatalf("attempt %d should still adjust, not skip", i+1)
			}
		}
		decision, corrected := correctUnknownToolCall("ghost_tool", available, attempts, max)
		if !corrected {
			t.Fatal("over-limit unknown tool should still be handled")
		}
		if !decision.Skip {
			t.Fatal("after max attempts the call must be skipped to avoid an unbounded loop")
		}
		if decision.Approved {
			t.Fatal("a skipped call should not also be approved")
		}
	})

	t.Run("attempt counters are per-tool-name", func(t *testing.T) {
		attempts := map[string]int{}
		correctUnknownToolCall("tool_a", available, attempts, max)
		correctUnknownToolCall("tool_b", available, attempts, max)
		if attempts["tool_a"] != 1 || attempts["tool_b"] != 1 {
			t.Fatalf("counters should be independent per name, got %v", attempts)
		}
	})
}
