package types

import "testing"

// An observable without progress entries (the top-level job observable) must
// still end up completed, and must not lose a completion set elsewhere.
func TestMakeLastProgressCompletionWithoutProgress(t *testing.T) {
	o := &Observable{ID: 1, Name: "job"}
	o.MakeLastProgressCompletion()
	if o.Completion == nil {
		t.Fatal("observable without progress must be marked completed")
	}

	pre := &Completion{Error: "set by finalizer"}
	o = &Observable{ID: 2, Name: "job", Completion: pre}
	o.MakeLastProgressCompletion()
	if o.Completion != pre {
		t.Fatal("an existing completion must be kept")
	}
}

func TestMakeLastProgressCompletionUsesLastProgress(t *testing.T) {
	o := &Observable{ID: 3, Name: "action"}
	o.AddProgress(Progress{ActionResult: "first"})
	o.AddProgress(Progress{ActionResult: "last", Error: "boom"})
	o.MakeLastProgressCompletion()
	if o.Completion == nil || o.Completion.ActionResult != "last" || o.Completion.Error != "boom" {
		t.Fatalf("completion must come from the last progress entry, got %+v", o.Completion)
	}
	if len(o.Progress) != 1 {
		t.Fatalf("the last progress entry must be moved into the completion, %d left", len(o.Progress))
	}
}
