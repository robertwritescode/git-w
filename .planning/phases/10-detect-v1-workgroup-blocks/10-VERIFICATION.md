---
phase: 10-detect-v1-workgroup-blocks
verified: 2026-04-07T00:00:00Z
status: passed
score: 5/5 must-haves verified
re_verification: false
---

# Phase 10: Detect v1 Workgroup Blocks — Verification Report

**Phase Goal:** Detect v1 `[[workgroup]]` blocks in `.gitw` at load time and return a hard error directing the user to `git w migrate`.
**Verified:** 2026-04-07
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Loading a `.gitw` with one or more `[[workgroup]]` blocks returns a non-nil error | ✓ VERIFIED | `detectV1Workgroups` returns `fmt.Errorf(...)` when `V1WorkgroupCount > 0`; test case "single workgroup block triggers error" confirms |
| 2 | The error message contains the count of `[[workgroup]]` blocks found | ✓ VERIFIED | Error format: `"v1 config detected: found %d [[workgroup]] block(s) ..."` with `cfg.V1WorkgroupCount`; test case "multiple workgroup blocks include count" asserts `"3"` in error for 3 blocks |
| 3 | The error message directs the user to run `'git w migrate'` | ✓ VERIFIED | Error message ends with `"run 'git w migrate' to upgrade"`; test case "migrate directive in error" asserts `"git w migrate"` in error |
| 4 | Loading a `.gitw` with no `[[workgroup]]` blocks succeeds normally | ✓ VERIFIED | `detectV1Workgroups` returns `nil` when `V1WorkgroupCount == 0`; test case "no workgroup blocks loads successfully" asserts `NoError` |
| 5 | The v1 detection error fires before any other validator (no noise from unrelated checks) | ✓ VERIFIED | `buildAndValidate` calls `detectV1Workgroups` at line 83 — first call before `validateRepoNames` (line 87) and all other validators; test case "workgroup error fires before repo path error" confirms v1 error wins over repo path error |

**Score:** 5/5 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pkg/config/config.go` | `V1WorkgroupCount int` field on `WorkspaceConfig` | ✓ VERIFIED | Line 24: `V1WorkgroupCount int // in-memory only; set when [[workgroup]] blocks found in v1 config` |
| `pkg/config/loader.go` | `detectV1Workgroups` check first in `buildAndValidate`; `countV1WorkgroupBlocks` raw scanner; assignment `cfg.V1WorkgroupCount = countV1WorkgroupBlocks(data)` | ✓ VERIFIED | Lines 65, 83, 423–443 — all present and substantive |
| `pkg/config/loader_test.go` | `TestV1WorkgroupDetection` suite method with 5 sub-cases | ✓ VERIFIED | Lines 972–1052 — all 5 cases present and testing distinct behaviors |

**Note on plan deviation:** The PLAN specified `WorkgroupList []map[string]any` on `diskConfig` as the detection mechanism. This was intentionally replaced with a raw byte scanner (`countV1WorkgroupBlocks`) because `go-toml` cannot unmarshal `[[workgroup]]` array-of-tables into `[]map[string]any` when existing configs use `[workgroup.NAME]` keyed-table format. The deviation was documented in `10-01-SUMMARY.md`. The goal is fully achieved by the alternative approach.

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| raw `.gitw` bytes | `WorkspaceConfig.V1WorkgroupCount` | `cfg.V1WorkgroupCount = countV1WorkgroupBlocks(data)` in `loadMainConfig` | ✓ WIRED | `loader.go:65` — assignment happens after `ensureWorkspaceMaps`, before `buildAndValidate` |
| `buildAndValidate` | `detectV1Workgroups` | first call at `loader.go:83`, before `validateRepoNames` at line 87 | ✓ WIRED | Order confirmed by reading `buildAndValidate` body lines 82–125 |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| CFG-10 | `10-01-PLAN.md` | Detect v1 `[[workgroup]]` blocks at config load time and hard-error with migration hint | ✓ SATISFIED | `detectV1Workgroups` returns descriptive error; all 5 test cases pass; `mage testfast` green |

---

### Anti-Patterns Found

None. Scanned `pkg/config/config.go`, `pkg/config/loader.go`, and `pkg/config/loader_test.go` for TODO/FIXME/placeholder comments, empty implementations, hardcoded stubs, and console-only handlers. None found.

---

### Human Verification Required

None. All observable behaviors are fully verifiable programmatically:
- Error returned or not on load is binary
- Error message content is asserted by tests
- Call order in `buildAndValidate` is confirmed by source reading

---

### Gaps Summary

No gaps. All 5 must-have truths are verified, all artifacts are substantive and wired, requirement CFG-10 is satisfied, and the full test suite passes.

---

_Verified: 2026-04-07_
_Verifier: the agent (gsd-verifier)_
