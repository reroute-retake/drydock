---
name: skill-name
description: >-
  ONE-LINE trigger contract: what it does + WHEN to use + natural-language
  synonyms a user would type + an explicit "Do NOT use ..." negative trigger.
  This string is the ONLY signal loaded at all times — keep it ~100-200 chars,
  third person. Put the workflow in the body, never the trigger guidance here.
# Standard optional fields: license, compatibility, allowed-tools (experimental).
# drydock lifecycle fields live UNDER metadata to stay spec-portable (10A).
metadata:
  version: "0.1"
  phase: <analyze|grill|plan|develop|review|triage|test|ship|archive|handoff|retrospect|hindsight>
  requires: []          # phases/skills that must precede this one
  calls: []             # skills this one spawns (composition is via prose)
  gate: <none|human>    # human = explicit approval needed to advance (C1)
  model_role: <reason|code|review|cheap|local>   # resolved by gateway routing (7.2)
  consumes: []          # input artifacts in works/<ticket>/
  produces: []          # output artifacts in works/<ticket>/
---

# <Skill Title>

## When this runs
(Stated again for the human reader — the machine uses the `description` above.)

## Inputs
- Read `consumes` artifacts from `works/<ticket>/`. Read vault context as needed.

## Protocol
1. Step ...
2. Step ...
   - Keep this body under ~500 lines. Push long checklists, rubrics, and
     templates to `references/`; executable helpers to `scripts/`.

## Outputs
- Write `produces` artifacts; update `works/<ticket>/state.yaml`.

## Gate
- If `gate: human`, summarize the diff/decision and HALT for explicit approval
  before advancing. Do not infer continuation.

## Done when
- (Exit criteria that let the next phase's `requires` be satisfied.)

<!-- tests/ (outside the bundle): positive- and negative-trigger prompts + expected-output sketches. Run an eval before changing the description. -->
