package connectors

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/cogito"
)

type recordingTelegramExecutor struct{ got *types.Job }

func (r *recordingTelegramExecutor) Execute(j *types.Job) *types.JobResult {
	r.got = j
	return j.Result
}

func TestTelegramExecutesTrackedJobIdentity(t *testing.T) {
	tracked := types.NewJob()
	executor := &recordingTelegramExecutor{}
	telegramExecuteJob(executor, tracked)
	if executor.got != tracked {
		t.Fatal("executor received a different job")
	}
}

type recordingTelegramBot struct {
	sends, edits []models.ParseMode
	texts        []string
	sendParams   []*bot.SendMessageParams
	deleted      []int
	nextID       int
}

type contextBlockingTelegramBot struct {
	recordingTelegramBot
	started  chan struct{}
	returned chan struct{}
}

func (b *contextBlockingTelegramBot) EditMessageText(ctx context.Context, _ *bot.EditMessageTextParams) (*models.Message, error) {
	close(b.started)
	<-ctx.Done()
	close(b.returned)
	return nil, ctx.Err()
}

func (b *recordingTelegramBot) SendMessage(_ context.Context, p *bot.SendMessageParams) (*models.Message, error) {
	b.sends = append(b.sends, p.ParseMode)
	b.sendParams = append(b.sendParams, p)
	b.texts = append(b.texts, p.Text)
	b.nextID++
	return &models.Message{ID: b.nextID}, nil
}
func (b *recordingTelegramBot) EditMessageText(_ context.Context, p *bot.EditMessageTextParams) (*models.Message, error) {
	b.edits = append(b.edits, p.ParseMode)
	b.texts = append(b.texts, p.Text)
	return &models.Message{ID: p.MessageID}, nil
}
func (b *recordingTelegramBot) DeleteMessage(_ context.Context, p *bot.DeleteMessageParams) (bool, error) {
	b.deleted = append(b.deleted, p.MessageID)
	return true, nil
}

func TestTelegramDeliveryClearResetsPlaceholderAndFallbackSendsAfterRich(t *testing.T) {
	tg := &Telegram{placeholders: map[string]int{}}
	b := &recordingTelegramBot{nextID: 10}
	d := tg.telegramDelivery(t.Context(), b, -1, 7, "job", 10)
	if err := d.clearPreview(t.Context(), -1); err != nil {
		t.Fatal(err)
	}
	if err := d.finalMarkdown(t.Context(), -1, []string{"later"}); err != nil {
		t.Fatal(err)
	}
	if len(b.deleted) != 1 || len(b.edits) != 0 || len(b.sends) != 1 {
		t.Fatalf("deleted/edits/sends = %v/%d/%d", b.deleted, len(b.edits), len(b.sends))
	}
	if b.sendParams[0].ReplyParameters == nil || b.sendParams[0].ReplyParameters.MessageID != 7 {
		t.Fatalf("fallback reply = %#v", b.sendParams[0].ReplyParameters)
	}
	if b.sendParams[0].LinkPreviewOptions == nil || b.sendParams[0].LinkPreviewOptions.IsDisabled == nil || !*b.sendParams[0].LinkPreviewOptions.IsDisabled {
		t.Fatalf("link previews not disabled: %#v", b.sendParams[0])
	}
}

func TestTelegramStreamingJobCancellationStillAllowsFinalDelivery(t *testing.T) {
	api := &telegramStreamAPI{}
	job, session := telegramNewJobWithStream(t.Context(), api, 1, true, telegramStreamDelivery{}, nil, "job", nil)
	defer session.Close()
	job.Cancel()
	if err := session.Finalize("completed answer", nil); err != nil {
		t.Fatalf("Finalize after normal job completion: %v", err)
	}
	_, finals := api.snapshot()
	if len(finals) != 1 || finals[0].RichMessage.Markdown != "completed answer" {
		t.Fatalf("finals = %#v, want persistent completed answer", finals)
	}
}

