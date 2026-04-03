---
phase: 03-enforce-repos-n-path-convention
plan: 01
subsystem: config
tags: [config, warnings, path-convention, repos, testutil]

requires:
  - phase: 02-add-track-branch-upstream-fields
    provides: "[[repo]] TOML format and RepoConfig fields used in test fixtures"
provides:
  - WorkspaceConfig.Warnings []string field (in-memory only, no TOML tag)
  - warnNonConformingRepoPaths function in pkg/config/loader.go
  - LoadConfig prints each warning to cmd.ErrOrStderr() via output.Writef
  - testutil helpers updated to create repos under repos/<n> paths (conforming)
affects:
  - 03-02
  - any phase that adds repo entries to test fixtures

tech-stack:
  added: []
  patterns:
    - "Load-time warnings appended to cfg.Warnings (non-blocking); printed to stderr in LoadConfig"
    - "Synthesized worktree repos excluded from path-convention checks via WorktreeRepoToSetIndex"
    - "testutil workspace helpers create repos at repos/<name> to conform with v2 path convention"

key-files:
  created: []
  modified:
    - pkg/config/config.go
    - pkg/config/loader.go
    - pkg/config/loader_test.go
    - pkg/testutil/cmd.go
    - pkg/testutil/helpers.go
    - pkg/branch/checkout_test.go
    - pkg/branch/create_test.go
    - pkg/git/commit_test.go
    - pkg/git/sync_test.go
    - pkg/workgroup/checkout_test.go
    - pkg/workgroup/create_test.go
    - pkg/workgroup/drop_test.go
    - pkg/workgroup/helpers_test.go

key-decisions:
  - "Warnings are non-blocking: config.Load never returns an error for non-conforming paths"
  - "cfg.Warnings is sorted before returning to ensure deterministic output (map iteration is non-deterministic)"
  - "Worktree-synthesized repos are excluded from path checks; their paths are owned by the worktree set convention"
  - "testutil helpers updated to emit conforming repos/<name> paths so existing tests don't generate spurious warnings"

patterns-established:
  - "Load-time warnings pattern: append to cfg.Warnings in buildAndValidate; print in LoadConfig to cmd.ErrOrStderr()"
  - "sort.Strings(cfg.Warnings) before returning from warnNonConformingRepoPaths to neutralize map iteration order"

requirements-completed:
  - CFG-03

duration: ~90min
completed: 2026-04-03
---

# Phase 03 Plan 01: Enforce repos/<n> Path Convention Summary

**Load-time warnings for non-conforming repo paths: cfg.Warnings field, warnNonConformingRepoPaths function, stderr output in LoadConfig, and testutil fixtures migrated to repos/<name>**

## Performance

- **Duration:** ~90 min
- **Started:** 2026-04-03
- **Completed:** 2026-04-03
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments
- `WorkspaceConfig.Warnings []string` field added (in-memory only, no TOML tag) for load-time advisory messages
- `warnNonConformingRepoPaths` checks every plain repo path against the `repos/<n>` convention and appends formatted warnings; worktree-synthesized repos are skipped
- `LoadConfig` prints each warning to `cmd.ErrOrStderr()` via `output.Writef` after loading
- `mage test` passes (all packages, race detector) with 8 test files updated across `testutil`, `branch`, `git`, and `workgroup` packages

## Task Commits

All implementation changes are uncommitted (staged for a single commit per plan instructions).

