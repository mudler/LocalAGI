package connectors

import (
	"context"
	"testing"

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
	nextID       int
}

func (b *recordingTelegramBot) SendMessage(_ context.Context, p *bot.SendMessageParams) (*models.Message, error) {
	b.sends = append(b.sends, p.ParseMode)
	b.texts = append(b.texts, p.Text)
	b.nextID++
	return &models.Message{ID: b.nextID}, nil
}
func (b *recordingTelegramBot) EditMessageText(_ context.Context, p *bot.EditMessageTextParams) (*models.Message, error) {
	b.edits = append(b.edits, p.ParseMode)
	b.texts = append(b.texts, p.Text)
	return &models.Message{ID: p.MessageID}, nil
}
func (b *recordingTelegramBot) DeleteMessage(context.Context, *bot.DeleteMessageParams) (bool, error) {
	return true, nil
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
