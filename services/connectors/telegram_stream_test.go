package connectors

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mudler/cogito"
)

type telegramStreamAPI struct {
	mu       sync.Mutex
	drafts   []telegramRichMessageDraft
	finals   []telegramRichMessage
	draftErr func(int) error
	inCall   atomic.Int32
	maxCalls atomic.Int32
	block    time.Duration
}

func (a *telegramStreamAPI) sendRichMessageDraft(_ context.Context, draft telegramRichMessageDraft) error {
	n := a.inCall.Add(1)
	defer a.inCall.Add(-1)
	for old := a.maxCalls.Load(); n > old && !a.maxCalls.CompareAndSwap(old, n); old = a.maxCalls.Load() {
	}
	if a.block > 0 {
		time.Sleep(a.block)
	}
	a.mu.Lock()
	a.drafts = append(a.drafts, draft)
	i := len(a.drafts)
	a.mu.Unlock()
	if a.draftErr != nil {
		return a.draftErr(i)
	}
	return nil
}

func (a *telegramStreamAPI) sendRichMessage(_ context.Context, final telegramRichMessage) error {
	n := a.inCall.Add(1)
	defer a.inCall.Add(-1)
	for old := a.maxCalls.Load(); n > old && !a.maxCalls.CompareAndSwap(old, n); old = a.maxCalls.Load() {
	}
	a.mu.Lock()
	a.finals = append(a.finals, final)
	a.mu.Unlock()
	return nil
}

func (a *telegramStreamAPI) snapshot() ([]telegramRichMessageDraft, []telegramRichMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]telegramRichMessageDraft(nil), a.drafts...), append([]telegramRichMessage(nil), a.finals...)
}

func waitTelegramStream(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for stream delivery")
}

func TestTelegramStreamPrivateUsesStableDraftAndRateLimitsSerializedCalls(t *testing.T) {
	api := &telegramStreamAPI{block: 25 * time.Millisecond}
	s := newTelegramStreamSession(context.Background(), api, 42, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })

	for _, delta := range []string{"one", " two", " three"} {
		s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: delta})
	}
	time.Sleep(200 * time.Millisecond)
	if drafts, _ := api.snapshot(); len(drafts) != 1 {
		t.Fatalf("calls before interval = %d, want 1", len(drafts))
	}
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 2 })
	drafts, _ := api.snapshot()
	if drafts[0].DraftID == 0 || drafts[1].DraftID != drafts[0].DraftID {
		t.Fatalf("draft IDs = %d, %d, want same nonzero ID", drafts[0].DraftID, drafts[1].DraftID)
	}
	if drafts[0].RichMessage.Markdown != telegramThinkingMessage || drafts[1].RichMessage.Markdown != "one two three" {
		t.Fatalf("drafts = %#v", drafts)
	}
	if api.maxCalls.Load() != 1 {
		t.Fatalf("maximum concurrent API calls = %d, want 1", api.maxCalls.Load())
	}
}

func TestTelegramStreamRetryAfterRetainsLatestPreview(t *testing.T) {
	api := &telegramStreamAPI{draftErr: func(i int) error {
		if i == 2 {
			return &telegramAPIError{Method: "sendRichMessageDraft", ErrorCode: 429, RetryAfter: 1}
		}
		return nil
	}}
	s := newTelegramStreamSession(context.Background(), api, 9, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "first"})
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 2 })
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: " latest"})
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 3 })
	drafts, _ := api.snapshot()
	if got := drafts[2].RichMessage.Markdown; got != "first latest" {
		t.Fatalf("retried preview = %q", got)
	}
}

func TestTelegramStreamNativeFailureFallsBackOnlyForThatSession(t *testing.T) {
	failing := &telegramStreamAPI{draftErr: func(int) error { return errors.New("unsupported") }}
	healthy := &telegramStreamAPI{}
	var mu sync.Mutex
	edits := map[int64][]string{}
	hooks := telegramStreamDelivery{editPreview: func(_ context.Context, chatID int64, text string) error {
		mu.Lock()
		edits[chatID] = append(edits[chatID], text)
		mu.Unlock()
		return nil
	}}
	a := newTelegramStreamSession(context.Background(), failing, 1, true, hooks)
	b := newTelegramStreamSession(context.Background(), healthy, 2, true, hooks)
	defer a.Close()
	defer b.Close()
	waitTelegramStream(t, func() bool { d, _ := healthy.snapshot(); return len(d) == 1 })
	a.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "fallback"})
	b.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "native"})
	waitTelegramStream(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(edits[1]) > 0 && edits[1][len(edits[1])-1] == "fallback"
	})
	waitTelegramStream(t, func() bool { d, _ := healthy.snapshot(); return len(d) >= 2 })
	mu.Lock()
	otherEdits := len(edits[2])
	mu.Unlock()
	if otherEdits != 0 {
		t.Fatalf("healthy session used edit fallback %d times", otherEdits)
	}
}

func TestTelegramStreamFlushAndFinalizePromptlyBypassPreviewRetry(t *testing.T) {
	api := &telegramStreamAPI{draftErr: func(i int) error {
		if i == 2 {
			return &telegramAPIError{ErrorCode: 429, RetryAfter: 10}
		}
		return nil
	}}
	s := newTelegramStreamSession(context.Background(), api, 5, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "answer"})
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 2 })
	start := time.Now()
	if err := s.Finalize("answer", nil); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("final delivery waited for retry_after")
	}
	_, finals := api.snapshot()
	if len(finals) != 1 || finals[0].RichMessage.Markdown != "answer" {
		t.Fatalf("finals = %#v", finals)
	}
}

func TestTelegramStreamCancelAndCloseStopLaterDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	api := &telegramStreamAPI{}
	s := newTelegramStreamSession(ctx, api, 7, true, telegramStreamDelivery{})
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })
	cancel()
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "never"})
	s.Close()
	drafts, _ := api.snapshot()
	time.Sleep(500 * time.Millisecond)
	after, _ := api.snapshot()
	if len(after) != len(drafts) {
		t.Fatalf("calls after close = %d, before = %d", len(after), len(drafts))
	}
	select {
	case <-s.done:
	default:
		t.Fatal("worker did not terminate")
	}
}

func TestTelegramStreamGroupEditsPlaceholder(t *testing.T) {
	api := &telegramStreamAPI{}
	got := make(chan string, 2)
	s := newTelegramStreamSession(context.Background(), api, -10, false, telegramStreamDelivery{editPreview: func(_ context.Context, _ int64, text string) error { got <- text; return nil }})
	defer s.Close()
	select {
	case text := <-got:
		if text != telegramThinkingMessage {
			t.Fatalf("initial edit = %q", text)
		}
	case <-time.After(time.Second):
		t.Fatal("missing initial edit")
	}
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "group answer"})
	select {
	case text := <-got:
		if text != "group answer" {
			t.Fatalf("content edit = %q", text)
		}
	case <-time.After(time.Second):
		t.Fatal("missing content edit")
	}
}
