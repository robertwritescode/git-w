---
status: complete
phase: 10-detect-v1-workgroup-blocks
source: 10-01-SUMMARY.md
started: 2026-04-07T00:00:00Z
updated: 2026-04-07T00:01:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Hard error on v1 config load
expected: Create a .gitw file containing a [[workgroup]] block (v1 format). Run any git-w command (e.g. git w repo list) against that workspace. The command should fail immediately with an error: "v1 config detected: found 1 [[workgroup]] block(s) — run 'git w migrate' to upgrade"
result: pass

### 2. Error message includes block count
expected: Create a .gitw with two [[workgroup]] blocks. Run any command. The error message should say "found 2 [[workgroup]] block(s)" — the count accurately reflects how many v1 blocks are present.
result: pass

### 3. Error includes migrate directive
expected: The error from loading a v1 config explicitly instructs the user to run "git w migrate" — confirming the upgrade path is communicated.
result: pass

### 4. Clean config loads normally
expected: A .gitw with no [[workgroup]] blocks (or with the v2 [workgroup.NAME] keyed-table format used in .gitw.local) loads without any error. Normal commands work as expected.
result: pass

### 5. V1 error fires before repo validation errors
expected: Create a .gitw with both a [[workgroup]] block AND an invalid repo entry. The v1 detection error is returned first — "v1 config detected" — not a repo validation error. The v1 block is the blocking concern.
result: pass

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
