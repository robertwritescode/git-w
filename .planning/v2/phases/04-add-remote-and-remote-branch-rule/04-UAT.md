---
status: complete
phase: 04-add-remote-and-remote-branch-rule
source: 04-01-SUMMARY.md, 04-02-SUMMARY.md
started: 2026-04-04T04:55:40Z
updated: 2026-04-04T05:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Remote block parses into config
expected: A .gitw file with a [[remote]] block loads without error. Any git-w command (e.g. `git w repo list`) succeeds when the config contains a valid [[remote]] entry.
result: pass

### 2. Remote with branch_rule entries parses
expected: A [[remote]] block with nested [[remote.branch_rule]] entries loads cleanly. Example config with `action = "allow"` and `patterns = ["main"]` should not produce any error on load.
result: pass

### 3. Validation rejects missing remote name
expected: A [[remote]] block with no `name` field causes `git w` to exit with an error message referencing the missing name (e.g. "remote at index 0: name is required" or similar).
result: pass

### 4. Validation rejects duplicate remote names
expected: Two [[remote]] blocks with the same `name` value causes `git w` to exit with an error mentioning the duplicate name.
result: pass

### 5. Validation rejects invalid remote kind
expected: A [[remote]] block with `kind = "bitbucket"` (not in the allowed set: gitea, forgejo, github, generic) causes `git w` to exit with an error mentioning the invalid kind.
result: pass

### 6. Validation rejects invalid branch_rule action
expected: A [[remote.branch_rule]] with `action = "ignore"` (not in: allow, block, warn, require-flag) causes `git w` to exit with an error mentioning the invalid action.
result: pass

### 7. Validation rejects private=true in shared .gitw
expected: A [[remote]] block with `private = true` in a standard `.gitw` file (not inside `.git/.gitw`) causes `git w` to exit with an error about private remotes not being allowed in the shared config.
result: pass

### 8. private=true is allowed in .git/.gitw
expected: A [[remote]] block with `private = true` in a `.git/.gitw` file (the private config path) loads without error.
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
