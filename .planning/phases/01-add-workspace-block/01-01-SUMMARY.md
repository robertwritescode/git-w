---
phase: 01-add-workspace-block
plan: 01
subsystem: config
tags: [toml, structs, config, metarepo, workspace-block]

requires: []
provides:
  - MetarepoConfig struct replacing WorkspaceMeta (TOML key: metarepo)
  - WorkspaceBlock struct for [[workspace]] array-of-tables
  - WorkspaceConfig.Workspaces []WorkspaceBlock field
  - DefaultRemotes and AgenticFrameworks fields on MetarepoConfig
  - Updated diskConfig and prepareDiskConfig for new field names
affects:
  - 01-02
  - 01-03

tech-stack:
  added: []
  patterns:
    - "TOML array-of-tables [[workspace]] parsed into []WorkspaceBlock slice"
    - "MetarepoConfig is value type (not pointer) — no nil check needed in ensureWorkspaceMaps"

key-files:
  created: []
  modified:
    - pkg/config/config.go
    - pkg/config/loader.go
    - pkg/config/config_test.go
    - pkg/config/loader_test.go
    - pkg/testutil/cmd.go
    - pkg/testutil/helpers.go
    - pkg/workspace/init.go

key-decisions:
  - "WorkspaceMeta deleted entirely — MetarepoConfig is the replacement type, not an alias"
  - "TOML key for top-level metarepo settings is 'metarepo' (was 'workspace')"
  - "TOML key 'workspace' is now reserved for [[workspace]] array-of-tables"
  - "All 13 test files updated to use [metarepo] instead of [workspace] TOML key"

requirements-completed:
  - CFG-01

duration: 5min
completed: 2026-04-02
---

# Phase 01 Plan 01: Struct Renames + WorkspaceBlock Summary

**Renamed WorkspaceMeta to MetarepoConfig (TOML key: metarepo), added WorkspaceBlock struct and Workspaces []WorkspaceBlock field, updated all 13 test fixtures from [workspace] to [metarepo]**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-02T21:31:07Z
- **Completed:** 2026-04-02T21:36:06Z
- **Tasks:** 2
- **Files modified:** 19

## Accomplishments
- `WorkspaceMeta` type fully deleted; `MetarepoConfig` replaces it with `DefaultRemotes` and `AgenticFrameworks` fields added
- `WorkspaceConfig.Metarepo MetarepoConfig toml:"metarepo"` and `Workspaces []WorkspaceBlock toml:"workspace"` fields established
- `diskConfig` updated to use `Metarepo` and `Workspaces`; `prepareDiskConfig` populates both fields
- `TestWorkspacesBlocksParse` added and passing: verifies `[[workspace]]` array-of-tables round-trip with two entries
- All 13 test files across 6 packages updated from `[workspace]` to `[metarepo]` TOML key; full test suite passes

## Task Commits

1. **Tasks 1 + 2: All struct renames, loader updates, and test fixture updates** - `e96d70c` (feat)

## Files Created/Modified
- `pkg/config/config.go` — MetarepoConfig and WorkspaceBlock structs, updated WorkspaceConfig
- `pkg/config/loader.go` — diskConfig and prepareDiskConfig updated
- `pkg/config/config_test.go` — Updated to MetarepoConfig; TestWorkspaceBlockFields added
- `pkg/config/loader_test.go` — All fixtures use [metarepo]; TestWorkspacesBlocksParse added
- `pkg/testutil/cmd.go` — newWorkspaceTOML helper uses [metarepo]
- `pkg/testutil/helpers.go` — setupWorkspaceDir uses [metarepo]
- `pkg/workspace/init.go` — writeInitialConfig template uses [metarepo]
- 12 test files in pkg/branch, pkg/git, pkg/repo, pkg/workgroup, pkg/config — fixtures updated

## Decisions Made
- All existing `[workspace]` references in test fixtures updated to `[metarepo]` — this was a blocking cascade from the struct rename (Rule 3 deviation, auto-fixed)
- `pkg/toml/preserve_test.go` left untouched — it uses its own local struct types with `[workspace]` for TOML preservation logic tests, not the config schema

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated 12 additional test files across 6 packages**
- **Found during:** Task 2 (running mage testfast after loader.go updates)
- **Issue:** 12 test files beyond pkg/config used `[workspace]` TOML key in fixtures; all failed with "toml: cannot store a table in a slice" after the WorkspaceConfig.Workspaces field was introduced
- **Fix:** Updated all string literals from `[workspace]` to `[metarepo]` in pkg/branch, pkg/git, pkg/repo, pkg/workgroup test files plus pkg/testutil helpers and pkg/workspace/init.go
- **Files modified:** 12 test files + testutil/cmd.go + testutil/helpers.go + workspace/init.go
- **Verification:** mage testfast passes all 13 packages
- **Committed in:** e96d70c

---

**Total deviations:** 1 auto-fixed (blocking cascade from struct rename)
**Impact on plan:** Auto-fix was necessary and expected — renaming the TOML key required all fixtures to follow. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- MetarepoConfig and WorkspaceBlock structs in place; pkg/agents bootstrap (01-02) can proceed immediately
- AgenticFrameworks []string field on MetarepoConfig is ready for validation wiring (01-03)
- All tests passing; no blockers

---
*Phase: 01-add-workspace-block*
*Completed: 2026-04-02*
