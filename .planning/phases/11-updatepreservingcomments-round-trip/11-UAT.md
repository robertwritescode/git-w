---
status: complete
phase: 11-updatepreservingcomments-round-trip
source: 11-01-SUMMARY.md, 11-02-SUMMARY.md
started: 2026-04-07T22:42:21Z
updated: 2026-04-07T22:47:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Round-trip test suite passes
expected: `mage test` runs all packages with the race detector. All 16 packages pass, including the new `RoundTripSuite` in `pkg/config/round_trip_test.go` with its 11 subtests covering every block type.
result: pass

### 2. Comment preservation — repo block
expected: A `.gitw` file with a `[repos.<name>]` block that has an inline comment above the `path` key survives `config.Load` + `config.Save` with the comment still present in the output file, positioned before the `path` key.
result: pass

### 3. Comment preservation — all-blocks integration
expected: A `.gitw` containing all block types (metarepo, workspace, repo, remote, sync_pair, workstream, groups, worktrees) with comments survives a `Load` + `Save` round-trip with every comment intact and in position.
result: skipped

### 4. applySmartUpdate error propagation
expected: The `applySmartUpdate` function in `pkg/toml/preserve.go` no longer silently swallows errors from `smartUpdate`. If `smartUpdate` returns an error, `applySmartUpdate` returns `(nil, err)` — it does not return `(newBytes, nil)` with comments silently dropped.
result: skipped

## Summary

total: 4
passed: 2
issues: 0
pending: 0
skipped: 2
blocked: 0

## Gaps

[none yet]
