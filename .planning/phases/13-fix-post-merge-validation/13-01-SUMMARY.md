---
plan: 13-01
phase: 13-fix-post-merge-validation
status: complete
completed: 2026-04-07
commits:
  - 50789c1
  - 54816b5
  - 9e7db99
---

## Summary

Closed three correctness gaps in `pkg/config/loader.go` identified by the M1 integration audit. All three fix paths are covered by tests that failed before the fix and pass after.

## What Was Built

**INT-01 — Re-validate workstream remotes after private config merge**

Added `revalidateWorkstreamRemotes(cfg)` called in `Load()` after `mergePrivateConfig`. Private config (`.git/.gitw`) can add workstreams referencing remotes that only exist in the private file. Without the second pass, invalid remote references in private-added workstreams were silently accepted. The new function re-checks all in-memory workstream remote references against the merged remotes list without the disk-count check that would spuriously fail when private config adds new workstreams.

**INT-02 — Validate sync_pair from/to remote names at load time**

`validateSyncPairFields` now checks that `p.From` and `p.To` exist in `cfg.Remotes` using `cfg.RemoteByName()`. Existing tests that defined valid sync_pairs without the referenced remotes were updated to include the required `[[remote]]` blocks. Three new test cases cover unknown-from, unknown-to, and valid-pair scenarios.

**INT-03 — Preserve path-convention warnings when alias field validation fails**

`loadMainConfig` now returns `cfg, err` (instead of `nil, err`) when `buildAndValidate` fails. This makes `cfg.Warnings` accessible to callers even when validation errors occur. `Load()` propagates the partial cfg from `loadMainConfig` on error. `LoadConfig` now displays warnings from a partial cfg before returning the error to the caller.

## Key Files Modified

- `pkg/config/loader.go` — three targeted fixes
- `pkg/config/loader_test.go` — six new tests (two per INT gap) plus fixture updates for INT-02
- `pkg/config/round_trip_test.go` — `[[remote]]` blocks added to sync_pair round-trip fixtures

## Self-Check: PASSED

- INT-01: `TestPrivateConfigWorkstreamUnknownRemoteAfterMerge` and `TestPrivateConfigWorkstreamValidRemoteAfterMerge` pass
- INT-02: `TestSyncPairRemoteValidation` (three cases) passes; all existing sync_pair tests pass with updated fixtures
- INT-03: `TestPathWarningsPreservedOnAliasError` passes
- `mage test` (race detector, no cache) — all packages pass, no regressions
