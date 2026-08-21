package connectors

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mudler/cogito"
)

const telegramStreamInterval = 400 * time.Millisecond

type telegramStreamDelivery struct {
	editPreview   func(context.Context, int64, string) error
	finalMarkdown func(context.Context, int64, []string) error
	finalPlain    func(context.Context, int64, []string) error
}

type telegramStreamCommand struct {
	kind     uint8
	markdown string
	urls     []string
	done     chan error
}

type telegramStreamSession struct {
	ctx      context.Context
	cancel   context.CancelFunc
	api      telegramAPI
	chatID   int64
	private  bool
	draftID  int64
	delivery telegramStreamDelivery

	mu              sync.Mutex
	content         string
	version         uint64
	dirty           bool
	thinkingPending bool
	closed          bool
	wake            chan struct{}
	command         chan telegramStreamCommand
	done            chan struct{}
}

var telegramDraftSequence atomic.Int64

func newTelegramStreamSession(parent context.Context, api telegramAPI, chatID int64, private bool, delivery telegramStreamDelivery) *telegramStreamSession {
	ctx, cancel := context.WithCancel(parent)
	draftID := telegramDraftSequence.Add(1)
	if draftID == 0 {
		draftID = telegramDraftSequence.Add(1)
	}
	s := &telegramStreamSession{
		ctx: ctx, cancel: cancel, api: api, chatID: chatID, private: private,
		draftID: draftID, delivery: delivery, wake: make(chan struct{}, 1),
		command: make(chan telegramStreamCommand), done: make(chan struct{}), dirty: true, thinkingPending: true,
	}
	go s.run()
	s.signal()
	return s
}

func (s *telegramStreamSession) Accept(event cogito.StreamEvent) {
	if event.Type == cogito.StreamEventDone {
		_ = s.Flush()
		return
	}
	if event.Type != cogito.StreamEventContent || event.Content == "" {
		return
	}
	s.mu.Lock()
	if s.closed || s.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	s.content += event.Content
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

	for {
		select {
		case <-s.ctx.Done():
			return
		case command := <-s.command:
			if command.kind == 1 {
				if !time.Now().Before(retryUntil) {
					_, retry := s.deliverPreview()
					if retry > 0 {
						retryUntil = time.Now().Add(retry)
					}
				}
				command.done <- nil
			} else {
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
			attempted, retry := s.deliverPreview()
			if attempted {
				nextPreview = time.Now().Add(telegramStreamInterval)
			}
			if retry > 0 {
				retryUntil = time.Now().Add(retry)
				schedule(retryUntil)
			}
		case <-timerC:
			timerC = nil
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
		return telegramThinkingMessage, s.version, true, true
	}
	text := s.content
	if text == "" {
		text = telegramThinkingMessage
	}
	return text, s.version, true, false
}

func (s *telegramStreamSession) deliverPreview() (bool, time.Duration) {
	text, version, ok, thinking := s.previewSnapshot()
	if !ok {
		return false, 0
	}
	var err error
	if s.private {
		err = s.api.sendRichMessageDraft(s.ctx, telegramRichMessageDraft{ChatID: s.chatID, DraftID: s.draftID, RichMessage: telegramInputRichMessage{Markdown: text}})
		var apiErr *telegramAPIError
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			return true, time.Duration(apiErr.RetryAfter) * time.Second
		}
		if err != nil {
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
		return true, time.Duration(apiErr.RetryAfter) * time.Second
	}
	if err == nil {
		s.mu.Lock()
		if thinking {
			s.thinkingPending = false
		}
		if s.version == version && (!thinking || version == 0) {
			s.dirty = false
		}
		s.mu.Unlock()
	}
	return true, 0
}

func (s *telegramStreamSession) deliverFinal(markdown string, urls []string) error {
	formatted := telegramFormatResponse(markdown, urls, telegramMaxMessageLength)
	if s.private {
		for _, chunk := range formatted {
			if err := s.api.sendRichMessage(s.ctx, telegramRichMessage{ChatID: s.chatID, RichMessage: telegramInputRichMessage{Markdown: chunk}}); err == nil {
				continue
			}
			if s.delivery.finalMarkdown != nil {
				if err := s.delivery.finalMarkdown(s.ctx, s.chatID, []string{telegramMarkdownV2(chunk)}); err == nil {
					continue
				}
			}
			if s.delivery.finalPlain != nil {
				if err := s.delivery.finalPlain(s.ctx, s.chatID, []string{telegramPlainText(chunk)}); err != nil {
					return err
				}
			}
		}
		return nil
	}
	markdownV2 := make([]string, len(formatted))
	for i := range formatted {
		markdownV2[i] = telegramMarkdownV2(formatted[i])
	}
	if s.delivery.finalMarkdown != nil {
		if err := s.delivery.finalMarkdown(s.ctx, s.chatID, markdownV2); err == nil {
			return nil
		}
	}
	plain := make([]string, len(formatted))
	for i := range formatted {
		plain[i] = telegramPlainText(formatted[i])
	}
	if s.delivery.finalPlain != nil {
		return s.delivery.finalPlain(s.ctx, s.chatID, plain)
	}
	return nil
}
