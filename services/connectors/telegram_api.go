package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const telegramBotAPIBaseURL = "https://api.telegram.org"

type telegramAPI interface {
	sendMessageDraft(context.Context, telegramMessageDraft) error
	sendRichMessage(context.Context, telegramRichMessage) error
}

type telegramInputRichMessage struct {
	Markdown string `json:"markdown"`
}

type telegramMessageDraft struct {
	ChatID  int64  `json:"chat_id"`
	DraftID int64  `json:"draft_id"`
	Text    string `json:"text"`
}

type telegramRichMessage struct {
	ChatID          int64                    `json:"chat_id"`
	RichMessage     telegramInputRichMessage `json:"rich_message"`
	ReplyParameters *telegramReplyParameters `json:"reply_parameters,omitempty"`
}

type telegramReplyParameters struct {
	MessageID int `json:"message_id"`
}

type telegramAPIError struct {
	Method      string
	ErrorCode   int
	Description string
	RetryAfter  int
}

func (e *telegramAPIError) Error() string {
	return fmt.Sprintf("telegram %s: %s", e.Method, e.Description)
}

type telegramHTTPAPI struct {
	token   string
	client  *http.Client
	baseURL string
}

func newTelegramHTTPAPI(token string, client *http.Client, baseURL string) telegramAPI {
	if client == nil {
		client = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = telegramBotAPIBaseURL
	}
	return &telegramHTTPAPI{
		token:   token,
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (a *telegramHTTPAPI) sendMessageDraft(ctx context.Context, input telegramMessageDraft) error {
	if input.DraftID == 0 {
		return fmt.Errorf("telegram sendMessageDraft: draft_id must be nonzero")
	}
	return a.call(ctx, "sendMessageDraft", input)
}

func (a *telegramHTTPAPI) sendRichMessage(ctx context.Context, input telegramRichMessage) error {
	return a.call(ctx, "sendRichMessage", input)
}

type telegramAPIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func (a *telegramHTTPAPI) call(ctx context.Context, method string, input any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return a.safeError(method, "encode request", err)
	}

	endpoint := a.baseURL + "/bot" + a.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return a.safeError(method, "create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return a.safeError(method, "send request", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return a.safeError(method, "read response", err)
	}

	var result telegramAPIResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return a.safeError(method, "decode response", err)
	}
	if !result.OK {
		description := a.redact(result.Description)
		if description == "" {
			description = http.StatusText(resp.StatusCode)
		}
		return &telegramAPIError{
			Method:      method,
			ErrorCode:   result.ErrorCode,
			Description: description,
			RetryAfter:  result.Parameters.RetryAfter,
		}
	}

	return nil
}

func (a *telegramHTTPAPI) safeError(method, action string, err error) error {
	return fmt.Errorf("telegram %s: %s: %s", method, action, a.redact(err.Error()))
}

func (a *telegramHTTPAPI) redact(value string) string {
	if a.token == "" {
		return value
	}
	return strings.ReplaceAll(value, a.token, "[REDACTED]")
}
