---
phase: 11-updatepreservingcomments-round-trip
verified: 2026-04-07T00:00:00Z
status: passed
score: 7/7 must-haves verified
re_verification: false
gaps: []
human_verification: []
---

# Phase 11: `UpdatePreservingComments` Round-Trip — Verification Report

**Phase Goal:** Fix two documented tech debt items in `pkg/toml/preserve.go` and `pkg/config/loader.go`, and add a round-trip test suite for all `diskConfig` block types.
**Verified:** 2026-04-07
**Status:** PASSED
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `applySmartUpdate` propagates errors from `smartUpdate` instead of swallowing them | ✓ VERIFIED | Lines 77–83 in `preserve.go`: `return nil, err` replaces the old `return newBytes, nil` |
| 2 | All `interface{}` in `preserve.go` and `loader.go` replaced with `any` | ✓ VERIFIED | `grep -c 'interface{}'` returns `0` for both files |
| 3 | `mage test` passes with no regressions (Plan 11-01) | ✓ VERIFIED | All 16 packages green under race detector |
| 4 | `config.Save()` + `config.Load()` preserves comments at correct positions for all `diskConfig` block types | ✓ VERIFIED | All 11 RoundTripSuite subtests pass |
| 5 | Field order is stable after a Save+Load round-trip | ✓ VERIFIED | `TestRoundTrip_AllBlocks` identity round-trip passes |
| 6 | All block types in `diskConfig` are exercised by the suite | ✓ VERIFIED | 10 per-block cases cover every field in `diskConfig` struct (see notes below) |
| 7 | `mage test` passes including the new suite (Plan 11-02) | ✓ VERIFIED | All 16 packages green under race detector |

**Score:** 7/7 truths verified

---

### Block Type Coverage Notes

The plan's context section listed `[[repo.branch_override]]` as a block type to exercise. `BranchOverrideConfig` does not exist in the codebase — it was referenced as a forward-looking interface but was never implemented in Phases 1–11. The test correctly replaced this with a "repo path mutation" case, which still exercises the `[[repo]]` array-of-tables block with a field mutation. The actual `diskConfig` struct contains exactly 8 field groups:

| diskConfig field | TOML block | Test case |
|---|---|---|
| `Metarepo` | `[metarepo]` | "metarepo" |
| `Workspaces` | `[[workspace]]` | "workspace block" |
| `RepoList` | `[[repo]]` | "repo", "repo path mutation" |
| `RemoteList` | `[[remote]]` | "remote", "remote with branch_rule" |
| `SyncPairList` | `[[sync_pair]]` | "sync_pair" |
| `WorkstreamList` | `[[workstream]]` | "workstream" |
| `Groups` | `[groups.*]` | "groups" |
| `Worktrees` | `[worktrees.*]` | "worktrees" |

All 8 `diskConfig` field groups are covered. The plan's mention of `[[repo.branch_override]]` was aspirational — the type does not exist yet and is deferred to Phase 2.

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `pkg/toml/preserve.go` | TOML comment-preservation logic, tech-debt-free; contains `func applySmartUpdate` | ✓ VERIFIED | 536 lines; `applySmartUpdate` propagates errors; zero `interface{}` occurrences |
| `pkg/config/loader.go` | Config save/load with updated interface signatures; contains `saveWithCommentPreservation` | ✓ VERIFIED | `saveWithCommentPreservation(path string, newConfig any)` and `marshalToml(cfg any)` confirmed |
| `pkg/config/round_trip_test.go` | Table-driven round-trip test suite; contains `RoundTripSuite`; min 150 lines | ✓ VERIFIED | 342 lines; `RoundTripSuite` present; `TestRoundTrip_PerBlockType` (10 cases) + `TestRoundTrip_AllBlocks` |

---

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `pkg/toml/preserve.go` | `applySmartUpdate` | error propagation | ✓ WIRED | `return nil, err` at line 80 (confirmed via grep and file read) |
| `pkg/config/loader.go` | `saveWithCommentPreservation` | `interface{}` → `any` | ✓ WIRED | Signature is `func saveWithCommentPreservation(path string, newConfig any)` at line 756 |
| `pkg/config/round_trip_test.go` | `pkg/config/loader.go` | `config.Save` + `config.Load` calls | ✓ WIRED | Lines 239, 244, 309, 313 confirmed via grep |
| `pkg/config/round_trip_test.go` | comment position assertion | `strings.Index` | ✓ WIRED | Lines 251–254 and 337–340 confirmed via grep |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| CFG-12 | 11-01, 11-02 | `UpdatePreservingComments` round-trips all v2 fields without losing comments or field order | ✓ SATISFIED | Round-trip suite passes with position assertions for all 8 `diskConfig` block types; error propagation fix prevents silent failure masking |

**Note on REQUIREMENTS.md status:** CFG-12 is listed as `Pending` in REQUIREMENTS.md traceability table. This is a tracking discrepancy — the ROADMAP.md marks Phase 11 as complete (`[x]`) and both plans are marked complete. REQUIREMENTS.md should be updated to `Complete` for CFG-12.

---

### Anti-Patterns Found

No blockers or warnings found.

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| — | — | — | — | None |

Scanned `pkg/toml/preserve.go`, `pkg/config/loader.go`, `pkg/config/round_trip_test.go` for TODO/FIXME, placeholder returns, empty implementations, and hardcoded stubs. Nothing flagged.

---

### Human Verification Required

None. All assertions are fully automated (comment presence + position index comparison) and the full test suite passes under the race detector.

---

## Summary

Phase 11 goal is fully achieved. Both documented tech debt items are resolved:

1. **`applySmartUpdate` error propagation** — the silent `return newBytes, nil` fallback is replaced by `return nil, err`, surfacing errors from `smartUpdate` to callers.
2. **`interface{}` → `any` migration** — all 9 function signatures and all local variable declarations in `pkg/toml/preserve.go` and the two relevant functions in `pkg/config/loader.go` now use `any`.

The round-trip test suite (`RoundTripSuite`) covers all 8 field groups in `diskConfig` across 10 per-block-type subtests and one all-blocks identity round-trip integration test. Comment preservation and position assertions (not just `Contains`) pass for every case. The full test suite (16 packages, race detector) is green.

The only non-blocking observation is that REQUIREMENTS.md has CFG-12 still marked `Pending` in the traceability table — this should be updated to `Complete`.

---

_Verified: 2026-04-07_
_Verifier: the agent (gsd-verifier)_
