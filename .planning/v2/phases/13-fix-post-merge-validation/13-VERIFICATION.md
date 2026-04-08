---
phase: 13-fix-post-merge-validation
verified: 2026-04-07T00:00:00Z
status: passed
score: 4/4 must-haves verified
---

# Phase 13: Fix Post-Merge Validation — Verification Report

**Phase Goal:** Close three correctness gaps in `pkg/config/loader.go` identified by the M1 integration audit: (INT-01) private-file workstream remote references not re-validated after merge, (INT-02) `sync_pair` entries can silently name nonexistent remotes, (INT-03) path-convention warnings dropped when alias validation fails.
**Verified:** 2026-04-07
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | Private-file workstream remote references are cross-validated after merge (INT-01) | ✓ VERIFIED | `revalidateWorkstreamRemotes(cfg)` called at `loader.go:32` immediately after `mergePrivateConfig` succeeds |
| 2  | `sync_pair` entries referencing nonexistent remotes produce a load-time error (INT-02) | ✓ VERIFIED | `validateSyncPairFields` calls `cfg.RemoteByName(p.From)` at line 246 and `cfg.RemoteByName(p.To)` at line 250; returns descriptive errors on unknown names |
| 3  | Path-convention warnings are preserved when `validateAliasFields` returns an error (INT-03) | ✓ VERIFIED | `loadMainConfig` returns `cfg, err` (not `nil, err`) at line 94; `LoadConfig` guards `cfg.Warnings` range with `if cfg != nil` at line 998 |
| 4  | All three fix paths are covered by tests that fail before the fix and pass after | ✓ VERIFIED | Four targeted tests confirmed (see Artifacts section) |

**Score:** 4/4 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pkg/config/loader.go` | Three corrected validation paths in `loadMainConfig`, `validateSyncPairFields`, and `Load` | ✓ VERIFIED | `revalidateWorkstreamRemotes` at line 32; `RemoteByName` lookups at lines 246, 250; `return cfg, err` at lines 23, 94 |
| `pkg/config/loader_test.go` | Tests for INT-01, INT-02, INT-03 scenarios | ✓ VERIFIED | `TestPrivateConfigWorkstreamValidRemoteAfterMerge` (line 2539), `TestPrivateConfigWorkstreamUnknownRemoteAfterMerge` (line 2563), `TestSyncPairRemoteValidation` (line 1563), `TestPathWarningsPreservedOnAliasError` (line 836) |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `Load()` | `revalidateWorkstreamRemotes` | second call after `mergePrivateConfig` | ✓ WIRED | `loader.go:32` — `if err := revalidateWorkstreamRemotes(cfg); err != nil` follows `mergePrivateConfig` block at lines 26-31 |
| `validateSyncPairFields` | `cfg.RemoteByName` | lookup of `from`/`to` remote names | ✓ WIRED | `loader.go:246` — `cfg.RemoteByName(p.From)`, `loader.go:250` — `cfg.RemoteByName(p.To)` |
| `loadMainConfig` | `cfg` with warnings | return `cfg` even on `buildAndValidate` error | ✓ WIRED | `loader.go:94` — `return cfg, err` (not `nil, err`); `LoadConfig` safely ranges `cfg.Warnings` with nil guard |

> **Note on key_link pattern mismatch:** The PLAN specified `pattern: "validateWorkstreams.*configPath.*cfg"` for the INT-01 link. The implementation correctly uses `revalidateWorkstreamRemotes` (a dedicated function) rather than a second call to `validateWorkstreams`. This was an intentional design decision documented in the plan — `validateWorkstreams` requires `configPath` and performs checks that are invalid on post-merge state; `revalidateWorkstreamRemotes` performs only the remote-reference cross-check. The link is verified by direct inspection.

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CFG-07 | 13-01-PLAN.md | Two-file merge (`.gitw` + `.gitw.local`) produces a correct merged config | ✓ SATISFIED | INT-01 adds post-merge revalidation; `TestPrivateConfigWorkstreamUnknownRemoteAfterMerge` verifies stale references are caught after merge |
| CFG-05 | 13-01-PLAN.md | `sync_pair` cycle detection and remote validation | ✓ SATISFIED | INT-02 adds `RemoteByName` checks in `validateSyncPairFields`; `TestSyncPairRemoteValidation` covers unknown-from, unknown-to, and valid-pair cases |
| CFG-02 | 13-01-PLAN.md | `track_branch`/`upstream` alias field handling | ✓ SATISFIED | INT-03 ensures `cfg` is returned (not `nil`) on alias validation error, so all accumulated warnings remain accessible to callers |
| CFG-03 | 13-01-PLAN.md | Path-convention warnings surface to the caller | ✓ SATISFIED | `TestPathWarningsPreservedOnAliasError` (line 836) confirms warnings are non-empty in the returned `cfg` even when an alias error is also present |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | — | — | No anti-patterns found in modified files |

---

### Human Verification Required

None. All three fix paths are deterministic logic branches fully exercisable by unit/integration tests. The test suite (`mage testfast`) passes with zero failures across all 16 packages.

---

### Gaps Summary

No gaps. All four must-have truths are verified, all artifacts are substantive and wired, all key links are confirmed in the actual code, and all four requirement IDs are satisfied.

---

_Verified: 2026-04-07_
_Verifier: the agent (gsd-verifier)_
