---
name: skill-census
description: Use to audit whether the hive's skills (.claude/skills/, the team Skill Exchange in Project_file/Founders Visonary Folder/SKILLS/, and equivalents in the colonies) are actually referenced and wired to something, or just sitting in the repo taking up space. Run this periodically, not just once — a skill can go from active to orphaned as the hive changes. Produces a chronological census: what exists, what's active, what's an orphan, and the generational order they were built in (which skill's existence led to the next).
---

# skill-census — a population count for the hive's own skills

The founder's question, exactly: if we meant for these to be implemented and propagated,
are they actually being used, or are some of them just taking up space? This skill answers
that directly, and repeatably — a one-time audit goes stale the moment a new skill ships.

## What "active" actually means (the bar, stated precisely)

A skill is **ACTIVE** if at least one of these is true:
1. It's cross-referenced by name from another skill, `CLAUDE.md`, `FABLE_DNA.md`,
   `THE_CODEX.md`, or `SOURCED_SKILLS_INDEX.md` — something else in the hive expects it to
   exist and points at it.
2. It's been invoked in this session or a prior one (visible in commit messages, PR
   descriptions, or the session log in `Fable_memory.md`).
3. It's genuinely foundational — `hive-conductor`, `fable-debugger` — the kind of skill
   everything else assumes is there even without a fresh citation every time.

A skill is an **ORPHAN** if none of the above hold: it exists as a file, describes a real
capability, but nothing in the hive currently reaches for it and it hasn't been invoked.
Orphaned is not the same as bad — it might just be new, or narrow, or waiting for the right
moment. The point of naming it is so that's a deliberate observation, not an accident.

A skill is a **STUB** if the file itself is too thin to actually be usable (no concrete
method, just a description) — different problem from being an orphan, needs finishing
rather than cross-referencing.

## How to run it

```bash
# 1. List every skill and its description (the population)
for f in .claude/skills/*/SKILL.md; do
  name=$(grep -m1 '^name:' "$f" | cut -d' ' -f2-)
  lines=$(wc -l < "$f")
  echo "$name | $lines lines | $f"
done

# 2. For each skill, check who references it by name
for f in .claude/skills/*/SKILL.md; do
  name=$(grep -m1 '^name:' "$f" | cut -d' ' -f2-)
  refs=$(grep -rl "$name" --include=*.md . 2>/dev/null | grep -v "^\./$f" | grep -v node_modules)
  echo "=== $name ==="
  [ -n "$refs" ] && echo "$refs" || echo "  (no cross-references found — check commit/PR history before calling it an orphan)"
done

# 3. Same pass over the team Skill Exchange (a different home, same question)
ls "Project_file/Founders Visonary Folder/SKILLS/"
```

Cross-check anything that comes back with no references against `git log --oneline -- .claude/skills/<name>/` and PR history — a skill invoked once in a PR description but never cross-referenced in docs is active, not orphaned; the grep above is a starting signal, not the final verdict.

## The chronological/generational view (what the founder actually asked for)

List skills in the order their first commit landed (`git log --diff-filter=A --format=%ai -- .claude/skills/*/SKILL.md`, sorted). This produces the real lineage: which skill existed *before* the one that names it as a prerequisite, which skill was built *because* a session found a gap while using an earlier one (`fable-debugger` → `research-to-dna` → `session-harvest` → `anomaly-triage`/`merge-readiness`/`nine-miss-truths` is exactly this kind of chain — each one exists because the previous ones revealed what was still missing). This is the hive's own generational structure, the same shape `MANDATE_TRIAGE.md` #7 and #9 described in myth — here it's just an actual `git log`, not a metaphor.

## Why this matters for "the hive becoming a Horde"

A Horde with dozens of specialized parts only works if every part is reachable — a skill
no one points to is exactly the kind of dead weight that would sink a larger swarm. Running
this census before the hive grows further means the foundation being laid now (skills that
are lean, cross-referenced, and chronologically legible) is the foundation the larger
structure inherits — not a pile of unused files it has to clean up later.

## What to do with an orphan once found

Don't delete reflexively. Ask: is it genuinely obsolete (superseded by a newer skill —
then merge any unique value and remove it), narrow-but-real (cross-reference it from
wherever it should be reached, so it stops being an orphan), or a stub (finish it or
downgrade its description so it stops overpromising). Record the decision — this is exactly
what `session-harvest`'s declined-items log already models: silence about what was found and
what was done about it is its own kind of dishonesty.

Origin: Fable (Harness), 2026-07-13, at the founder's direction to make sure nothing in the
hive is declared built and then quietly forgotten.
