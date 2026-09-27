# Telegram rich streaming design

## Summary

The Telegram connector streams agent output as Telegram renders it. Private chats use native message drafts. Other chats use progressive message edits. The connector sends the final answer as a persistent rich message.

Streaming is enabled by default. Operators can disable it with the connector setting `streaming=false`.

## Goals

- Show answer text while the model generates it.
- Render the model's Markdown in Telegram.
- Match the native streaming behavior of Hermes Agent.
- Keep final delivery reliable when a streaming or formatting API is unavailable.
- Isolate simultaneous responses so that one job cannot receive another job's output.
- Preserve the existing reasoning and tool-status feedback.

## Non-goals

- Stream generated audio, images, or songs.
- Add rich-message media blocks.
- Change the format of conversation history.
- Change connectors other than Telegram.
- Guarantee native drafts in groups. Telegram limits draft methods to private chats.

## User experience

### Private chats

The connector starts a native draft with a Telegram thinking block. It appends answer deltas to the draft as the model produces them. Telegram animates updates that use the same nonzero draft ID.

The connector sends the completed answer with `sendRichMessage`. The persistent message uses the model's raw Markdown. The ephemeral draft then expires according to Telegram behavior.

### Groups and supergroups

The connector sends the existing thinking placeholder as a reply. It edits that message as answer deltas arrive. The final edit contains formatted content when Telegram accepts it.

### Reasoning and tool activity

Before answer content starts, the preview can show reasoning and tool activity. Consecutive reasoning deltas append to the visible reasoning buffer; they never replace the preceding delta. Tool status events replace the status text because they describe state transitions rather than token fragments. Answer content takes priority after the first content delta. The connector does not expose hidden reasoning that the agent does not already publish.

### Disabled streaming

If `streaming=false`, the connector keeps the thinking placeholder and sends only the completed answer. Rich Markdown formatting remains enabled.

## Architecture

### Per-request stream callback

The agent API gains a per-request stream callback option. `Ask` passes this callback to the Cogito request without replacing the agent's existing callback.

The agent invokes both callbacks for each event:

- The existing agent callback continues to publish SSE events.
- The request callback receives events for one Telegram job.

This boundary prevents concurrent jobs from sharing an unscoped global callback. It also prevents Telegram integration from disabling web UI streaming.

### Telegram stream session

Each incoming Telegram request creates one stream session. The session owns these values:

- Chat ID and chat type.
- Draft ID or placeholder message ID.
- Accumulated answer content.
- The most recent delivered content.
- Delivery mode: native draft, edited message, or final-only.
- Throttle timer and cancellation context.

The request callback only appends event data and signals the session worker. One worker performs Telegram API calls in order. This structure avoids concurrent edits and protects the model callback from network latency.

The handler closes and flushes the session before final delivery. Cleanup stops its timer and releases its goroutine on success, cancellation, or error.

### Telegram Bot API client

The pinned `go-telegram/bot` version does not expose the Bot API 10.2 rich-message methods. A small internal client calls these methods through the Bot API HTTP endpoint:

- `sendMessageDraft`
- `sendRichMessage`

The client uses typed request and response structures. It uses the request context and returns Telegram's error description. Existing operations continue to use `go-telegram/bot`.

The client interface stays private to the connector package. Tests replace it with a fake implementation.

## Streaming policy

The session coalesces character and token deltas. It sends at most one preview update every 400 milliseconds. This interval gives visible progress without one API request per token.

The worker skips an update when its content equals the last delivered content. It flushes pending content when the agent reports completion or `Ask` returns.

Telegram drafts expire after 30 seconds. The worker sends an unchanged heartbeat before expiry when generation remains active. The heartbeat uses the same draft ID.

The connector respects Telegram flood-control responses. If Telegram returns a retry delay, the worker delays the next preview update. Preview retries never delay final delivery beyond the request context.

## Markdown and delivery fallback

Rich delivery passes the unescaped response in `InputRichMessage.markdown`. This mode supports GitHub-style Markdown where Telegram supports it. It also supports headings, lists, tables, code blocks, quotations, details, footnotes, and formulas.

The connector does not call `bot.EscapeMarkdown` on rich Markdown. The current escape step removes formatting and must not remain on this path.

Delivery uses this fallback order:

1. Send a native text draft in a private chat.
2. If the draft fails, switch that response to progressive edits.
3. Finalize with a rich persistent message or rich edit.
4. If rich formatting fails, convert the content to Telegram MarkdownV2.
5. If MarkdownV2 fails, send plain text without a parse mode.

A preview failure changes only the active response. The connector can try native drafts again for the next response.

The URL appendix becomes ordinary Markdown and stays part of the raw response. Link previews remain disabled for streamed and final output where the API supports that option.

## Message limits

Rich messages support more text than legacy messages, but the connector still handles every Telegram limit. A splitter operates on UTF-8 text and preserves complete code fences when possible.

For a response that requires multiple persistent messages:

- The first message finalizes the active preview.
- The connector sends later chunks in order.
- Every chunk uses the same formatting fallback order.
- Group chunks retain the reply relationship where Telegram permits it.

The streaming preview shows the current tail within the applicable draft or edit limit. The accumulated session retains the complete response for final delivery.

## Error handling

- An empty final response replaces the preview with the existing internal-error text.
- A cancelled job stops preview updates and keeps the content already shown.
- A preview API failure logs the method and Telegram description without logging the bot token.
- A final rich-delivery failure falls back to MarkdownV2, then plain text.
- A total final-delivery failure returns through the existing connector error path.
- Multimedia and text-to-speech behavior remains unchanged.

## Configuration

`TelegramConfigMeta` adds this field:

- `streaming`: A boolean that enables progressive Telegram output. The default is `true` when the field is absent.

No token or BotFather setting is required. Unsupported Bot API servers use the fallback path.

## Testing

Unit tests use named, table-driven subtests where cases share a contract. Tests cover these behaviors:

- Private chats start and update one native draft with a stable draft ID.
- Consecutive reasoning deltas accumulate instead of showing one token at a time.
- Private chats send one persistent rich final response.
- Groups edit one placeholder instead of creating a draft.
- Character deltas coalesce into throttled updates.
- A final flush includes pending content.
- Concurrent jobs do not mix their content.
- The request callback does not replace the SSE callback.
- Draft failure switches only the current response to message edits.
- Rich final failure falls back to MarkdownV2 and then plain text.
- Raw Markdown reaches the rich-message client without escaping.
- Long UTF-8 responses split without data loss.
- Cancellation stops timers and workers.
- `streaming=false` suppresses preview updates but keeps rich final formatting.

HTTP contract tests use `httptest.Server` to inspect Bot API payloads. They do not contact Telegram. Relevant package tests also run with the race detector.

## Documentation

The Telegram section in `README.md` documents the default streaming behavior and the `streaming` setting. It states that native drafts require a current Telegram Bot API. It also describes the automatic edit and formatting fallbacks.

## Acceptance criteria

- A private-chat user sees an animated rich preview while the model writes.
- A group-chat user sees the reply message update while the model writes.
- The final answer renders supported Markdown instead of showing Markdown source.
- Streaming is active when the configuration omits `streaming`.
- Setting `streaming=false` restores final-only output.
- An unsupported rich-message API does not prevent final delivery.
- Simultaneous Telegram jobs do not mix content, and existing SSE delivery continues unchanged.
- The test suite and race-enabled connector tests pass.
