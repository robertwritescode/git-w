---
phase: 02-add-track-branch-and-upstream-fields
plan: 01
subsystem: config
tags: [config, toml, repo, array-of-tables, clone-url, testutil]

requires:
  - phase: 01-add-workspace-block
    provides: diskConfig split pattern (disk struct → in-memory WorkspaceConfig) established for WorkspaceBlock
provides:
  - RepoConfig.Name and RepoConfig.CloneURL fields with [[repo]] array-of-tables TOML format
  - buildReposIndex: converts [[repo]] slice to cfg.Repos map, validates missing/duplicate names
  - validateRepoNames: name presence and uniqueness enforcement in buildAndValidate
  - RepoByName accessor on WorkspaceConfig
  - prepareDiskConfig outputs [[repo]] sorted by name, synthesized repos excluded
  - testutil helpers updated to [[repo]] format
affects:
  - 02-02 (adds TrackBranch/Upstream on top of this RepoConfig shape)
  - all phases using testutil cmd.go/helpers.go

tech-stack:
  added: []
  patterns:
    - "diskConfig split: [[repo]] array-of-tables unmarshals into diskConfig.RepoList; buildReposIndex builds in-memory cfg.Repos map"
    - "Name field as first field on RepoConfig (TOML: name) is the required identifier"
    - "URL renamed to CloneURL (TOML: clone_url) for v2 naming clarity"

key-files:
  created: []
  modified:
    - pkg/config/config.go
    - pkg/config/loader.go
    - pkg/config/config_test.go
    - pkg/config/loader_test.go
    - pkg/testutil/cmd.go
    - pkg/repo/add.go
    - pkg/repo/clone.go
    - pkg/repo/restore.go
    - pkg/repo/add_test.go
    - pkg/repo/restore_test.go
    - pkg/workspace/init.go

key-decisions:
  - "TOML key is 'repo' (not 'repos'), producing [[repo]] array-of-tables syntax"
  - "diskConfig.RepoList []RepoConfig toml:'repo,omitempty' — disk-only field; WorkspaceConfig.Repos is in-memory only"
  - "buildReposIndex errors on missing name with 'missing required name field'; errors on duplicate with 'duplicate [[repo]] name'"
  - "validateRepoNames runs first in buildAndValidate (before validateWorktreePaths)"
  - "RepoByName is a thin map lookup wrapper on *WorkspaceConfig"
  - "prepareDiskConfig → buildRepoList excludes synthesized repos, sorts by name"

patterns-established:
  - "diskConfig split: [[repo]] loaded via RepoList, then indexed into cfg.Repos by buildReposIndex — same pattern as WorkspaceBlock"
  - "testutil appendRepoTOML and makeWorkspaceWithRepoNames write [[repo]] format with name field"

requirements-completed:
  - CFG-02

duration: included in single Phase 02 commit
completed: 2026-04-03
---

# Phase 02 Plan 01: Migrate RepoConfig to [[repo]] Array-of-Tables Summary

**`[[repo]]` array-of-tables TOML format with required `name` field and `clone_url` replacing `url` — load/save pipeline, validation, RepoByName accessor, and testutil helpers all updated**

## Performance

- **Duration:** (part of combined Phase 02 delivery)
- **Started:** 2026-04-03
- **Completed:** 2026-04-03
- **Tasks:** 2
- **Files modified:** 11+

## Accomplishments
- Migrated RepoConfig from v1 `[repos.<name>]` map format to v2 `[[repo]]` array-of-tables with required `name` field
- `URL` field renamed to `CloneURL` (TOML: `clone_url`) throughout config, add, clone, and restore paths
- `buildReposIndex` populates in-memory `cfg.Repos` map from `diskConfig.RepoList`; `validateRepoNames` enforces presence and uniqueness as the first step in `buildAndValidate`
- `RepoByName(name string) (RepoConfig, bool)` accessor added to `*WorkspaceConfig`
- `prepareDiskConfig` writes `[[repo]]` array sorted by name, synthesized repos excluded
- `testutil` helpers (`appendRepoTOML`, `makeWorkspaceWithRepoNames`) updated to emit `[[repo]]` format; all downstream test packages updated

## Task Commits

No per-task commits — all Phase 02 (Plan 01 + Plan 02) delivered in one atomic PR commit:
- `9b23729` — Add track_branch and upstream fields to [[repo]] (#102)

## Files Created/Modified
- `pkg/config/config.go` — RepoConfig: added Name, renamed URL→CloneURL; RepoByName accessor
- `pkg/config/loader.go` — diskConfig.RepoList, buildReposIndex, validateRepoNames, updated prepareDiskConfig/buildRepoList
- `pkg/config/config_test.go` — updated fixtures to [[repo]] format
- `pkg/config/loader_test.go` — TestRepoByName, updated fixtures, TestAliasFieldValidation (covers Plan 02)
- `pkg/testutil/cmd.go` — appendRepoTOML and makeWorkspaceWithRepoNames write [[repo]] format
- `pkg/repo/add.go` — uses CloneURL
- `pkg/repo/clone.go` — uses CloneURL
- `pkg/repo/restore.go` — reads rc.CloneURL
- `pkg/repo/add_test.go` — fixture updated
- `pkg/repo/restore_test.go` — fixture updated
- `pkg/workspace/init.go` — (no [[repo]] template entries needed)
- Various branch/git/workgroup test files — testutil helper cascade

## Decisions Made
- TOML key `repo` (not `repos`) so that `[[repo]]` is the array-of-tables syntax
- `WorkspaceConfig.Repos` is in-memory only (no TOML tag) — same diskConfig split pattern as Phase 01's WorkspaceBlock
- `buildReposIndex` is called before `ensureWorkspaceMaps` and `buildAndValidate`
- `validateRepoNames` is wired as the first step in `buildAndValidate`

## Deviations from Plan
None — plan executed exactly as written.

## Issues Encountered
None.

## Next Phase Readiness
- RepoConfig has `Name`, `CloneURL` fields; Plan 02 adds `TrackBranch` and `Upstream` on top
- All downstream packages use `CloneURL`; no remaining `rc.URL` references outside WorktreeConfig

---
*Phase: 02-add-track-branch-and-upstream-fields*
*Completed: 2026-04-03*
