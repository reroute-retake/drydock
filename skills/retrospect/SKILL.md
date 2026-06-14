---
name: retrospect
description: >-
  Mines past session telemetry to propose small, human-gated improvements to the
  lifecycle skills. Use on "retrospect", "what can we learn from recent runs",
  "optimize the skills from telemetry". Do NOT use to analyze a single ticket's
  phase artifacts (that's hindsight) or to write a handoff (handoff).
metadata:
  version: "0.1"
  phase: retrospect
  requires: []
  gate: human
  model_role: reason
  consumes: ["/workspace/telemetry"]
  produces: ["skill-change proposals"]
---

# Retrospect

## When this runs
Periodically, after several sessions. Operates on telemetry, not a live ticket.

## Inputs
- Aggregated session telemetry: run `dock telemetry --json` (or read the mounted
  `/workspace/telemetry/**/events.jsonl`).

## Protocol
1. Aggregate telemetry across recent sessions. Look for signals: phases with high
   failure counts, excessive tool calls, high token/cost, or repeated retries.
2. Apply **bounded-learning guardrails**: only propose a change backed by a real
   pattern (several observations, not one), keep each change small and targeted,
   and change one thing at a time.
3. For each improvement: write the **revised SKILL.md** to a file, **bump
   `metadata.version`**, and include trigger positive/negative test notes. Then:
   `dock skill propose <skill> --source retrospect --rationale "<pattern → fix>" --content <file>`.
4. **Never edit a skill directly.** Proposals stay pending until a human runs
   `dock skill apply` (principle P6 / constraint C9).

## Gate
- `human`: a person reviews each proposal (`dock skill show <id>`) and
  applies or rejects it.

## Done when
- Findings are summarized and any improvements exist as pending proposals.
