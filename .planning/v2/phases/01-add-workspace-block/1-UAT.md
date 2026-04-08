---
status: complete
phase: 01-add-workspace-block
source: 01-01-SUMMARY.md, 01-02-SUMMARY.md, 01-03-SUMMARY.md
started: 2026-04-02T00:00:00Z
updated: 2026-04-03T00:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Init writes [metarepo] block
expected: Run `git w init` in a temp directory. Open the generated `.gitw` file. The top-level settings block should be `[metarepo]`, not `[workspace]`.
result: pass

### 2. [metarepo] + [[workspace]] config loads cleanly
expected: Create a `.gitw` file with a `[metarepo]` section and one or more `[[workspace]]` blocks. Run any command that loads config (e.g., `git w info`). The command should run without a config parse error.
result: pass

### 3. Unknown agentic_frameworks value errors clearly
expected: In a `.gitw` file, add `agentic_frameworks = ["speckit"]` under `[metarepo]`. Run `git w info`. You should get a clear error message indicating `speckit` is not a valid framework (not a generic parse error).
result: pass

### 4. Missing agentic_frameworks defaults silently
expected: In a `.gitw` file, omit `agentic_frameworks` entirely under `[metarepo]`. Run `git w info`. The command should work without any error — no mention of agentic_frameworks in output.
result: pass

### 5. Valid agentic_frameworks = ["gsd"] accepted
expected: In a `.gitw` file, set `agentic_frameworks = ["gsd"]` under `[metarepo]`. Run `git w info`. The command should work without any error related to agentic_frameworks.
result: pass

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]
