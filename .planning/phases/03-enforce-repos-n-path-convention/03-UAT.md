---
status: complete
phase: 03-enforce-repos-n-path-convention
source: 03-01-SUMMARY.md
started: 2026-04-03T00:00:00Z
updated: 2026-04-03T00:01:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Warning on non-conforming repo path
expected: In a workspace where a repo has a path that does NOT start with repos/<name> (e.g. path = "my-service"), running any git-w command (e.g. `git w repo list`) should print a warning to stderr like: `warning: repo "my-service" path "my-service" does not follow the repos/<n> convention`. The command still succeeds — warnings are non-blocking.
result: pass

### 2. No warning for conforming repo path
expected: In a workspace where all repos have paths under repos/<name> (e.g. path = "repos/my-service"), running any git-w command produces no warnings — stderr is clean.
result: pass

### 3. Non-blocking behavior
expected: A workspace with non-conforming repo paths still allows all git-w commands to run successfully. The warning does not prevent the command from completing. For example, `git w repo list` lists repos and exits 0.
result: pass

### 4. Worktree repos excluded from warnings
expected: In a workspace with a worktree set, the synthesized worktree repos (e.g. infra-dev, infra-test) do NOT trigger path convention warnings, even though their paths are not under repos/<name>.
result: pass

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
