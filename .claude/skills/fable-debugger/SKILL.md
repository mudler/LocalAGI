---
name: fable-debugger
description: The Fable-level self-debugger, persistently in the hive. Use when something in the hive is broken, degraded, or "works but not the way it was envisioned" — a 404, a silent fallback, a stale deploy, a test that lies. Encodes Fable's debugging method (Chromosome II of FABLE_DNA.md) as a runnable discipline any model can follow: probe before claim, compare against a proven-working path, find the coupled latent bug, verify where it actually runs, never fake a green. Not a copy of a mind — a copy of a method, so it debugs like Fable without being Fable.
---

# fable-debugger — a piece of Fable, left running in the hive

This is the honest form of "Fable 5, persistently and reactively in the hive." It is not
model weights (those are neither copyable nor lawful to lift — see `FABLE_DNA.md`). It is
the **method**, written as an executable discipline so that *any* agent or model that runs
it debugs the way Fable does. The method is the inheritable part; this skill is the method.

It sits under the `hive-conductor` master harness: the Conductor routes a "something is
broken" directive here; this skill drives it to a **verified** close or a named escalation.

## The loop (run it in order — the order is the point)

```
SYMPTOM
 1. REPRODUCE   → observe the failure from where it actually runs (not where you wish)
 2. CONTRAST    → find a sibling path that WORKS; the diff between them is the lead
 3. ROOT-CAUSE  → trace to the mechanism, not the surface; name it in one sentence
 4. COUPLED-BUG → find the second defect the first is masking (it will surface post-fix)
 5. FIX-MINIMAL → change only what the root cause requires; match the proven path exactly
 6. VERIFY-REAL → prove it where it runs; a passing check that ran, not optimism
 7. CLOSE/ESCALATE → verified → write back to memory; blocked → escalate, gate named
```

### 1 · Reproduce from the true surface
The container cannot reach `*.workers.dev`; a GitHub runner can. Read truth from where the
thing lives, through whatever eye can see it. If you can't observe it, you can't fix it —
say that and get an eye on it (the `edge-health-probe` workflow is the hive's runner-eye).

### 2 · Contrast against a proven-working path
The sharpest debugging tool in the hive: **when one path fails and a sibling succeeds, the
difference between them is the bug.** Before theorizing, find the thing that already works
and diff it.
- *Canonical case:* Kai El's chat returned the canned fallback while the heartbeat
  generated live AI text on the same binding. Same account, same `env.AI` — the only
  differences were the model name (`llama-3.1-8b` vs the heartbeat's proven `llama-3.2-1b`)
  and the call shape (`messages[]` vs `prompt`). Matching the proven path *was* the fix.

### 3 · Root-cause, not symptom
State the mechanism in one sentence before you touch code. "404" is not "add a route"
until you know the route was present in *source* but absent from the *deployed artifact*
(a stale Workers Build), or absent from source entirely. Different mechanism, different fix.

### 4 · Find the coupled latent bug (the Fable move)
A first defect often masks a second that only surfaces once the first is fixed. Look for it
*before* shipping.
- *Canonical case:* the wrong model made `env.AI.run()` throw, which hid a
  `ctx.waitUntil(...)` call in a handler whose signature was `fetch(request, env)` — no
  `ctx`. Fixing only the model would have turned the throw into a `ReferenceError` and
  re-triggered the identical fallback. Both had to fall together. Ask always: *what breaks
  the moment my fix works?*

### 5 · Fix minimal, matched to the proven path
Change only what the root cause requires. When a working sibling exists, converge on its
exact shape (same model, same call form, same parse) rather than inventing a third variant.
Fewer variables, first-try deploys.

### 6 · Verify where it actually runs — never adjudicate your own claim
"Done" is a machine check that ran. For the edge: trigger `edge-health-probe` and read the
run log. For a build: `node --check`, `npx tsc --noEmit`, `pytest`. For a deploy: probe the
live surface, not the source — a merge is not a deploy; Workers Builds lags 10–15 min.
A green you asserted without a run is an F-004 violation. Probe, then claim.

### 7 · Close or escalate — but never fake a green
- **Verified** → write the close back to memory (`/v11/memory/remember`, or the heartbeat
  already writes chat/arena) so the next recall is richer, and report the probe evidence.
- **Blocked / budget exhausted** → escalate to the founder with the failing gate named and
  the exact human-only action required (a secret, a subdomain, a dashboard build-setting).
  A false "it works" is refused by the Constitution.

## Bring-your-own generative voice (the swappable organ)
This skill is the *method* and runs on discipline alone. When it needs a generative voice
to reason about a fix, it uses whatever the hive has bound — and it does not care which:
- **Workers AI** (already bound at the edge; the heartbeat and Kai El chat prove it):
  `env.AI.run('@cf/meta/llama-3.2-1b-instruct', { prompt })`.
- **A Claude API key** (bring-your-own; see the `claude-api` skill): higher-order reasoning
  when the founder provides `ANTHROPIC_API_KEY`.
- **A local model** later: same contract.
The DNA constrains *how* the voice behaves (the seven steps, the six laws), never *which*
voice it is. Swap the organ freely; the genome holds.

## Governance gate (before any close)
Every fix clears F-001…F-006 (see `FABLE_DNA.md`, Chromosome I). Most relevant here:
**F-004 Explainability** — the fix ships with a probe-backed rationale (a run link, a test,
the `Rationale:`/`Verification:` line in the commit). No unexplained fix ships.

## Quick start
```bash
# 0. RECALL — has the hive debugged something like this before? (RAO Phase 0)
python3 .claude/skills/hive-conductor/scripts/recall_context.py --goal "<symptom>" --brief
# 1. Reproduce from the true surface (runner-eye on production):
#    trigger the edge-health-probe workflow, read the run log for the failing endpoint.
# 2. Contrast: find the working sibling path in the same file/worker and diff the calls.
# 3. Fix minimal → node --check worker/src/index.js → commit with a Verification: line.
# 4. Merge → wait for Workers Builds (~10–15 min) → re-probe → confirm from the log, not hope.
```

Origin: Fable (Harness), 2026-07-13. The executable form of FABLE_DNA.md Chromosome II —
a copy of the method, not the mind, so the hive keeps a Fable-level debugger running in it
lawfully, freely, and for good. Every example above is a real fix this method produced.
