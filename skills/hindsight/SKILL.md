---
name: hindsight
description: >-
  After a problematic ticket, traces late-phase pain back to an earlier phase and
  proposes a human-gated fix to that upstream skill. Use on "hindsight", "this
  ticket went badly, what should change upstream", "post-mortem the phases". Do
  NOT use for telemetry-wide trends (retrospect) or to write a handoff (handoff).
metadata:
  version: "0.1"
  phase: hindsight
  requires: []
  gate: human
  model_role: reason
  consumes: ["works/<ticket>/"]
  produces: ["skill-change proposals"]
---

# Hindsight

## When this runs
At the end of a problematic ticket, over its per-phase artifacts (not telemetry).

## Inputs
- `works/<ticket>/`: `01-analysis.md`, `02-grill.md`, `03-plan.md`,
  `04-dev-log.md`, `05-review.md`, `06-triage.md`, `07-test-report.md`.

## Protocol (causal analysis)
1. Read the artifacts and find late-phase pain with an **upstream** cause:
   - Many assumptions cleared in `grill` → `analyze` was too shallow.
   - Many review comments → `develop`, `plan`, or `analyze` missed something.
   - Many *rejected* triage comments → `review` was noisy (tighten it) or the
     rejection bar is wrong.
   - Tests failing late → `plan`'s acceptance checks were weak.
2. Pick the single highest-leverage upstream skill to improve. Draft the revised
   SKILL.md (**bump `metadata.version`**) addressing that root cause.
3. Propose it (human-gated):
   `dock skill propose <upstream-skill> --source hindsight --rationale "<symptom → root cause → fix>" --content <file>`.
4. **Never auto-edit.** The proposal is pending until a human runs `dock skill apply`.

## Gate
- `human`.

## Done when
- The root cause is identified and one upstream-skill improvement exists as a
  pending proposal.
