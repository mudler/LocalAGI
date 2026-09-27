package types

import (
	"errors"
	"testing"

	"github.com/sashabaranov/go-openai"
)

// A job finished twice must not panic (close of closed channel); the first
// error wins and the finalizers run exactly once.
func TestJobResultFinishIsIdempotent(t *testing.T) {
	j := NewJobResult()
	calls := 0
	j.AddFinalizer(func(c []openai.ChatCompletionMessage) { calls++ })
	j.Finish(errors.New("first"))
	j.Finish(nil) // second call: must be a no-op
	if j.Error == nil || j.Error.Error() != "first" {
		t.Fatalf("first error must win, got %v", j.Error)
	}
	if calls != 1 {
		t.Fatalf("finalizers must run once, ran %d times", calls)
	}
	select {
	case <-j.ready:
	default:
		t.Fatal("ready channel must be closed after Finish")
	}
}
