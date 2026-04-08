---
phase: 10-detect-v1-workgroup-blocks
plan: 01
subsystem: config
tags: [toml, loader, validation, migration, v1-detection]

requires: []
provides:
  - "Load-time hard error when [[workgroup]] blocks found in .gitw"
  - "V1WorkgroupCount int field on WorkspaceConfig (in-memory only)"
  - "detectV1Workgroups private function, first check in buildAndValidate"
  - "countV1WorkgroupBlocks raw byte scanner for [[workgroup]] header detection"
affects: [phase-59-migrate-command]

tech-stack:
  added: []
  patterns:
    - "Raw byte line scanning for v1 config detection (avoids TOML struct field conflicts)"
    - "V1 detection fires before all other validators via first call in buildAndValidate"

key-files:
  created: []
  modified:
    - pkg/config/config.go
    - pkg/config/loader.go
    - pkg/config/loader_test.go

key-decisions:
  - "Used raw byte scanning (countV1WorkgroupBlocks) instead of diskConfig struct field — []map[string]any conflicts with [workgroup.NAME] keyed-table format used in tests"
  - "Detection targets [[workgroup]] array-of-tables (v1 format); [workgroup.NAME] keyed tables (v2 .gitw.local format) are unaffected"
  - "detectV1Workgroups is the first call in buildAndValidate — v1 configs produce no noise from other validators"

patterns-established:
  - "Raw byte line scanning is appropriate when a TOML struct field would conflict with an existing field sharing the same key in a different format"

requirements-completed: [CFG-10]

duration: 25min
completed: 2026-04-07
---

# Phase 10: detect-v1-workgroup-blocks Summary

**Load-time hard error for v1 `[[workgroup]]` blocks using raw byte scanning, with migrate directive and 5-case test coverage**

## Performance

- **Duration:** 25 min
- **Completed:** 2026-04-07
- **Tasks:** 1
- **Files modified:** 3

## Accomplishments

- `V1WorkgroupCount int` field added to `WorkspaceConfig` (in-memory only, alongside `Warnings`)
- `detectV1Workgroups` added as the first check in `buildAndValidate` — fires before `validateRepoNames` and all other validators
- Error message: `"v1 config detected: found N [[workgroup]] block(s) — run 'git w migrate' to upgrade"`
- 5 sub-cases in `TestV1WorkgroupDetection`: single block, count in message, migrate directive, clean config, fires-before-repo-error

## Task Commits

1. **Task 1: Add v1 workgroup detection to config loader** - `56375eb` (feat)

## Files Created/Modified

- `pkg/config/config.go` — `V1WorkgroupCount int` field added to `WorkspaceConfig`
- `pkg/config/loader.go` — `countV1WorkgroupBlocks`, `detectV1Workgroups` added; `buildAndValidate` updated
- `pkg/config/loader_test.go` — `TestV1WorkgroupDetection` suite method with 5 table-driven cases

## Decisions Made

Used raw byte scanning (`countV1WorkgroupBlocks`) rather than the planned `diskConfig` struct field approach. The plan specified `WorkgroupList []map[string]any \`toml:"workgroup,omitempty"\`` on `diskConfig`, but go-toml cannot unmarshal `[workgroup.NAME]` (keyed table) into a `[]map[string]any` (slice). Existing tests write `[workgroup.fix-auth]` into `.gitw` directly, which caused `toml: cannot store a table in a slice` failures. Raw byte scanning is more robust: it matches only `[[workgroup]]` headers exactly, leaves `[workgroup.NAME]` untouched, and aligns with the Phase 6 precedent of targeted raw TOML scanning for strict checks.

## Deviations from Plan

### Auto-fixed Issues

**1. [Implementation] diskConfig struct field replaced with raw byte scanner**
- **Found during:** Task 1 (initial implementation)
- **Issue:** `[]map[string]any \`toml:"workgroup,omitempty"\`` on `diskConfig` caused `toml: cannot store a table in a slice` when parsing `[workgroup.fix-auth]` keyed-table format used in existing `pkg/git` tests
- **Fix:** Replaced with `countV1WorkgroupBlocks(data []byte) int` that scans for `[[workgroup]]` lines in raw TOML bytes before struct unmarshaling
- **Files modified:** `pkg/config/loader.go`
- **Verification:** `mage test` passes with all prior tests (including `TestInfoSuite`) and new `TestV1WorkgroupDetection`
- **Committed in:** `56375eb`

---

**Total deviations:** 1 auto-fixed (implementation approach)
**Impact on plan:** No scope change. All plan requirements and must-haves satisfied. Detection is more robust than the struct field approach.

## Issues Encountered

None beyond the deviation above.

## Next Phase Readiness

- CFG-10 delivered: `Load` returns a hard error for any `.gitw` containing `[[workgroup]]` blocks
- Phase 59 (`git w migrate`) can rely on this error firing at load time before any migration logic runs
- No blockers

---
*Phase: 10-detect-v1-workgroup-blocks*
*Completed: 2026-04-07*
