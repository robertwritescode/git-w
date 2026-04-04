---
phase: 04-add-remote-and-remote-branch-rule
plan: "02"
subsystem: config
tags: [config, validation, remote, branch-rule, security]
dependency_graph:
  requires: [04-01]
  provides: [validateRemotes in loader.go]
  affects: [pkg/config/loader.go, pkg/config/loader_test.go]
tech_stack:
  added: []
  patterns: [buildAndValidate extension, private file path detection]
key_files:
  created: []
  modified:
    - pkg/config/loader.go
    - pkg/config/loader_test.go
decisions:
  - "validateRemotes is a single consolidated function (not split into sub-helpers) keeping it readable within ~20 lines per function guideline"
  - "private enforcement uses strings.HasSuffix(filepath.ToSlash(cfgPath), .git/.gitw) exactly as specified in D-09"
metrics:
  duration: 167s
  completed: "2026-04-04"
  tasks_completed: 1
  files_modified: 2
---

# Phase 04 Plan 02: validateRemotes Validation Summary

`validateRemotes` wired into `buildAndValidate` enforcing all 5 checks from D-08 and D-09; CFG-04 complete.

## What Was Built

**pkg/config/loader.go:**
- `validateRemotes(cfgPath string, cfg *WorkspaceConfig) error` function with:
  1. Non-empty name check (index in error message)
  2. Unique name check (duplicate name in error message)
  3. Valid kind enum check: `gitea`, `forgejo`, `github`, `generic`
  4. Valid branch_rule action enum check: `allow`, `block`, `warn`, `require-flag`
  5. `private=true` enforcement: error if in `.gitw` (not `.git/.gitw`)
- `buildAndValidate` calls `validateRemotes(configPath, cfg)` before `validateAliasFields`

**pkg/config/loader_test.go (LoaderSuite):**
- `TestRemoteValidation` — 8 table-driven cases: 2 valid (no rules, with rules), missing name, duplicate name, invalid kind, invalid action, private-in-public-file, private-in-private-file-ok

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Self-Check: PASSED
