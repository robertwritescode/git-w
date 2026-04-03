---
status: complete
phase: 02-add-track-branch-and-upstream-fields
source: 02-01-PLAN.md, 02-02-PLAN.md
started: 2026-04-03T00:00:00Z
updated: 2026-04-03T00:10:00Z
---

## Current Test

[testing complete]

## Tests

### 1. [[repo]] format in .gitw loads correctly
expected: Create a .gitw using [[repo]] blocks (see above). Run `git w repo list`. Repos appear in the list without errors.
result: pass

### 2. clone_url field is stored on [[repo]] entries
expected: Add `clone_url = "https://github.com/org/repo"` to a [[repo]] block. Run `git w repo list`. The command works; the URL is silently stored (not displayed in list, but no errors).
result: pass

### 3. [[repo]] missing name field gives a load-time error
expected: Create a [[repo]] block with no `name =` field. Run any `git w` command. You see an error message containing something like "missing required name field".
result: pass

### 4. Duplicate [[repo]] names give a load-time error
expected: Create two [[repo]] blocks with the same `name = "api"`. Run any `git w` command. You see an error message containing "duplicate".
result: pass

### 5. track_branch and upstream fields load correctly
expected: Add `track_branch = "dev"` and `upstream = "infra"` to a [[repo]] block. Run `git w repo list`. The command succeeds — no errors, fields are stored.
result: pass

### 6. track_branch without upstream gives a load-time error
expected: Add `track_branch = "dev"` to a [[repo]] block but omit `upstream`. Run any `git w` command. You see an error about both fields needing to be set together.
result: pass

### 7. upstream without track_branch gives a load-time error
expected: Add `upstream = "infra"` to a [[repo]] block but omit `track_branch`. Run any `git w` command. You see an error about both fields needing to be set together.
result: pass

### 8. Duplicate track_branch within same upstream group gives a load-time error
expected: Two [[repo]] blocks both have `upstream = "infra"` and `track_branch = "dev"`. Run any `git w` command. You see an error mentioning the duplicate track_branch within the upstream group.
result: pass

### 9. Same track_branch in different upstream groups is allowed
expected: One [[repo]] has `upstream = "infra"` + `track_branch = "dev"`. Another has `upstream = "platform"` + `track_branch = "dev"`. Run any `git w` command. No error — same track_branch is fine when upstream groups differ.
result: pass

## Summary

total: 9
passed: 9
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
