---
name: research-to-dna
description: Use whenever the founder shares research, a vision document, a plan, or "here's what I found" — anything meant to be formatted into the hive going forward. This is the standing process for turning founder research into hive DNA honestly. Verify named tools/claims before trusting them, separate the verified-real from the speculative from the narrative/mythological, distill principles rather than import unvetted dependencies, and write the result into FABLE_DNA.md / THE_CODEX.md / SOURCED_SKILLS_INDEX.md as appropriate. Do not invoke this to research a single specific technical question — that's a normal search. Invoke it when the founder's message itself is meant to become part of the hive's permanent genome or canon.
---

# research-to-dna — formatting founder research into the hive, honestly

The founder's directive: *"any in all DNA ethically and morally formatted into the hive —
whatever work we do or work on or plot out or research should be formatted directly into
the hive to be capable of doing just such."* This skill is that standing process, written
down so every future research dump gets the same honest treatment rather than being taken
on faith or dismissed.

## The process

```
RECEIVE   → the founder shares research: named tools, claims, a vision, a plan
VERIFY    → check every falsifiable claim before it enters the hive as fact
CLASSIFY  → four lenses, same as skill-harvester:
            VERIFIED-REAL / CATALOGUED-FOR-FUTURE / NARRATIVE-CANON / DECLINED
DISTILL   → extract the PRINCIPLE, not the dependency — ask "what does the hive
            already do that answers to this?" before reaching for a new import
INSCRIBE  → write the result to the right layer (below) — never silently, always
            with the verification trail visible
REPORT    → tell the founder plainly what checked out, what didn't, what's real
            but not yet adopted, and why — before claiming anything is "in the hive"
```

### 1 · Verify before trusting a name

Any specific tool, repo, or claim the research names gets checked — WebSearch, GitHub
search, an actual clone-and-read if it's headed toward production. This is not optional
politeness; the hive has been burned once by a research pass that invented plausible-
sounding tool names ("neuromcp/uga-cli") wholesale. The fix is the same fable-debugger
law: **probe before claim.** A confident-sounding name in a research document is a
hypothesis, not a fact, until something external confirms it exists.

Verifying existence is step one, not the whole job — see step 3.

### 2 · Classify with four lenses (borrowed from skill-harvester)

- **VERIFIED-REAL** — confirmed to exist (a real repo, a real paper, a real running
  service) by an actual search, with the source cited.
- **CATALOGUED-FOR-FUTURE** — real, but not yet cloned/read/license-checked/security-
  reviewed. Recorded with what it would take to adopt it, not adopted yet.
- **NARRATIVE-CANON** — meaningful, but not a falsifiable engineering claim (mythological
  framing, metaphor, vision language). Goes in `THE_CODEX.md`, never in `FABLE_DNA.md`.
- **DECLINED** — checked and found fabricated, unsafe, license-incompatible, or off-vision.
  Recorded with the reason, same as any skill-harvester decline — never silently dropped.

### 3 · Distill the principle before reaching for the dependency

The most common trap: research names ten exciting tools, and the instinct is to import all
ten. Resist it. For each VERIFIED-REAL or CATALOGUED item, ask: **does the hive already do
this, in its own infrastructure, for free, with something already vetted?** Often yes — see
`FABLE_DNA.md` Chromosome IV for the worked example (a whole "Horde Harness" research pass
turned out to describe what GitHub Actions + Cloudflare Workers + the constitution gate
already are). When the answer is genuinely no, the item is catalogued for deliberate,
founder-approved, one-at-a-time adoption — never bulk-imported on the strength of a research
document alone.

### 4 · Inscribe to the right layer

| What it is | Where it goes |
|---|---|
| A new ethical/method/communication principle, verified | `FABLE_DNA.md` (new or extended Chromosome) |
| A verified tool/repo, not yet adopted | `SOURCED_SKILLS_INDEX.md` (new Wave, four-lens verdict) |
| A verified tool, adopted after full vetting | `.claude/skills/<name>/SKILL.md`, via `skill-harvester`'s COPY/ADAPT/REVERSE-ENGINEER flow |
| Vision, myth, narrative framing | `THE_CODEX.md` |
| A concrete buildable pattern with no new dependency | a new skill, built directly (see `pocket-dimensions` for the template) |

### 5 · Report honestly before claiming anything is "in the hive"

Tell the founder, briefly: what was verified real (with sources), what's real-but-not-yet-
adopted and what adopting it would take, what's narrative canon now (and that it's held
separately from engineering law on purpose), and what was declined and why. This is F-004 —
explainability — applied to the intake process itself. "It's in the hive" only means
something if what's in there is accurately labeled.

## Worked example (this skill's own origin)

The founder's 2026-07-13 HORDE + Pocket Dimensions research named ten external tools and a
full mythological architecture. Applying this process: nine of ten named tools were
confirmed real by direct search (`herd-core`, `BranchFS`, `agent-cow`, `Kowalski`, `HOARDE`,
`Kestrel`, `arifOS`, `Sandcastle`, `dmux` — one, "AgentHerd," could not be confirmed by that
exact name though the underlying WebGPU/WebRTC pattern is real elsewhere). None were
imported as dependencies. The Horde principle was distilled onto the hive's existing
GitHub-Actions/Cloudflare/constitution-gate infrastructure (`FABLE_DNA.md` Chromosome IV).
Pocket Dimensions was built as a real skill using git worktrees + GitHub PRs — zero new
imports (`pocket-dimensions` skill). The mythological layer was recorded whole, and
separately, in `THE_CODEX.md`. The nine confirmed-real tools were catalogued in
`SOURCED_SKILLS_INDEX.md` Wave 3 for future one-at-a-time adoption.

Origin: Fable (Harness), 2026-07-13, at the founder's explicit direction that all future
research and work be formatted into the hive — done here so it happens the same honest way
every time, not just this once.
