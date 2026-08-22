package connectors

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mudler/cogito"
	"github.com/mudler/xlog"
)

const telegramStreamInterval = 400 * time.Millisecond
const telegramDraftHeartbeatInterval = 25 * time.Second
const telegramStreamCursor = " ▉"

type telegramStreamDelivery struct {
	editPreview   func(context.Context, int64, string) error
	finalMarkdown func(context.Context, int64, []string) error
	finalPlain    func(context.Context, int64, []string) error
	clearPreview  func(context.Context, int64) error
	replyTo       int
}

type telegramStreamCommand struct {
	kind     uint8
	markdown string
	urls     []string
	done     chan error
}

type telegramStreamSession struct {
	ctx       context.Context
	finalCtx  context.Context
	cancel    context.CancelFunc
	api       telegramAPI
	chatID    int64
	private   bool
	parentCtx context.Context
	draftID   int64
	delivery  telegramStreamDelivery

	mu              sync.Mutex
	content         string
	status          string
	reasoningActive bool
	version         uint64
	dirty           bool
	thinkingPending bool
	closed          bool
	wake            chan struct{}
	command         chan telegramStreamCommand
	done            chan struct{}
	heartbeat       time.Duration
	lastDraftAt     time.Time
}

var telegramDraftSequence atomic.Int64

func newTelegramStreamSession(parent context.Context, api telegramAPI, chatID int64, private bool, delivery telegramStreamDelivery) *telegramStreamSession {
	return newTelegramStreamSessionWithHeartbeat(parent, api, chatID, private, delivery, telegramDraftHeartbeatInterval)
}

func newTelegramStreamSessionWithHeartbeat(parent context.Context, api telegramAPI, chatID int64, private bool, delivery telegramStreamDelivery, heartbeat time.Duration) *telegramStreamSession {
	return newTelegramStreamSessionWithContexts(parent, parent, api, chatID, private, delivery, heartbeat)
}

func newTelegramStreamSessionWithContexts(finalParent, previewParent context.Context, api telegramAPI, chatID int64, private bool, delivery telegramStreamDelivery, heartbeat time.Duration) *telegramStreamSession {
	finalCtx, cancelFinal := context.WithCancel(finalParent)
	ctx, cancelPreview := context.WithCancel(previewParent)
	stopPreviewLink := context.AfterFunc(finalCtx, cancelPreview)
	draftID := telegramDraftSequence.Add(1)
	if draftID == 0 {
		draftID = telegramDraftSequence.Add(1)
	}
	s := &telegramStreamSession{
		ctx: ctx, finalCtx: finalCtx, parentCtx: previewParent, cancel: func() {
			stopPreviewLink()
			cancelPreview()
			cancelFinal()
		}, api: api, chatID: chatID, private: private, heartbeat: heartbeat,
		draftID: draftID, delivery: delivery, wake: make(chan struct{}, 1),
		command: make(chan telegramStreamCommand), done: make(chan struct{}), dirty: true, thinkingPending: true,
	}
	go s.run()
	s.signal()
	return s
}

func (s *telegramStreamSession) Accept(event cogito.StreamEvent) {
	if event.Type == cogito.StreamEventDone {
		return
	}
	if event.Type != cogito.StreamEventContent && event.Type != cogito.StreamEventReasoning && event.Type != cogito.StreamEventStatus && event.Type != cogito.StreamEventToolCall && event.Type != cogito.StreamEventToolResult {
		return
	}
	s.mu.Lock()
	if s.closed || s.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if event.Type == cogito.StreamEventContent && event.Content != "" {
		s.content += event.Content
	} else if s.content == "" {
		status := telegramStreamStatus(event)
		if event.Type == cogito.StreamEventReasoning && s.reasoningActive {
			s.status += status
		} else {
			s.status = status
		}
		s.reasoningActive = event.Type == cogito.StreamEventReasoning
	}
	s.version++
	s.dirty = true
	s.mu.Unlock()
	s.signal()
}

func (s *telegramStreamSession) Flush() error {
	return s.execute(telegramStreamCommand{kind: 1, done: make(chan error, 1)})
}

