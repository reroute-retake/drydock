---
name: grill
description: >-
  Pressure-tests a drafted analysis for ungrounded assumptions before planning.
  Use when an analysis is marked draft/agreed and needs verification; triggers on
  "grill", "challenge the analysis", "validate assumptions", "are we sure about".
  Do NOT use to create the analysis (that's analyze) or to break work into tasks
  (that's plan).
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
After `analyze` produces an agreed `01-analysis.md`, and before `plan`.

## Inputs
- `works/<ticket>/01-analysis.md`
- Grounding sources: the codebase (Serena), the vault, MCP servers (Context7,
  GitHub, Sentry), and web research (Exa).

## Protocol
1. Extract every claim and assumption in the analysis into a checklist.
2. For each, classify the evidence: grounded in code / vault / MCP / web — or
   **unverified**. Record the source for grounded items (constraint C2).
3. For each unverified assumption, either:
   - clear it with a citation, or
   - run a short, time-boxed **spike** (call the `spike` skill) and record the result, or
   - escalate to the user as an open question.
4. Write `02-grill.md`: the assumption ledger (claim, status, evidence/spike, resolution).
5. Produce `02-analysis.grilled.md`: the streamlined, de-risked analysis that
   becomes the input to `plan`.

## Outputs
- `02-grill.md`, `02-analysis.grilled.md`; update `state.yaml` (grill -> agreed).

## Gate
- `human`: present the open questions and the grilled analysis; HALT for
  explicit agreement before `plan` may start.

## Done when
- Every assumption is grounded, spiked, or explicitly accepted by the user, and
  `02-analysis.grilled.md` exists.
