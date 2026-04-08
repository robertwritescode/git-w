---
phase: 02-add-track-branch-and-upstream-fields
plan: 02
subsystem: config
tags: [config, toml, repo, track-branch, upstream, alias, validation]

requires:
  - phase: 02-01
    provides: RepoConfig with Name and CloneURL fields; [[repo]] load/save pipeline
provides:
  - TrackBranch and Upstream fields on RepoConfig (TOML: track_branch, upstream)
  - IsAlias() method on RepoConfig
  - validateAliasFields: D-01 co-presence check and D-02 uniqueness-per-upstream-group
  - Full CFG-02 delivery: env alias repo annotation with load-time validation
affects:
  - Phase 17 (sync behavior using track_branch deferred here)
  - Phase 13 (INT-03: alias validation interaction with path warnings)

tech-stack:
  added: []
  patterns:
    - "IsAlias() bool: method on RepoConfig value receiver, returns TrackBranch != ''"
    - "validateAliasFields: two-pass — D-01 co-presence per repo, D-02 uniqueness via seen[upstream][trackBranch] map"
    - "validateAliasFields wired as last step in buildAndValidate (after validateAgenticFrameworks)"

key-files:
  created: []
  modified:
    - pkg/config/config.go
    - pkg/config/loader.go
    - pkg/config/config_test.go
    - pkg/config/loader_test.go

key-decisions:
  - "D-01 error message: 'track_branch and upstream must both be set or both be absent'"
  - "D-02 error message: 'track_branch %q already used by %q in upstream group %q'"
  - "validateAliasFields is last in buildAndValidate (after validateAgenticFrameworks)"
  - "IsAlias() reports true iff TrackBranch is non-empty (Upstream co-presence guaranteed by validation)"
  - "Sync behavior using track_branch explicitly deferred to Phase 17"

patterns-established:
  - "Alias repo pattern: [[repo]] with track_branch + upstream identifies env aliases (Pattern A)"
  - "Two-pass validator for co-presence + uniqueness within a named group"

requirements-completed:
  - CFG-02

duration: included in single Phase 02 commit
completed: 2026-04-03
---

# Phase 02 Plan 02: Add TrackBranch/Upstream Fields and IsAlias Validation Summary

**`track_branch` and `upstream` fields on `[[repo]]` with co-presence (D-01) and per-group uniqueness (D-02) validation — env alias annotation fully deliverable at config load time**

## Performance

- **Duration:** (part of combined Phase 02 delivery)
- **Started:** 2026-04-03
- **Completed:** 2026-04-03
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- `TrackBranch string toml:"track_branch,omitempty"` and `Upstream string toml:"upstream,omitempty"` added to `RepoConfig`
- `IsAlias() bool` method added — returns `r.TrackBranch != ""`
- `validateAliasFields` implements D-01 (both-or-neither) and D-02 (unique `track_branch` per `upstream` group), wired as last step in `buildAndValidate`
- Full test coverage: `TestRepoConfigIsAlias` (config_test.go), `TestAliasFieldValidation` (7 table-driven cases in loader_test.go), `TestAliasFieldsRoundTrip` (TOML load→save→reload preserves fields)

## Task Commits

No per-task commits — all Phase 02 (Plan 01 + Plan 02) delivered in one atomic PR commit:
- `9b23729` — Add track_branch and upstream fields to [[repo]] (#102)

## Files Created/Modified
- `pkg/config/config.go` — TrackBranch, Upstream fields; IsAlias() method
- `pkg/config/loader.go` — validateAliasFields; wired into buildAndValidate
- `pkg/config/config_test.go` — TestRepoConfigIsAlias (true/false cases)
- `pkg/config/loader_test.go` — TestAliasFieldValidation (7 cases), TestAliasFieldsRoundTrip

## Decisions Made
- `validateAliasFields` placed last in `buildAndValidate` (after `validateAgenticFrameworks`) per plan spec
- Sync behavior using `track_branch` is explicitly deferred to Phase 17; this phase only stores and validates
- Error messages include the upstream group name for D-02 to aid user debugging

## Deviations from Plan
None — plan executed exactly as written.

## Issues Encountered
None.

## Next Phase Readiness
- CFG-02 complete: `track_branch` and `upstream` fields parse, validate, and round-trip correctly
- `IsAlias()` available for downstream phases that need to distinguish alias repos from plain repos
- Phase 17 can build sync behavior on `TrackBranch` / `Upstream` fields without schema changes

---
*Phase: 02-add-track-branch-and-upstream-fields*
*Completed: 2026-04-03*
