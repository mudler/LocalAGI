package connectors

import (
	"testing"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/cogito"
)

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