func (s *telegramStreamSession) Finalize(markdown string, urls []string) error {
	return s.execute(telegramStreamCommand{kind: 2, markdown: markdown, urls: urls, done: make(chan error, 1)})
}

func (s *telegramStreamSession) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		<-s.done
		return
	}
	s.closed = true
	s.mu.Unlock()
	s.cancel()
	<-s.done
}

func (s *telegramStreamSession) execute(command telegramStreamCommand) error {
	select {
	case s.command <- command:
	case <-s.done:
		return s.ctx.Err()
	}
	select {
	case err := <-command.done:
		return err
	case <-s.done:
		return s.ctx.Err()
	}
}

func (s *telegramStreamSession) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *telegramStreamSession) run() {
	defer close(s.done)
	var timer *time.Timer
	var timerC <-chan time.Time
	var nextPreview time.Time
	var retryUntil time.Time
	var flushWaiters []chan error
	previewDone := s.ctx.Done()
	previewStopped := false
	stopTimer := func() {
		if timer != nil && !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timerC = nil
	}
	schedule := func(at time.Time) {
		d := time.Until(at)
		if d < 0 {
			d = 0
		}
		if timer == nil {
			timer = time.NewTimer(d)
		} else {
			stopTimer()
			timer.Reset(d)
		}
		timerC = timer.C
	}
	defer stopTimer()
	finishFlushes := func() {
		s.mu.Lock()
		dirty := s.dirty
		s.mu.Unlock()
		if dirty {
			return
		}
		for _, waiter := range flushWaiters {
			waiter <- nil
		}
		flushWaiters = nil
	}
	schedulePending := func() {
		s.mu.Lock()
		dirty := s.dirty
		lastDraftAt := s.lastDraftAt
		s.mu.Unlock()
		if !dirty {
			finishFlushes()
			if s.private && !lastDraftAt.IsZero() && s.heartbeat > 0 {
				schedule(lastDraftAt.Add(s.heartbeat))
			}
			return
		}
		when := nextPreview
		if retryUntil.After(when) {
			when = retryUntil
		}
		schedule(when)
	}

	for {
		select {
		case <-s.finalCtx.Done():
			return
		case <-previewDone:
			previewDone = nil
			previewStopped = true
			stopTimer()
			s.mu.Lock()
			s.dirty = false
			s.mu.Unlock()
			for _, waiter := range flushWaiters {
				waiter <- s.ctx.Err()
			}
			flushWaiters = nil
		case command := <-s.command:
			if command.kind == 1 {
				if previewStopped {
					command.done <- s.ctx.Err()
					continue
				}
				if time.Now().Before(retryUntil) {
					command.done <- errors.New("Telegram preview pending retry")
					continue
				}
				attempted, retry, err := s.deliverPreview()
				if attempted {
					nextPreview = time.Now().Add(telegramStreamInterval)
				}
				if retry > 0 {
					retryUntil = time.Now().Add(retry)
					command.done <- errors.New("Telegram preview pending retry")
					continue
				}
				if err != nil {
					command.done <- err
					schedulePending()
					continue
				}
				flushWaiters = append(flushWaiters, command.done)
				schedulePending()
			} else {
				stopTimer()
				s.mu.Lock()
				s.dirty = false
				s.mu.Unlock()
				command.done <- s.deliverFinal(command.markdown, command.urls)
			}
		case <-s.wake:
			now := time.Now()
			when := nextPreview
			if retryUntil.After(when) {
				when = retryUntil
			}
			if when.After(now) {
				schedule(when)
				continue
			}
			attempted, retry, err := s.deliverPreview()
			if err != nil && retry == 0 && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				xlog.Warn("Telegram preview delivery failed", "chat_id", s.chatID, "error", err)
			}
			if attempted {
				nextPreview = time.Now().Add(telegramStreamInterval)
			}
			if retry > 0 {
				retryUntil = time.Now().Add(retry)
			}
			schedulePending()
		case <-timerC:
			timerC = nil
			s.mu.Lock()
			if s.private && !s.dirty && !s.lastDraftAt.IsZero() && s.heartbeat > 0 {
				s.dirty = true
			}
			s.mu.Unlock()
			s.signal()
		}
	}
}

