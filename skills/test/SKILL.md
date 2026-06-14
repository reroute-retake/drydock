---
name: test
description: >-
  Prepares and runs the full test suite for the ticket and records the results.
  Use on "test", "run the tests", "verify it works", after triage. Do NOT use to
  open a PR (ship) or to decide on review comments (triage).
metadata:
  version: "0.1"
  phase: test
  requires: [triage]
  gate: none
  model_role: code
  consumes: ["03-plan.md"]
  produces: ["07-test-report.md"]
---

# Test

## When this runs
After `triage` (accepted fixes implemented). The last check before `ship`.

## Inputs
- The codebase and `works/<ticket>/03-plan.md` (acceptance checks per task).

## Protocol
1. Ensure coverage exists for the plan's acceptance checks; add missing tests.
2. Run the suites the stack provides — unit, integration, and (for frontend
   spaces) end-to-end via the Playwright MCP against the dev server (declare the
   port in `space.yaml` `ports:` so it's reachable; design 7B).
3. Write `works/<ticket>/07-test-report.md`: what ran, pass/fail counts, coverage
   notes, and any flaky or skipped tests with reasons.
4. If anything fails, loop back to `develop`; re-run `test` until green.
5. Record: `dock work set test done --artifact 07-test-report.md`.

## Gate
- none — `ship` runs the pre-push gates again before pushing.

## Done when
- All tests pass and `07-test-report.md` records the run.
