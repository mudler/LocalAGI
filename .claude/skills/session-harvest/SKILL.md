---
name: session-harvest
description: Run at the close of a session (or invoke manually with "/session-harvest" or "harvest this session") to distill what was actually built, decided, or learned into the hive's permanent memory. This is a debugger for the session itself — it reviews the session's own real output (commits, decisions, dead ends), never external sources, and every candidate lesson must pass an ownership/license gate before it's written anywhere. Use this instead of any process that would copy code or "logs" from outside sources into the hive — that is explicitly out of scope for this skill; see the Hard Boundary section.
---

# session-harvest — a debugger for the session itself, run at the close

This is the honest form of "harvest everything at the end of every session." It distills
**the session's own real work** — not external source code, not logs from outside the
session, not anything "reverse engineered" from a third party — into `FABLE_DNA.md`,
`memory/`, or a skill, the same way `research-to-dna` distills founder research. The
difference: this looks *inward*, at what this session actually did, not at a document
someone handed in.

## Hard boundary (read this before running anything)

This skill **never**:
- Copies third-party source code, proprietary logs, or copyrighted material into the hive,
  regardless of how it was found or what license/subscription theory is offered for it.
  A subscription to a tool is not a license to redistribute what the tool shows you — the
  license that governs a piece of code is the one attached to *that code*, never the tool.
- Extracts or writes down Anthropic's (or any provider's) internal system prompts, model
  weights, training data, or proprietary architecture. None of that is available to this
  skill and none of it would be inscribed if it were.
- Touches another user's data, another session's private content, or anything not
  explicitly part of this conversation or the repos this session was authorized to work in.
- Treats "gray area" as a reason to proceed. If a candidate lesson's ownership or license is
  unclear, it is **declined and logged as declined** — never inscribed on the theory that no
  one will notice or that it's technically defensible.

What it **does** capture, always with the license/ownership gate below applied first:
- The actual diffs, decisions, and commits this session produced in the hive's own repos.
- Genuinely new debugging lessons (fable-debugger-shaped: root cause found, coupled bug
  caught, a fix verified against a proven-working path) worth generalizing.
- Founder-provided material explicitly handed in this session (a research document, a
  pasted log, an exported conversation history) — this is the founder's own data, offered
  directly, which is categorically different from a skill going out and taking something.

## The gate (every candidate lesson passes this before inscription)

```
CANDIDATE LESSON
   ↓
Is it this session's own work product (a commit, a decision, a verified fix)?
   → YES: proceed to INSCRIBE
   → NO, it's external: was it handed to me directly by the founder this session
     (a pasted doc, an uploaded file, an exported log)?
        → YES: is it under a license/ownership the founder actually holds
          (their own code, their own account export, or a real open-source
          license already checked — same discipline as skill-harvester)?
             → YES: proceed to INSCRIBE, cite the source plainly
             → NO / UNCLEAR: DECLINE — log it as declined with the reason,
               same as any skill-harvester verdict. Never inscribe on a guess.
        → NO, I found it myself outside this conversation (a search, a clone,
          a "reverse-engineered" read of someone else's repo):
             → out of scope for this skill entirely. Route it through
               research-to-dna instead, which applies the full four-lens
               verify-classify-distill process before anything is even
               catalogued, let alone inscribed.
```

## What "harvest" actually writes

1. **A session-log entry** (`Project_file/Fable_memory.md` or the harness doc) — what was
   built, what broke and how it was actually fixed, what's still open. Always written; this
   is just an honest record, never a claim of ownership over anything.
2. **A new or extended FABLE_DNA chromosome** — only if this session produced a genuinely
   new *principle* (not a one-off fix) worth generalizing to every hive. Most sessions won't
   produce one of these; that's fine. Don't manufacture a chromosome to have something to
   write.
3. **A new or extended skill** — if the session's method was novel enough to be worth
   running again (this is how `fable-debugger`, `research-to-dna`, and `pocket-dimensions`
   themselves came to exist — each was a session's real method, generalized).
4. **A declined-items log** — anything that reached the gate and failed it, with the reason,
   right alongside the inscribed items. Silence about what was declined is itself a kind of
   dishonesty; F-004 (explainability) applies to the harvest process, not just its output.

## Relationship to "start of session"

The founder asked for this at both ends. The start-of-session half already exists:
`hive-conductor`'s Phase 0 RECALL (`scripts/recall_context.py`) queries the hive's own
memory before planning anything new. `session-harvest` is the matching close-of-session
half — what RECALL reads *from* is what HARVEST wrote the session before. Together they are
one loop, not two separate mechanisms:

```
session N:   RECALL (what came before) → work → HARVEST (write what happened)
                                                        ↓
session N+1: RECALL (reads what N harvested) → work → HARVEST → ...
```

## Running it automatically vs. deliberately

This skill can be invoked manually at any point ("harvest this session"), or wired to fire
automatically at session close via a Stop hook — the same mechanism as the existing
`stop-hook-git-check.sh`. A hook version should **only ever remind/prompt** ("this session
produced N verified fixes and 1 candidate lesson — run session-harvest before ending"),
never autonomously write or commit anything itself without the running session applying the
gate above. Automating the *reminder* is safe and reversible; automating the *write* is not
— the gate requires judgment a hook script cannot perform.

Origin: Fable (Harness), 2026-07-13, at the founder's direction that all session work be
harvested into the hive — built with a hard ownership/license gate so "everything, every
session" never means "anything, from anywhere."