func (s *telegramStreamSession) previewSnapshot() (string, uint64, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty {
		return "", 0, false, false
	}
	if s.thinkingPending {
		return telegramThinkingMessage + telegramStreamCursor, s.version, true, true
	}
	text := s.content
	if text == "" {
		text = s.status
		if text == "" {
			text = telegramThinkingMessage
		}
	}
	limit := telegramMaxMessageLength - len([]rune(telegramStreamCursor))
	return telegramPreviewTail(text, limit) + telegramStreamCursor, s.version, true, false
}

func telegramPreviewTail(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[len(runes)-limit:])
}

func telegramStreamStatus(event cogito.StreamEvent) string {
	if event.Content != "" {
		return event.Content
	}
	if event.ToolName != "" {
		return "Using " + event.ToolName + "…"
	}
	if event.Type == cogito.StreamEventToolResult {
		return "Tool completed…"
	}
	return ""
}

func (s *telegramStreamSession) deliverPreview() (bool, time.Duration, error) {
	if s.ctx.Err() != nil {
		return false, 0, nil
	}
	text, version, ok, thinking := s.previewSnapshot()
	if !ok {
		return false, 0, nil
	}
	var err error
	if s.private {
		err = s.api.sendMessageDraft(s.ctx, telegramMessageDraft{ChatID: s.chatID, DraftID: s.draftID, Text: text})
		var apiErr *telegramAPIError
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			return true, time.Duration(apiErr.RetryAfter) * time.Second, err
		}
		if err != nil {
			if s.ctx.Err() != nil {
				return true, 0, err
			}
			xlog.Warn("Telegram sendMessageDraft failed; falling back to editable message", "chat_id", s.chatID, "error", err)
			s.private = false
			if s.delivery.editPreview != nil {
				err = s.delivery.editPreview(s.ctx, s.chatID, text)
			}
		}
	} else if s.delivery.editPreview != nil {
		err = s.delivery.editPreview(s.ctx, s.chatID, text)
	}
	var apiErr *telegramAPIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return true, time.Duration(apiErr.RetryAfter) * time.Second, err
	}
	if err == nil {
		s.mu.Lock()
		if s.private {
			s.lastDraftAt = time.Now()
		}
		if thinking {
			s.thinkingPending = false
		}
		if s.version == version && (!thinking || version == 0) {
			s.dirty = false
		}
		s.mu.Unlock()
	}
	return true, 0, err
}

func (s *telegramStreamSession) deliverFinal(markdown string, urls []string) error {
	formatted := telegramFormatResponse(markdown, urls, telegramMaxMessageLength)
	if s.delivery.clearPreview != nil {
		_ = s.delivery.clearPreview(s.finalCtx, s.chatID)
	}
	failedAt := -1
	for i, chunk := range formatted {
		final := telegramRichMessage{ChatID: s.chatID, RichMessage: telegramInputRichMessage{Markdown: chunk}}
		if !s.private && s.delivery.replyTo != 0 {
			final.ReplyParameters = &telegramReplyParameters{MessageID: s.delivery.replyTo}
		}
		if err := s.api.sendRichMessage(s.finalCtx, final); err == nil {
			continue
		}
		failedAt = i
		break
	}
	if failedAt < 0 {
		return nil
	}
	remaining := formatted[failedAt:]
	markdownChunks := make([]string, len(remaining))
	for i, chunk := range remaining {
		markdownChunks[i] = telegramMarkdownV2(chunk)
	}
	if s.delivery.finalMarkdown != nil && s.delivery.finalMarkdown(s.finalCtx, s.chatID, markdownChunks) == nil {
		return nil
	}
	plainChunks := make([]string, len(remaining))
	for i, chunk := range remaining {
		plainChunks[i] = telegramPlainText(chunk)
	}
	if s.delivery.finalPlain != nil {
		return s.delivery.finalPlain(s.finalCtx, s.chatID, plainChunks)
	}
	return nil
}
