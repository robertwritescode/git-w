---
phase: 05-add-sync-pair-parsing
plan: "01"
status: complete
completed_at: "2026-04-04"
files_modified:
  - pkg/config/config.go
  - pkg/config/config_test.go
---

# Summary: 05-01 — SyncPairConfig struct and MergeSyncPair

## What Was Built

Added `SyncPairConfig` struct, `SyncPairs []SyncPairConfig` field on `WorkspaceConfig`, and `MergeSyncPair` pure function to `pkg/config/config.go`. Added 8 table-driven test cases in `TestMergeSyncPair` in `pkg/config/config_test.go`.

## Key Decisions

- **D-01**: `SyncPairs` lives directly on `WorkspaceConfig` with no TOML tag (same as `Remotes`); populated by loader
- **D-02**: `Refs` field uses `toml:"refs,omitempty"` — omit when empty
- **D-03**: `MergeSyncPair` defined alongside the struct, same pure function pattern as `MergeRemote`
- **D-04**: Refs override wins if `len > 0`; nil or empty slice override keeps base Refs

## Artifacts Created

**`pkg/config/config.go`:**
- `SyncPairConfig` struct (placed after `RemoteConfig`): From, To, Refs fields with correct TOML tags
- `SyncPairs []SyncPairConfig` field on `WorkspaceConfig` (after `Remotes`, no TOML tag)
- `MergeSyncPair(base, override SyncPairConfig) SyncPairConfig` (placed after `MergeRemote`)

**`pkg/config/config_test.go`:**
- `TestMergeSyncPair` with 8 table-driven cases covering all merge paths (empty override, scalar wins, slice wins, nil/empty slice keeps base)

## Test Result

`mage testfast` — all packages pass, no failures.
