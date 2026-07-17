# LocalAGI — Governance

LocalAGI is the Body colony (archetype: agent platform / swarm) of the
Sovereign Hive federation. This file is a short, colony-local pointer into
the federation-wide governance convention defined in
[TehutiRaEl/THEHIVE](https://github.com/TehutiRaEl/THEHIVE/blob/claude/session-continuation-owj5wr/docs/GOVERNANCE.md).

This repository is a fork of the upstream
[LocalAGI](https://github.com/mudler/LocalAGI) project; this governance
layer is additive documentation only and does not change upstream
conventions, code style, or licensing.

## Principles

Contributions here favor reviewable, single-purpose commits; no destructive
operation (deleting data, force-pushing, dropping schema) happens without
explicit human confirmation; and any change that affects the colony's
public `/colony/*` contract or cross-repo behavior gets a human reviewer
before merge, in line with the federation-wide principles in THEHIVE's
`docs/GOVERNANCE.md`.

## Relevant Roles

The full 101-role catalog lives in
[THEHIVE/docs/ROLES.md](https://github.com/TehutiRaEl/THEHIVE/blob/claude/session-continuation-owj5wr/docs/ROLES.md).
Roles most relevant to this repo:

| # | Role Title | Responsibility |
|---|---|---|
| 13 | Body Colony Director | Owns LocalAGI's agent-platform roadmap (upstream-aligned). |
| 26 | Agent Platform Lead | Owns skills/RAG/multimodal agent conventions. |
| 38 | Go Services Director | Owns Fiber route/middleware conventions. |
| 61 | Agent Architecture Lead | Owns agent/skill architecture (upstream-aligned). |
| 70 | Backend Engineer (LocalAGI) | Implements Go service features. |
| 88 | Code Reviewer Apprentice (LocalAGI) | Performs first-pass PR review. |

## Commit / PR Convention

```
[ROLE: <Role Title>] type(scope): description

Rationale: <one sentence on why this change is needed>
```

`type` follows Conventional Commits (`feat`/`fix`/`docs`/`chore`/`refactor`/`test`).

## Enforcement

Advisory only — see `.github/workflows/governance-check.yml`. It never
fails the build and is not a required status check.
