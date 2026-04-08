---
status: complete
phase: 06-add-workstream-root-config-block
source: [06-01-SUMMARY.md, 06-02-SUMMARY.md]
started: 2026-04-05T20:21:15Z
updated: 2026-04-05T20:35:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Parse valid workstream block
expected: `.gitw` with `[[remote]] name = "origin"` and `[[workstream]] name = "ws-1" remotes = ["origin"]` loads successfully with no error.
result: pass

### 2. Require remotes field
expected: `[[workstream]]` block with only `name` and no `remotes` key fails load with a missing required remotes error.
result: pass

### 3. Reject unknown workstream keys
expected: `[[workstream]]` block with an unrecognized key (e.g. `color`) fails load with an unknown key error.
result: pass

### 4. Reject duplicate workstream names
expected: Two `[[workstream]]` blocks with the same `name` fail load with a duplicate name error.
result: pass

### 5. Reject reference to undeclared remote
expected: `[[workstream]]` referencing a remote not declared in any `[[remote]]` block in the same file fails load with an unknown remote error. Cross-file remote references (e.g. private remote in `.git/.gitw`) are a phase 7 concern.
result: pass

### 6. Normalize workstream ordering deterministically
expected: After a save/reload cycle, `remotes` within a workstream are stored in sorted order.
result: pass

### 7. Accept workstream blocks in public config
expected: `[[workstream]]` defined in `.gitw` (shared config) loads successfully; it is not restricted to local-only config.
result: pass

## Summary

total: 7
passed: 7
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]
