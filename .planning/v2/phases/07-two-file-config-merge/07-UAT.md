---
status: complete
phase: 07-two-file-config-merge
source: 07-01-SUMMARY.md, 07-02-SUMMARY.md
started: 2026-04-06T05:15:00Z
updated: 2026-04-06T05:20:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Absent .git/.gitw is silently skipped
expected: In a workspace that has a .gitw but no .git/.gitw file, all git-w commands (e.g. `git w repo list`) work normally with no error and no mention of a missing private config file.
result: pass

### 2. Private remote override is visible after merge
expected: Create a .git/.gitw file that overrides an existing remote's `url` field. Run `git w info` (or any command that surfaces remote info). The command sees the private URL, not the shared .gitw URL.
result: pass

### 3. Private repo field override is visible
expected: Create a .git/.gitw that overrides a field on an existing repo (e.g. sets a path or flag). Run `git w repo list`. The overridden field value from .git/.gitw takes effect.
result: pass

### 4. Unknown repo name in .git/.gitw is a load error
expected: Add a [[repo]] block in .git/.gitw with a name that does not exist in .gitw. Running any git-w command produces a clear error at load time (not a silent ignore) referencing the unknown repo name.
result: pass

### 5. New remote in .git/.gitw is appended
expected: Add a [[remote]] block in .git/.gitw with a name not in .gitw. Commands that list remotes (e.g. `git w info --remotes`) show the private remote alongside the shared ones.
result: skipped
reason: No command surfaces the remote list directly; `git w info --remotes` does not exist. Remote merge is exercised only indirectly via sync/fetch operations — covered by unit tests, not observable through CLI listing.

### 6. .gitw.local context still wins over .git/.gitw
expected: Set an active context in .gitw.local. Also set a different context-equivalent override in .git/.gitw. The .gitw.local value is what git-w commands respect — it is the final override layer.
result: skipped

## Summary

total: 6
passed: 4
issues: 0
pending: 0
skipped: 2
blocked: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
