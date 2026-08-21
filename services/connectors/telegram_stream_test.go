package connectors

import (
	"context"
	"errors"
	"runtime"
	"strings"
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
	finalErr func(int) error
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
	i := len(a.finals)
	a.mu.Unlock()
	if a.finalErr != nil {
		return a.finalErr(i)
	}
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

func TestTelegramStreamAlwaysDeliversThinkingBeforeImmediateContent(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	api := &telegramStreamAPI{}
	s := newTelegramStreamSession(context.Background(), api, 42, true, telegramStreamDelivery{})
	defer s.Close()
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "immediate"})

	waitTelegramStream(t, func() bool { drafts, _ := api.snapshot(); return len(drafts) == 2 })
	drafts, _ := api.snapshot()
	if got := drafts[0].RichMessage.Markdown; got != telegramThinkingMessage {
		t.Fatalf("initial draft = %q, want thinking draft", got)
	}
	if got := drafts[1].RichMessage.Markdown; got != "immediate" {
		t.Fatalf("content draft = %q, want immediate content without another event", got)
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

func TestTelegramStreamEditRetryAfterReschedulesPendingPreviewWithoutNewContent(t *testing.T) {
	api := &telegramStreamAPI{}
	var calls atomic.Int32
	got := make(chan string, 2)
	s := newTelegramStreamSession(context.Background(), api, -9, false, telegramStreamDelivery{editPreview: func(_ context.Context, _ int64, text string) error {
		if calls.Add(1) == 1 {
			return &telegramAPIError{Method: "editMessageText", ErrorCode: 429, RetryAfter: 1}
		}
		got <- text
		return nil
	}})
	defer s.Close()

	select {
	case text := <-got:
		if text != telegramThinkingMessage {
			t.Fatalf("retried preview = %q, want thinking preview", text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending preview was not retried after edit retry_after")
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("edit calls = %d, want 2", got)
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

func TestTelegramStreamFlushDeliversPendingContentBeforeReturning(t *testing.T) {
	api := &telegramStreamAPI{}
	s := newTelegramStreamSession(context.Background(), api, 5, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })

	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "pending answer"})
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	drafts, _ := api.snapshot()
	if len(drafts) != 2 {
		t.Fatalf("drafts when Flush returned = %d, want pending content delivered", len(drafts))
	}
	if got := drafts[1].RichMessage.Markdown; got != "pending answer" {
		t.Fatalf("flushed preview = %q, want pending answer", got)
	}
}

func TestTelegramStreamFlushReportsPendingRetryPromptly(t *testing.T) {
	api := &telegramStreamAPI{draftErr: func(i int) error {
		if i == 2 {
			return &telegramAPIError{ErrorCode: 429, RetryAfter: 1}
		}
		return nil
	}}
	s := newTelegramStreamSession(context.Background(), api, 5, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })

	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "pending after retry"})
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 2 })
	start := time.Now()
	if err := s.Flush(); err == nil {
		t.Fatal("Flush error = nil, want pending preview error")
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("Flush waited for retry_after")
	}
}

func TestTelegramStreamPreviewUsesUTF8SafeTail(t *testing.T) {
	api := &telegramStreamAPI{}
	s := newTelegramStreamSession(context.Background(), api, 5, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })
	full := strings.Repeat("🙂", telegramMaxMessageLength+10)
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: full})
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	drafts, _ := api.snapshot()
	got := drafts[len(drafts)-1].RichMessage.Markdown
	if len([]rune(got)) != telegramMaxMessageLength || got != strings.Repeat("🙂", telegramMaxMessageLength) {
		t.Fatalf("preview rune length = %d, want tail of %d", len([]rune(got)), telegramMaxMessageLength)
	}
	if s.content != full {
		t.Fatal("preview truncation discarded final content")
	}
}

func TestTelegramStreamPrivateShowsPublishedStatusBeforeContent(t *testing.T) {
	api := &telegramStreamAPI{}
	s := newTelegramStreamSession(context.Background(), api, 5, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventReasoning, Content: "checking sources"})
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	drafts, _ := api.snapshot()
	if got := drafts[len(drafts)-1].RichMessage.Markdown; got != "checking sources" {
		t.Fatalf("status = %q", got)
	}
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "answer"})
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	drafts, _ = api.snapshot()
	if got := drafts[len(drafts)-1].RichMessage.Markdown; got != "answer" {
		t.Fatalf("answer = %q", got)
	}
}

func TestTelegramStreamDoneEventIsNonblocking(t *testing.T) {
	api := &telegramStreamAPI{block: 500 * time.Millisecond}
	s := newTelegramStreamSession(context.Background(), api, 5, true, telegramStreamDelivery{})
	defer s.Close()
	waitTelegramStream(t, func() bool { return api.inCall.Load() == 1 })

	returned := make(chan struct{})
	go func() {
		s.Accept(cogito.StreamEvent{Type: cogito.StreamEventDone})
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Done event blocked on stream delivery")
	}
}

