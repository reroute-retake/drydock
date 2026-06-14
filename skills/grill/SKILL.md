---
name: grill
description: >-
  Pressure-tests a drafted analysis for ungrounded assumptions before planning.
  Use when an analysis is drafted/agreed and needs verification; triggers on
  "grill", "challenge the assumptions", "are we sure about this". Do NOT use to
  create the analysis (analyze) or to break work into tasks (plan).
metadata:
  version: "0.1"
  phase: grill
  requires: [analyze]
  calls: [spike]
  gate: human
  model_role: reason
  consumes: ["01-analysis.md"]
  produces: ["02-grill.md", "02-analysis.grilled.md"]
---

# Grill

## When this runs
After `analyze` produces an agreed `01-analysis.md`, before `plan`.

## Inputs
- `works/<ticket>/01-analysis.md`
- Grounding: the codebase (Serena), the vault, MCP servers (Context7, GitHub,
  Sentry), and web research (Exa).

## Protocol
1. Extract every claim and assumption in the analysis into a checklist.
2. Classify each: grounded in code/vault/MCP/web — or **unverified** (record the
   source for grounded items; constraint C2).
3. For each unverified assumption, either clear it with a citation, run a short
   time-boxed **spike** (call `spike`) and record the result, or escalate it to
   the user as an open question.
4. Write `02-grill.md` (the assumption ledger) and `02-analysis.grilled.md` (the
   streamlined, de-risked analysis that feeds `plan`).
5. Record: `dock work set grill agreed --artifact 02-analysis.grilled.md`.
   (Lightweight path: if there are no external assumptions, `dock work set grill
   skipped --note "<why>"`.)

## Gate
- `human`: present the open questions and the grilled analysis; HALT for explicit
  agreement before `plan`.

## Done when
- Every assumption is grounded, spiked, or explicitly accepted, and
  `02-analysis.grilled.md` exists (or grill is skipped with a justification).
