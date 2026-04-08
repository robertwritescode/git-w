---
phase: 05-add-sync-pair-parsing
plan: "02"
status: complete
completed_at: "2026-04-04"
files_modified:
  - pkg/config/loader.go
  - pkg/config/loader_test.go
---

# Summary: 05-02 — sync_pair loader wiring, validation, and cycle detection

## What Was Built

Wired `[[sync_pair]]` parsing fully into the loader:
- Added `SyncPairList []SyncPairConfig` to `diskConfig` (after `RemoteList`)
- `loadMainConfig` now sets `cfg.SyncPairs = dc.SyncPairList`
- `prepareDiskConfig` now sets `SyncPairList: cfg.SyncPairs` for round-trip save
- Added `validateSyncPairFields` — checks empty `from`/`to`, detects duplicate `(from, to)` pairs
- Added `detectSyncCycles` + `dfsSyncCycle` — DFS with visited/in-stack sets; stops at first cycle
- Both wired into `buildAndValidate` after `validateRemotes`, before `validateAliasFields`
- Added 4 test functions: `TestSyncPairBlocksParse` (4 cases), `TestSyncPairRoundTrip`, `TestSyncPairValidation` (7 cases), `TestSyncCycleDetection` (7 cases)

## Key Decisions

- **D-05**: Two separate functions (`validateSyncPairFields`, `detectSyncCycles`), both called from `buildAndValidate`
- **D-06**: No name-reference validation in Phase 5 (deferred to Phase 7)
- **D-07**: DFS with visited/in-stack sets — O(V+E)
- **D-08**: Stop at first cycle found; report only that cycle
- **D-09**: Error format exactly: `"sync_pair cycle detected: origin → personal → origin"` — full path, closing node repeated

## Artifacts Created / Modified

**`pkg/config/loader.go`:**
- `diskConfig.SyncPairList []SyncPairConfig` field
- `loadMainConfig` sets `SyncPairs: dc.SyncPairList`
- `prepareDiskConfig` sets `SyncPairList: cfg.SyncPairs`
- `validateSyncPairFields(cfg *WorkspaceConfig) error`
- `detectSyncCycles(cfg *WorkspaceConfig) error`
- `dfsSyncCycle(node string, adj map[string][]string, visited, inStack map[string]bool, path []string) []string`
- `buildAndValidate` calls both validation functions after `validateRemotes`

**`pkg/config/loader_test.go`:**
- `TestSyncPairBlocksParse` — 4 table-driven cases (no blocks, single no refs, single with refs, multiple)
- `TestSyncPairRoundTrip` — save/load round-trip preserves two sync_pair entries
- `TestSyncPairValidation` — 7 table-driven cases (valid pairs, missing from, missing to, duplicate, same-from-different-to, same-to-different-from)
- `TestSyncCycleDetection` — 7 table-driven cases (no pairs, linear chain, 2-node cycle, 3-node cycle, self-loop, mid-chain cycle, diamond)

## Cycle Detection Implementation Note

`dfsSyncCycle` builds the cycle slice using `make+copy` to avoid the shared-backing-array hazard from `append(path[i:], neighbor)`. The 7 cycle test cases (including `cycle_in_longer_chain` which exercises `b→c→b` where `a→b` exists) confirm correctness under the race detector.

## Test Result

`mage test` (race detector) — all 16 packages pass, 0 failures.
