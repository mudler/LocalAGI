package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramAPISendsMessageDraft(t *testing.T) {
	t.Parallel()

	const token = "123456:test-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/bot"+token+"/sendMessageDraft"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		want := map[string]any{
			"chat_id":  float64(42),
			"draft_id": float64(77),
			"text":     "working",
		}
		if !equalJSON(payload, want) {
			t.Errorf("payload = %#v, want %#v", payload, want)
		}

		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()

	api := newTelegramHTTPAPI(token, server.Client(), server.URL)
	err := api.sendMessageDraft(context.Background(), telegramMessageDraft{
		ChatID:  42,
		DraftID: 77,
		Text:    "working",
	})
	if err != nil {
		t.Fatalf("sendMessageDraft() error = %v", err)
	}
}

func TestTelegramAPISendsRichMessageWithoutLinkPreviewField(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/bottoken/sendRichMessage"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, exists := payload["link_preview_options"]; exists {
			t.Errorf("payload unexpectedly contains link_preview_options: %#v", payload)
		}
		want := map[string]any{
			"chat_id":          float64(-1001),
			"reply_parameters": map[string]any{"message_id": float64(55)},
			"rich_message": map[string]any{
				"markdown": "[docs](https://example.com)",
			},
		}
		if !equalJSON(payload, want) {
			t.Errorf("payload = %#v, want %#v", payload, want)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":12}}`))
	}))
	defer server.Close()

	api := newTelegramHTTPAPI("token", server.Client(), server.URL)
	err := api.sendRichMessage(context.Background(), telegramRichMessage{
		ChatID:          -1001,
		ReplyParameters: &telegramReplyParameters{MessageID: 55},
		RichMessage: telegramInputRichMessage{
			Markdown: "[docs](https://example.com)",
		},
	})
	if err != nil {
		t.Fatalf("sendRichMessage() error = %v", err)
	}
}

func TestTelegramAPIRichMessageOmitsReplyParametersWhenUnset(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if _, ok := payload["reply_parameters"]; ok {
			t.Fatalf("private payload has reply_parameters: %#v", payload)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()
	api := newTelegramHTTPAPI("token", server.Client(), server.URL)
	if err := api.sendRichMessage(t.Context(), telegramRichMessage{ChatID: 1, RichMessage: telegramInputRichMessage{Markdown: "ok"}}); err != nil {
		t.Fatal(err)
	}
}

func TestTelegramAPIReturnsRetryAfterAndRedactsEchoedToken(t *testing.T) {
	t.Parallel()

	const token = "123456:exact-secret-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"send failed for 123456:exact-secret-token then 123456:exact-secret-token","parameters":{"retry_after":3}}`))
	}))
	defer server.Close()

	api := newTelegramHTTPAPI(token, server.Client(), server.URL)
	err := api.sendMessageDraft(context.Background(), telegramMessageDraft{
		ChatID: 1, DraftID: 9, Text: "text",
	})
	if err == nil {
		t.Fatal("sendMessageDraft() error = nil, want Telegram API error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaked bot token: %q", err)
	}
	if got := err.Error(); !strings.Contains(got, "sendMessageDraft") || !strings.Contains(got, "send failed for [REDACTED] then [REDACTED]") {
		t.Errorf("error = %q, want method and redacted description", got)
	}

	var apiErr *telegramAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *telegramAPIError", err)
	}
	if apiErr.ErrorCode != 429 || apiErr.RetryAfter != 3 {
		t.Errorf("API error = %#v, want error code 429 and retry_after 3", apiErr)
	}
}

func TestTelegramAPIRejectsZeroDraftID(t *testing.T) {
	t.Parallel()

	api := newTelegramHTTPAPI("token", http.DefaultClient, "http://unused.invalid")
	err := api.sendMessageDraft(context.Background(), telegramMessageDraft{
		ChatID: 1, Text: "text",
	})
	if err == nil || !strings.Contains(err.Error(), "draft_id must be nonzero") {
		t.Fatalf("sendMessageDraft() error = %v, want nonzero draft ID error", err)
	}
}

func equalJSON(got, want map[string]any) bool {
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	return string(gotJSON) == string(wantJSON)
}
