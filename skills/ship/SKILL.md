---
name: ship
description: >-
  Squashes the ticket's commits, runs the pre-ship gates, pushes, and opens a PR.
  Use on "ship", "open the PR", "let's merge this", after tests pass. Do NOT use
  to implement code (develop) or to run/triage tests (test/triage).
metadata:
  version: "0.1"
  phase: ship
  requires: [test]
  gate: human
  model_role: cheap
  consumes: ["03-plan.md", "07-test-report.md"]
  produces: ["08-ship.md"]
---

# Ship

## When this runs
After `test` passes and `triage` has resolved or explicitly deferred all blockers.

## Protocol
1. Squash the ticket's per-task commits into a clean, well-described commit.
2. Verify only relevant files are staged — no scratch files, no secrets, no
   generated junk (`.drydock/`, `dist/`, `.env`).
3. Run the pre-ship gates (constraint C6) and do not proceed unless they pass:
   - build, full test suite, lint/format
   - `pre-commit run --all-files` (includes a secret scan, e.g. gitleaks)
4. Push the branch and open a PR. Put the summary, the test report link, and the
   list of addressed review comments in the PR body.
5. Write `works/<ticket>/08-ship.md`: the PR URL, the squash record, and the gate
   results. Record: `dock work set ship done --artifact 08-ship.md`.

## Gate
- `human (merge)`: open the PR for review; the human merges. Do not self-merge.

## Done when
- The PR is open with green pre-ship gates, and `08-ship.md` records its URL.
  (After merge, `archive` ingests the artifacts into the vault and clears works/.)