func TestTelegramStreamCancelAndCloseStopPendingThrottledDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	api := &telegramStreamAPI{}
	s := newTelegramStreamSession(ctx, api, 7, true, telegramStreamDelivery{})
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) == 1 })
	s.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "pending"})
	time.Sleep(100 * time.Millisecond)
	if drafts, _ := api.snapshot(); len(drafts) != 1 {
		t.Fatalf("drafts before throttle elapsed = %d, want 1", len(drafts))
	}

	cancel()
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

func TestTelegramStreamLongPrivateFinalUsesRichMarkdownForEveryChunkInOrder(t *testing.T) {
	api := &telegramStreamAPI{}
	s := newTelegramStreamSession(context.Background(), api, 42, true, telegramStreamDelivery{})
	defer s.Close()

	markdown := strings.Repeat("a", telegramMaxMessageLength) + strings.Repeat("b", 17)
	if err := s.Finalize(markdown, nil); err != nil {
		t.Fatal(err)
	}
	_, finals := api.snapshot()
	if len(finals) != 2 {
		t.Fatalf("rich final calls = %d, want 2", len(finals))
	}
	if got := finals[0].RichMessage.Markdown; got != strings.Repeat("a", telegramMaxMessageLength) {
		t.Fatalf("first rich chunk length/content = %d/%q", len(got), got[:min(len(got), 20)])
	}
	if got := finals[1].RichMessage.Markdown; got != strings.Repeat("b", 17) {
		t.Fatalf("second rich chunk = %q", got)
	}
}

func TestTelegramStreamLongPrivateFinalFallsBackWithoutLosingOrReorderingChunks(t *testing.T) {
	api := &telegramStreamAPI{finalErr: func(i int) error {
		if i == 1 {
			return errors.New("rich markdown rejected")
		}
		return nil
	}}
	var markdownChunks, plainChunks []string
	s := newTelegramStreamSession(context.Background(), api, 42, true, telegramStreamDelivery{
		finalMarkdown: func(_ context.Context, _ int64, chunks []string) error {
			markdownChunks = append([]string(nil), chunks...)
			return errors.New("MarkdownV2 rejected")
		},
		finalPlain: func(_ context.Context, _ int64, chunks []string) error {
			plainChunks = append([]string(nil), chunks...)
			return nil
		},
	})
	defer s.Close()

	markdown := strings.Repeat("a", telegramMaxMessageLength) + strings.Repeat("b", 17)
	if err := s.Finalize(markdown, nil); err != nil {
		t.Fatal(err)
	}
	_, finals := api.snapshot()
	if len(finals) != 1 {
		t.Fatalf("rich final calls = %d, want rich delivery to stop at first failure", len(finals))
	}
	wantMarkdown := []string{strings.Repeat("a", telegramMaxMessageLength), strings.Repeat("b", 17)}
	if len(markdownChunks) != 2 || markdownChunks[0] != wantMarkdown[0] || markdownChunks[1] != wantMarkdown[1] {
		t.Fatalf("MarkdownV2 fallback chunks lost or reordered: lengths %d, %d", len(markdownChunks), len(plainChunks))
	}
	wantPlain := []string{strings.Repeat("a", telegramMaxMessageLength), strings.Repeat("b", 17)}
	if len(plainChunks) != 2 || plainChunks[0] != wantPlain[0] || plainChunks[1] != wantPlain[1] {
		t.Fatalf("plain fallback chunks lost or reordered: %#v", plainChunks)
	}
}

func TestTelegramStreamNativeDraftHeartbeatAndClose(t *testing.T) {
	api := &telegramStreamAPI{}
	s := newTelegramStreamSessionWithHeartbeat(context.Background(), api, 42, true, telegramStreamDelivery{}, 20*time.Millisecond)
	waitTelegramStream(t, func() bool { d, _ := api.snapshot(); return len(d) >= 2 })
	drafts, _ := api.snapshot()
	if drafts[0].DraftID != drafts[1].DraftID || drafts[0].RichMessage.Markdown != drafts[1].RichMessage.Markdown {
		t.Fatalf("heartbeats changed draft: %#v", drafts[:2])
	}
	s.Close()
	n := len(drafts)
	time.Sleep(50 * time.Millisecond)
	after, _ := api.snapshot()
	if len(after) != n {
		t.Fatalf("heartbeat continued after close: %d -> %d", n, len(after))
	}
}

func TestTelegramStreamGroupDoesNotHeartbeat(t *testing.T) {
	var calls atomic.Int32
	s := newTelegramStreamSessionWithHeartbeat(context.Background(), &telegramStreamAPI{}, -1, false, telegramStreamDelivery{editPreview: func(context.Context, int64, string) error { calls.Add(1); return nil }}, 20*time.Millisecond)
	defer s.Close()
	waitTelegramStream(t, func() bool { return calls.Load() == 1 })
	time.Sleep(60 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("group heartbeat calls = %d", calls.Load())
	}
}