func TestTelegramDeliveryPreviewCancellationAbortsInflightLegacyEdit(t *testing.T) {
	tg := &Telegram{placeholders: map[string]int{"job": 10}}
	b := &contextBlockingTelegramBot{
		started:  make(chan struct{}),
		returned: make(chan struct{}),
	}
	delivery := tg.telegramDelivery(context.Background(), b, -1, 7, "job", 10)
	job, session := telegramNewJobWithStream(t.Context(), &telegramStreamAPI{}, -1, false, delivery, nil, "job", nil)
	defer session.Close()

	session.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "working"})
	select {
	case <-b.started:
	case <-time.After(time.Second):
		t.Fatal("legacy preview edit did not start")
	}

	job.Cancel()
	select {
	case <-b.returned:
	case <-time.After(time.Second):
		t.Fatal("preview cancellation did not unblock the legacy edit")
	}
}

type cancellingTelegramAPI struct {
	started  chan struct{}
	returned chan struct{}
	calls    atomic.Int32
}

func (a *cancellingTelegramAPI) sendRichMessageDraft(ctx context.Context, _ telegramRichMessageDraft) error {
	if a.calls.Add(1) == 1 {
		close(a.started)
	}
	<-ctx.Done()
	close(a.returned)
	return ctx.Err()
}
func (*cancellingTelegramAPI) sendRichMessage(context.Context, telegramRichMessage) error { return nil }

func TestTelegramTrackedJobCancellationAbortsInflightDraftRequest(t *testing.T) {
	api := &cancellingTelegramAPI{started: make(chan struct{}), returned: make(chan struct{})}
	job, session := telegramNewJobWithStream(t.Context(), api, 1, true, telegramStreamDelivery{}, nil, "job", nil)
	defer session.Close()
	select {
	case <-api.started:
	case <-time.After(time.Second):
		t.Fatal("draft request did not start")
	}
	job.Cancel()
	select {
	case <-api.returned:
	case <-time.After(time.Second):
		t.Fatal("in-flight draft request was not cancelled")
	}
	session.Accept(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "must not restart previews"})
	time.Sleep(telegramStreamInterval + 50*time.Millisecond)
	if got := api.calls.Load(); got != 1 {
		t.Fatalf("draft calls after job cancellation = %d, want 1", got)
	}
	select {
	case <-session.done:
		t.Fatal("job cancellation stopped final-delivery orchestration")
	default:
	}
	if err := session.Finalize("answer after cancellation", nil); err != nil {
		t.Fatalf("Finalize after cancellation: %v", err)
	}
}

type orderedTelegramAPI struct {
	mu     sync.Mutex
	events *[]string
	calls  int
}

func (*orderedTelegramAPI) sendRichMessageDraft(context.Context, telegramRichMessageDraft) error {
	return nil
}
func (a *orderedTelegramAPI) sendRichMessage(_ context.Context, m telegramRichMessage) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	*a.events = append(*a.events, "rich:"+m.RichMessage.Markdown)
	if a.calls == 2 {
		return errors.New("rejected")
	}
	return nil
}

type orderedTelegramBot struct {
	recordingTelegramBot
	events *[]string
}

func (b *orderedTelegramBot) SendMessage(ctx context.Context, p *bot.SendMessageParams) (*models.Message, error) {
	*b.events = append(*b.events, "send:"+p.Text)
	return b.recordingTelegramBot.SendMessage(ctx, p)
}
func (b *orderedTelegramBot) DeleteMessage(ctx context.Context, p *bot.DeleteMessageParams) (bool, error) {
	*b.events = append(*b.events, "delete")
	return b.recordingTelegramBot.DeleteMessage(ctx, p)
}

func TestTelegramMixedRichFallbackUsesRealDeliveryInOrder(t *testing.T) {
	var events []string
	tg := &Telegram{placeholders: map[string]int{"job": 10}}
	b := &orderedTelegramBot{recordingTelegramBot: recordingTelegramBot{nextID: 10}, events: &events}
	delivery := tg.telegramDelivery(t.Context(), b, -1, 7, "job", 10)
	api := &orderedTelegramAPI{events: &events}
	s := telegramFinalSession(t.Context(), api, -1, false, delivery)
	answer := strings.Repeat("a", telegramMaxMessageLength) + "second"
	if err := s.deliverFinal(answer, nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"delete", "rich:" + strings.Repeat("a", telegramMaxMessageLength), "rich:second", "send:second"}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("events = %#v, want %#v", events, want)
		}
	}
	if len(b.edits) != 0 {
		t.Fatalf("fallback edited old placeholder: %#v", b.edits)
	}
}

