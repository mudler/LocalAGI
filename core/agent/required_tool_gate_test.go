package agent

import (
	"strings"
	"testing"
)

func TestRequiredToolResultOK(t *testing.T) {
	cases := []struct {
		name   string
		result string
		want   bool
	}{
		{"clean ok true", `{"ok": true, "flags": []}`, true},
		{"clean ok false", `{"ok": false, "flags": [{"severity":"high"}]}`, false},
		{"compact ok true", `{"ok":true}`, true},
		{"lenient wrapped ok true", `tool output: {"ok": true, "summary": {}} done`, true},
		{"garbage", `not json at all`, false},
		{"empty", ``, false},
		{"ok false substring", `{"ok": false}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := requiredToolResultOK(c.result); got != c.want {
				t.Fatalf("requiredToolResultOK(%q) = %v, want %v", c.result, got, c.want)
			}
		})
	}
}

func TestRequiredToolGate(t *testing.T) {
	const max = 3

	t.Run("not available -> allow", func(t *testing.T) {
		n := 0
		if _, blocked := requiredToolGate(false, false, "nudge", &n, max); blocked {
			t.Fatal("grounding not available must not block")
		}
		if n != 0 {
			t.Fatalf("attempts must not change, got %d", n)
		}
	})

	t.Run("already passed -> allow", func(t *testing.T) {
		n := 0
		if _, blocked := requiredToolGate(true, true, "nudge", &n, max); blocked {
			t.Fatal("passed grounding must not block")
		}
		if n != 0 {
			t.Fatalf("attempts must not change, got %d", n)
		}
	})

	t.Run("available and not passed -> block with adjustment", func(t *testing.T) {
		n := 0
		decision, blocked := requiredToolGate(true, false, "nudge", &n, max)
		if !blocked {
			t.Fatal("must block until the required tool passes")
		}
		if !decision.Approved {
			t.Fatal("must keep Approved=true so cogito re-runs selection (not abort the run)")
		}
		if decision.Adjustment == "" {
			t.Fatal("expected a non-empty adjustment naming the required tool")
		}
		if n != 1 {
			t.Fatalf("expected attempt counter 1, got %d", n)
		}
	})

	t.Run("bounded: allows through after max attempts", func(t *testing.T) {
		n := 0
		for i := 0; i < max; i++ {
			if _, blocked := requiredToolGate(true, false, "nudge", &n, max); !blocked {
				t.Fatalf("attempt %d should still block", i+1)
			}
		}
		if _, blocked := requiredToolGate(true, false, "nudge", &n, max); blocked {
			t.Fatal("after max attempts the answer must be allowed through (no unbounded loop)")
		}
		if n != max {
			t.Fatalf("attempts should cap at %d, got %d", max, n)
		}
	})
}

func TestTextFinalizationNeedsRequiredTool(t *testing.T) {
	const max = 3
	cases := []struct {
		name              string
		available, passed bool
		attempts          int
		role, content     string
		want              bool
	}{
		{"grounding unavailable", false, false, 0, "assistant", "answer", false},
		{"already passed", true, true, 0, "assistant", "answer", false},
		{"max attempts reached (graceful bypass)", true, false, max, "assistant", "answer", false},
		{"last message is a tool call, not text", true, false, 0, "tool", "", false},
		{"empty text answer", true, false, 0, "assistant", "   ", false},
		{"text finalization without grounding -> gate", true, false, 0, "assistant", "here is my answer", true},
		{"still under max -> gate", true, false, max - 1, "assistant", "answer", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := textFinalizationNeedsRequiredTool(c.available, c.passed, c.attempts, max, c.role, c.content); got != c.want {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

func TestRequiredFinishPromptFor(t *testing.T) {
	t.Run("default names the tool", func(t *testing.T) {
		got := requiredFinishPromptFor("check_policy", "")
		if !strings.Contains(got, "check_policy") {
			t.Fatalf("default prompt must name the tool, got %q", got)
		}
	})
	t.Run("override wins", func(t *testing.T) {
		if got := requiredFinishPromptFor("check_policy", "do the thing"); got != "do the thing" {
			t.Fatalf("override must be used verbatim, got %q", got)
		}
	})
}

// The gate must be OFF unless someone asks for it: an agent without the option, and an
// agent whose required tool is not bound, must never be blocked. That is what makes the
// feature safe to enable pool-wide.
func TestRequiredToolGateIsOffByDefault(t *testing.T) {
	var o options
	if o.requiredFinishTool != "" {
		t.Fatalf("gate must be disabled by default, got %q", o.requiredFinishTool)
	}
	n := 0
	if _, blocked := requiredToolGate(false, false, "nudge", &n, defaultRequiredFinishAttempts); blocked {
		t.Fatal("a tool that is not bound to the agent must never block")
	}
}

func TestWithRequiredToolBeforeFinish(t *testing.T) {
	var o options
	for _, opt := range []Option{
		WithRequiredToolBeforeFinish("check_policy"),
		WithRequiredToolBeforeFinishPrompt("call it first"),
		WithRequiredToolBeforeFinishAttempts(5),
	} {
		if err := opt(&o); err != nil {
			t.Fatalf("option returned %v", err)
		}
	}
	if o.requiredFinishTool != "check_policy" {
		t.Fatalf("tool = %q", o.requiredFinishTool)
	}
	if o.requiredFinishPrompt != "call it first" {
		t.Fatalf("prompt = %q", o.requiredFinishPrompt)
	}
	if o.requiredFinishAttempts != 5 {
		t.Fatalf("attempts = %d", o.requiredFinishAttempts)
	}
}
