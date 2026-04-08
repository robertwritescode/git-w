---
phase: 04-add-remote-and-remote-branch-rule
plan: "01"
subsystem: config
tags: [config, schema, remote, branch-rule, types]
dependency_graph:
  requires: []
  provides: [RemoteConfig, BranchRuleConfig, BranchAction, MergeRemote, RemoteByName, diskConfig.RemoteList]
  affects: [pkg/config/config.go, pkg/config/loader.go]
tech_stack:
  added: []
  patterns: [array-of-tables, *bool pointer for optional bools, pure merge function]
key_files:
  created: []
  modified:
    - pkg/config/config.go
    - pkg/config/config_test.go
    - pkg/config/loader.go
    - pkg/config/loader_test.go
decisions:
  - "Remotes []RemoteConfig lives directly on WorkspaceConfig (no diskConfig split) matching WorkspaceBlock pattern"
  - "BranchRules nil override keeps base; non-nil replaces entirely"
  - "UseSSH/Critical/Private use bool (not *bool) since false is a valid explicit override vs. omit"
metrics:
  duration: 337s
  completed: "2026-04-04"
  tasks_completed: 2
  files_modified: 4
---

# Phase 04 Plan 01: Add RemoteConfig, BranchRuleConfig Types and Wire Loader Summary

Config schema layer for `[[remote]]` and `[[remote.branch_rule]]` parsing: types defined, loader wired, round-trip verified.

## What Was Built

**pkg/config/config.go:**
- `BranchAction` typed string alias with `ActionAllow`, `ActionBlock`, `ActionWarn`, `ActionRequireFlag` constants
- `BranchRuleConfig` struct with `*bool` for `Untracked`/`Explicit` (nil = not-set)
- `RemoteConfig` struct with all `[[remote]]` fields including nested `BranchRules []BranchRuleConfig`
- `Remotes []RemoteConfig` on `WorkspaceConfig` (in-memory, no TOML tag)
- `Remotes []string` on `RepoConfig` with `toml:"remotes,omitempty"`
- `MergeRemote(base, override RemoteConfig) RemoteConfig` pure function
- `RemoteByName(name string) (RemoteConfig, bool)` accessor on `WorkspaceConfig`

**pkg/config/loader.go:**
- `RemoteList []RemoteConfig \`toml:"remote,omitempty"\`` added to `diskConfig`
- `loadMainConfig` populates `cfg.Remotes = dc.RemoteList`
- `prepareDiskConfig` includes `RemoteList: cfg.Remotes` for Save round-trip

## Tests Added

**config_test.go (ConfigSuite):**
- `TestBranchActionConstants` — verifies all 4 string values
- `TestMergeRemote` — 9 table-driven cases covering empty override, field wins, bool fields, BranchRules nil/non-nil behavior
- `TestRemoteByName` — found, not found, first-match semantics

**loader_test.go (LoaderSuite):**
- `TestRemoteBlocksParse` — 5 cases: no blocks, single remote, single with branch rules, multiple remotes, `*bool` fields
- `TestRemoteRoundTrip` — Save/Load cycle preserves all remote and branch rule fields

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — types are fully defined; downstream phases (7, 9, 12/13) will consume them.

## Self-Check: PASSED