func TestTelegramDeliveryCreatesLazyPlaceholderWithoutIdenticalEditAndUsesMarkdownV2(t *testing.T) {
	tg := &Telegram{placeholders: map[string]int{}}
	b := &recordingTelegramBot{}
	d := tg.telegramDelivery(t.Context(), b, 1, 0, "job", 0)
	if err := d.editPreview(t.Context(), 1, "thinking"); err != nil {
		t.Fatal(err)
	}
	if len(b.sends) != 1 || len(b.edits) != 0 {
		t.Fatalf("sends/edits = %d/%d", len(b.sends), len(b.edits))
	}
	if err := d.finalMarkdown(t.Context(), 1, []string{"final"}); err != nil {
		t.Fatal(err)
	}
	if len(b.edits) != 1 || b.edits[0] != models.ParseModeMarkdown {
		t.Fatalf("parse modes = %#v", b.edits)
	}
}

func TestTelegramDeliveryLazyFinalCreationCarriesMarkdownV2ParseMode(t *testing.T) {
	tg := &Telegram{placeholders: map[string]int{}}
	b := &recordingTelegramBot{}
	d := tg.telegramDelivery(t.Context(), b, 1, 0, "job", 0)
	if err := d.finalMarkdown(t.Context(), 1, []string{"*final*"}); err != nil {
		t.Fatal(err)
	}
	if len(b.sends) != 1 || b.sends[0] != models.ParseModeMarkdown || len(b.edits) != 0 {
		t.Fatalf("send modes/edits = %#v/%d", b.sends, len(b.edits))
	}
}

func TestTelegramStreamingDefaultsEnabled(t *testing.T) {
	tg, err := NewTelegramConnector(map[string]string{"token": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !tg.streaming {
		t.Fatal("streaming = false, want true when omitted")
	}
}

func TestTelegramStreamingCanBeDisabled(t *testing.T) {
	tg, err := NewTelegramConnector(map[string]string{"token": "test", "streaming": "false"})
	if err != nil {
		t.Fatal(err)
	}
	if tg.streaming {
		t.Fatal("streaming = true, want false")
	}
}

func TestTelegramAskOptionsAttachMatchingSession(t *testing.T) {
	api := &telegramStreamAPI{}
	session := newTelegramStreamSession(t.Context(), api, 42, true, telegramStreamDelivery{})
	defer session.Close()

	job := types.NewJob(telegramAskOptions(nil, "job", map[string]any{"chatID": int64(42)}, session)...)
	if job.StreamCallback == nil {
		t.Fatal("request stream callback is nil")
	}
	job.StreamCallback(cogito.StreamEvent{Type: cogito.StreamEventContent, Content: "hello"})
	if err := session.Flush(); err != nil {
		t.Fatal(err)
	}
	drafts, _ := api.snapshot()
	if got := drafts[len(drafts)-1].RichMessage.Markdown; got != "hello" {
		t.Fatalf("preview = %q, want hello", got)
	}
}

func TestTelegramAskOptionsWithoutSessionDoesNotStream(t *testing.T) {
	job := types.NewJob(telegramAskOptions(nil, "job", nil, nil)...)
	if job.StreamCallback != nil {
		t.Fatal("request stream callback is set while streaming is disabled")
	}
}

func TestTelegramGroupFinalAttemptsRichBeforeFallback(t *testing.T) {
	api := &telegramStreamAPI{}
	session := telegramFinalSession(t.Context(), api, -42, false, telegramStreamDelivery{})
	if err := session.deliverFinal("**answer**", nil); err != nil {
		t.Fatal(err)
	}
	_, finals := api.snapshot()
	if len(finals) != 1 || finals[0].RichMessage.Markdown != "**answer**" {
		t.Fatalf("rich finals = %#v, want raw Markdown attempted once", finals)
	}
}

func TestTelegramFinalDeliveryContinuesAfterPreviewCleanupFailure(t *testing.T) {
	api := &telegramStreamAPI{}
	session := telegramFinalSession(t.Context(), api, -42, false, telegramStreamDelivery{
		clearPreview: func(context.Context, int64) error {
			return errors.New("delete failed")
		},
	})
	if err := session.deliverFinal("persistent answer", nil); err != nil {
		t.Fatalf("deliverFinal after cleanup failure: %v", err)
	}
	_, finals := api.snapshot()
	if len(finals) != 1 || finals[0].RichMessage.Markdown != "persistent answer" {
		t.Fatalf("finals = %#v, want persistent answer", finals)
	}
}