## Files Created/Modified
- `pkg/config/config.go` — Added `Warnings []string` field to `WorkspaceConfig` after `Workgroups`
- `pkg/config/loader.go` — Added `warnNonConformingRepoPaths`, wired into `buildAndValidate`, updated `LoadConfig` to print warnings; added `sort` and `output` imports
- `pkg/config/loader_test.go` — Added `TestPathConventionWarnings` (7 table-driven cases) and `TestPathConventionWarnings_SkipsSynthesizedRepos`; added `"strings"` import
- `pkg/testutil/cmd.go` — `makeWorkspaceWithNLocalRepos`, `makeWorkspaceWithNRemoteRepos`, `makeWorkspaceWithRepoNames` updated to use `repos/<name>` paths
- `pkg/testutil/helpers.go` — `makeGitRepo` creates repos under `repos/` subdir when inside a workspace
- `pkg/branch/checkout_test.go` — `writeWorkspaceConfig` writes `path = "repos/<name>"`
- `pkg/branch/create_test.go` — `repoPath` helper returns `filepath.Join(wsDir, "repos", name)`; `appendRepoEntries` writes `path = "repos/<name>"`
- `pkg/git/commit_test.go` — All `filepath.Join(wsDir, name)` repo path constructions updated to include `"repos"` segment; `setupWorkgroup` fixed
- `pkg/git/sync_test.go` — Direct repo path constructions and `makeWorkspaceWithRemoteRepoAndSyncPush` config writing updated to `repos/<name>`
- `pkg/workgroup/checkout_test.go` — Direct repo path constructions and `setupRemoteWorkspace` updated
- `pkg/workgroup/create_test.go` — `repoDir` construction updated
- `pkg/workgroup/drop_test.go` — `repoDir` construction updated
- `pkg/workgroup/helpers_test.go` — `currentBranchAt` calls and `rewriteConfigWithDefaultBranch` config writing updated

## Decisions Made
- Warnings are sorted before returning from `warnNonConformingRepoPaths` to neutralize non-deterministic map iteration order — without this, warning order in tests was flaky
- `testutil` workspace helpers migrated to `repos/<name>` paths so existing test suites don't emit spurious warnings that would bleed into output assertions; this was a necessary cascade

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Non-deterministic warning order from map iteration**
- **Found during:** Task 1 (writing `TestPathConventionWarnings` multi-repo case)
- **Issue:** `warnNonConformingRepoPaths` iterates `cfg.Repos` (a map); output order is non-deterministic, making `cfg.Warnings[0]` assertions flaky
- **Fix:** Added `sort.Strings(cfg.Warnings)` at the end of `warnNonConformingRepoPaths`
- **Files modified:** `pkg/config/loader.go`
- **Verification:** `mage testfast` passes consistently; multi-repo test asserts deterministic warning order

**2. [Rule 3 - Blocking] testutil helpers created repos at non-conforming paths**
- **Found during:** Task 2 (running `mage testfast` after Task 1)
- **Issue:** `makeWorkspaceWithNLocalRepos`, `makeWorkspaceWithNRemoteRepos`, `makeWorkspaceWithRepoNames`, and `makeGitRepo` created repos at `wsDir/<name>` with `path = "<name>"` — non-conforming. After Task 1, these fixtures produced warnings that bled into output string assertions across 8 test files.
- **Fix:** Updated testutil helpers to create repos under `wsDir/repos/<name>` with `path = "repos/<name>"`. Then fixed all downstream test files that constructed absolute paths or wrote config TOML directly.
- **Files modified:** `pkg/testutil/cmd.go`, `pkg/testutil/helpers.go`, `pkg/branch/checkout_test.go`, `pkg/branch/create_test.go`, `pkg/git/commit_test.go`, `pkg/git/sync_test.go`, `pkg/workgroup/checkout_test.go`, `pkg/workgroup/create_test.go`, `pkg/workgroup/drop_test.go`, `pkg/workgroup/helpers_test.go`
- **Verification:** `mage test` passes all packages under race detector

---

**Total deviations:** 2 auto-fixed (1 correctness fix for map ordering, 1 blocking cascade from testutil migration)
**Impact on plan:** Both fixes were necessary. The testutil migration is the correct long-term state — conforming paths in all test fixtures means future phases get clean output with no spurious warnings. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `cfg.Warnings` infrastructure is in place; any future load-time advisory can append to it and it will print to stderr automatically via `LoadConfig`
- All test fixtures now use conforming `repos/<name>` paths; new tests should follow this pattern
- Phase 03-02 (migrate command) can rely on `cfg.Warnings` to identify repos needing migration

---
*Phase: 03-enforce-repos-n-path-convention*
*Completed: 2026-04-03*
