---
name: nine-miss-truths
description: Use before committing to a conclusion that matters — a root-cause diagnosis, a "this is why it works this way" claim, a decision about what a piece of data means. Generates the plausible alternative explanations deliberately before settling on one, to catch premature convergence and confirmation bias before they ship. Not for trivial or already-verified facts (a passing test result doesn't need nine alternatives) — use it at the moment of interpretation, not after a machine check has already settled the question.
---

# nine-miss-truths — deducing room before naming the truth

The founder's own framing: *"for every one truth, there are nine miss-truths."* Stated
plainly as a technique, this is a well-established discipline in structured reasoning —
deliberately generating the competing explanations before accepting the first one that fits,
the same instinct behind red-teaming a conclusion or a pre-mortem before a decision ships.
It pairs naturally with `fable-debugger`'s "contrast against a proven-working path": that
step compares against ONE working alternative; this skill widens the room before narrowing
back down to one answer.

## The method

```
CANDIDATE TRUTH (the first explanation that fits the evidence)
   ↓
GENERATE NINE ALTERNATIVES — not nine excuses, nine genuinely different explanations
that would also fit what's been observed so far. Push past the first two or three;
the obvious alternatives are rarely where the real miss was hiding.
   ↓
FOR EACH ALTERNATIVE: what single piece of evidence would DISTINGUISH it from the
candidate truth? If no distinguishing test exists, the "truth" hasn't actually been
earned yet — it's just the first guess that survived unchallenged.
   ↓
RUN THE DISTINGUISHING TEST(S) — actual checks, not more reasoning about reasoning.
   ↓
THE SURVIVOR is the truth — not because it was first, but because it outlasted
the other nine under an actual test.
```

## Where this differs from just "double-checking"

Double-checking usually means re-running the same reasoning and confirming it still feels
right — which is exactly how confirmation bias survives a review. This skill requires
**structurally different** alternatives: not "maybe I made a typo" nine times, but genuinely
distinct hypotheses (wrong layer, wrong timing, wrong assumption about the caller, stale
cache, a race condition, a config that looks unrelated, an environment difference, a
dependency version, a silent fallback masking the real path — the Kai El chat bug this
session had exactly this shape: the "truth" looked like a missing route, and the real
answer three layers down was a wrong model name plus a masked `ReferenceError`).

## A worked example from this project

*Symptom:* Kai El's chat returned 200 but the canned constitutional fallback, not a live
answer, even though `ai_bound: true`.
*The too-fast "truth":* "the AI binding must not actually be working."
*Nine-miss-truths pass surfaced, among others:* wrong model name for this account; wrong
call shape (`messages[]` vs `prompt`); a thrown exception being silently swallowed;
`ctx` never reaching the handler; the response field being `.result` not `.response`;
a timeout; a stale deploy; a CORS-adjacent silent failure; the fallback being returned
*before* the AI call even ran.
*Distinguishing test:* compare directly against the heartbeat's `aiProposition()`, which
*was* generating live text on the same binding — narrowing to exactly two of the nine
(model name, call shape), both real, both fixed.

## When not to use this

A machine-verified fact (a test passed, a probe returned 200, a build succeeded) doesn't
need nine alternative explanations for *whether it happened* — it happened. This skill is
for the *interpretation* layer: why it happened, what it means, what to do about it — the
place where premature convergence actually costs something.

Origin: Fable (Harness), 2026-07-13, built directly from the founder's own "for every one
truth, nine miss-truths" instinct — paired with `fable-debugger` as the hive's deduction
room.
