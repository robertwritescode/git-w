# Codebase Concerns

**Analysis Date:** 2026-04-01

## Tech Debt

**Silent Error Swallowing in TOML Comment Preservation:**
- Issue: `applySmartUpdate` in `pkg/toml/preserve.go:77-83` silently discards smartUpdate failures by returning `newBytes, nil` — comments are lost with no user-visible warning.
- Files: `pkg/toml/preserve.go`, `pkg/config/loader.go`
- Impact: Config saves silently strip user comments on any TOML parsing edge case. Users may not notice until comments are gone permanently.
- Fix approach: Log a warning or surface the error rather than silently falling back to `newBytes`.

**`interface{}` Instead of `any` Throughout TOML Package:**
- Issue: `pkg/toml/preserve.go` uses `interface{}` in 15+ function signatures (`Marshal`, `Unmarshal`, `UpdatePreservingComments`, `saveWithCommentPreservation`, etc.) rather than the idiomatic Go 1.18+ `any` alias.
- Files: `pkg/toml/preserve.go`, `pkg/config/loader.go`
- Impact: Minor style inconsistency with the rest of the codebase which uses `any` elsewhere.
- Fix approach: Replace all `interface{}` with `any` in these files.

**`context.Background()` Used Inside Command Handlers:**
- Issue: Several functions create fresh `context.Background()` contexts instead of threading the command's context through. Affected call sites: `pkg/worktree/clone.go:151,155,185,196`, `pkg/worktree/add.go:123`, `pkg/repo/clone.go:40`.
- Files: `pkg/worktree/clone.go`, `pkg/worktree/add.go`, `pkg/repo/clone.go`
- Impact: Ctrl-C / signal cancellation does not propagate to git subprocesses launched from these paths. Long clones cannot be interrupted cleanly.
- Fix approach: Pass `cmd.Context()` (or the caller's `ctx`) down to `gitutil.*` calls instead of `context.Background()`.

**`buildContext` in `pkg/git/executor.go` Ignores Caller's Context:**
- Issue: `buildContext` always derives from `context.Background()` (lines 73-75), so `RunParallel` (used for `commit`, `rollback`, `status`, etc.) cannot be cancelled by the command's context.
- Files: `pkg/git/executor.go`
- Impact: Same as above — Ctrl-C does not stop in-flight git subprocesses for commit/rollback/exec operations.
- Fix approach: Accept a parent context as parameter and use `context.WithTimeout(parent, ...)` / `context.WithCancel(parent)`.

**Cobra Flag Errors Silently Ignored:**
- Issue: Many flag reads use the `val, _ =` pattern, discarding errors. Affected sites: `pkg/git/sync.go:70-71`, `pkg/branch/checkout.go:78`, `pkg/branch/default.go:48`, `pkg/workspace/group.go:88-89,152,198-199`, `pkg/worktree/rm.go:63`, `pkg/worktree/drop.go:76`, `pkg/cmdutil/flags.go:12-13`, `pkg/workgroup/create.go:82`, `pkg/workgroup/common.go:100,111`, `pkg/workgroup/drop.go:72-73`, `pkg/repo/add.go:40-41`, `pkg/repo/clone.go:33`, `pkg/repo/restore.go:54`.
- Files: Multiple (see above)
- Impact: Flag type mismatches or programmatic flag registration bugs are invisible. In practice cobra flag errors on defined flags are rare, but the pattern masks real bugs during development.
- Fix approach: Propagate flag errors or panic on registration errors (the latter is idiomatic for cobra). The `parseCommitFlags` in `pkg/git/commit.go` shows the correct pattern.

**`repo rename` Does Not Rename the Directory on Disk:**
- Issue: `pkg/repo/rename.go` renames the workspace key and updates group references but leaves the directory at its original path. A prominent `NOTE:` message warns about this in the output.
- Files: `pkg/repo/rename.go`
- Impact: After rename, the workspace key and filesystem path diverge. The user must manually `mv` the directory. This is a known limitation surfaced in the README.
- Fix approach: Add an optional `--move` flag that also renames the directory and updates the `path` in config.

**`fetchWithAllRepoDedup` Uses Non-Wrapping `fmt.Errorf("%s", ...)`:**
- Issue: `pkg/git/commands.go:144` uses `fmt.Errorf("%s", strings.Join(...))` to combine failures, discarding all structured error information. Should use `errors.Join` (Go 1.20+) or wrap each failure with `%w`.
- Files: `pkg/git/commands.go`
- Impact: Callers cannot use `errors.Is`/`errors.As` on the returned error. Minor, since this is a terminal user-facing error.
- Fix approach: Replace with `errors.Join` or `parallel.FormatFailureError` (already used in sync/branch).

## Known Bugs

**Worktree Branches Not Pulled During `restore` if Already Exist:**
- Symptoms: `pkg/repo/restore.go:289-298` — `restoreExistingWorktree` calls `SetBranchTrackingToOrigin` then `Pull`, but if `SetBranchTrackingToOrigin` fails (e.g. remote branch doesn't exist yet), the whole restore for that branch fails even if the worktree is otherwise healthy.
- Files: `pkg/repo/restore.go`
- Trigger: Worktree branch exists locally with no upstream tracking ref yet set.
- Workaround: Manual `git branch --set-upstream-to` then re-run restore.

**`drop` for Workgroup Fails Silently If Repo Not in Config:**
- Symptoms: `pkg/workgroup/drop.go:133` calls `repoAbsPath` which returns an error if the repo key is not in `cfg.Repos`. This causes the whole drop to fail mid-way through, leaving some worktrees removed and others intact, with no partial-cleanup logic.
- Files: `pkg/workgroup/drop.go`
- Trigger: Manually editing `.gitw` to remove a repo while a workgroup referencing that repo still exists in `.gitw.local`.
- Workaround: Restore the repo key in `.gitw` before dropping the workgroup.

## Security Considerations

**`GIT_W_CONFIG` Environment Variable Accepted Without Validation:**
- Risk: `pkg/config/discovery.go:17` accepts `GIT_W_CONFIG` as a raw path override with no validation. If an attacker can set this env var, they can point the tool at an arbitrary TOML config file.
- Files: `pkg/config/discovery.go`
- Current mitigation: The config is only read, not executed. Path traversal is validated on repo paths inside the config. The tool runs as the current user, so the attack surface is limited to privilege escalation in shared environments.
- Recommendations: Document that `GIT_W_CONFIG` should not be inherited from untrusted environments.

**URL Values Passed Directly to `git clone`:**
- Risk: `pkg/gitutil/gitutil.go:93-99`, `pkg/worktree/clone.go:151` — URLs from config or CLI are passed directly as arguments to `exec.Command("git", "clone", url, ...)`. Git itself sanitizes these, but unusual URL schemes could trigger unexpected git behaviors.
- Files: `pkg/gitutil/gitutil.go`, `pkg/repo/clone.go`, `pkg/worktree/clone.go`
- Current mitigation: URLs pass through `exec.Command` argument list (no shell injection possible). Git validates URL format.
- Recommendations: Acceptable risk given the tool's local-developer focus.

**Config File Written With 0o600 Permissions But Read With Default umask:**
- Risk: Temp files in `pkg/config/loader.go:264` are chmoded to `0o600`, but the original config file may have been created with looser permissions. After atomic rename, the resulting file inherits the temp file's `0o600` mode.
- Files: `pkg/config/loader.go`
- Current mitigation: `0o600` is actually stricter than necessary, which is safe.
- Recommendations: Document this behavior; consider preserving the original file's permissions on save.

## Performance Bottlenecks

**`WorktreeBranchForRepo` is O(sets × branches) Per Call:**
- Problem: `pkg/config/config.go:108-118` iterates all worktree sets and all branches on every call. Called from `ResolveDefaultBranch`, `plainUnits`, `worktreeUnits` in branch operations.
- Files: `pkg/config/config.go`, `pkg/branch/create.go`, `pkg/branch/default.go`, `pkg/workgroup/create.go`, `pkg/workgroup/add.go`, `pkg/workgroup/checkout.go`
- Cause: No pre-built index is cached on `WorkspaceConfig`. `WorktreeRepoToSetIndex` exists as an indexed version but is not reused by `WorktreeBranchForRepo`.
- Improvement path: Compute the reverse index once at load time and store it on `WorkspaceConfig`, or at minimum call `WorktreeRepoToSetIndex` once per command and pass it down. Acceptable for current workspace sizes (comment in `pkg/worktree/common.go:21` acknowledges this).

**`hasRemote` Spawns a Subprocess Per Repo Per Operation:**
- Problem: `pkg/branch/create.go:485-486` and `pkg/branch/checkout.go` call `gitutil.HasRemote` → `gitutil.RemoteURL` → `exec.Command("git", "remote", "get-url", "origin")` once per repo per operation step. In branch create with `--sync-source`, this is called twice per repo (lines 280 and 292).
- Files: `pkg/branch/create.go`, `pkg/branch/checkout.go`, `pkg/branch/default.go`, `pkg/gitutil/gitutil.go`
- Cause: No caching of remote presence within a single command invocation.
- Improvement path: Check once at the start of each command and pass a boolean flag into per-repo functions, or cache `RemoteURL` on the `repo.Repo` struct.

**Worktree Bare Fetches Are Sequential Within `fetchWithAllRepoDedup`:**
- Problem: `pkg/git/commands.go:188-202` iterates bare paths sequentially in `fetchWorktreeBareTargets`, even though each is an independent network operation.
- Files: `pkg/git/commands.go`
- Cause: Simple `for` loop rather than fan-out.
- Improvement path: Use `parallel.RunFanOut` for multiple bare repos to fetch them concurrently.

## Fragile Areas

**TOML Comment Preservation (`pkg/toml/preserve.go`):**
- Files: `pkg/toml/preserve.go`
- Why fragile: The comment preservation algorithm (525 lines) re-implements TOML structural parsing via regex and line scanning. It uses anchor-based comment injection (`extractCommentAnchors`, `injectSectionComments`) that can misplace comments when section order changes or subsection names collide with key names.
- Safe modification: Always run the full test suite in `pkg/toml/` and `pkg/config/` after any changes. Add test cases covering the specific TOML shape before changing behavior.
- Test coverage: `pkg/toml/preserve_test.go` (275 lines) — reasonably covered but complex edge cases (inline tables, multi-line values, comment-only sections) may be missing.

**Parallel Commit with Rollback:**
- Files: `pkg/git/commit.go`
- Why fragile: `executeCommit` runs commits in parallel then attempts rollback on successes if any repo fails. The rollback (`git reset --soft HEAD~1`) is also parallel. If rollback fails for a repo, a message tells the user to run manually — but there is no mechanism to detect or retry partial rollback failures automatically.
- Safe modification: Add integration tests that simulate mixed success/failure across repos before changing `executeCommit` or `rollback`.
- Test coverage: `pkg/git/commit_test.go` covers rollback but relies on real git repos; concurrent failure scenarios may be under-tested.

**Atomic Config Write Under Concurrent Processes:**
- Files: `pkg/config/loader.go`
- Why fragile: `atomicWriteFile` uses write-to-temp + rename, which is atomic on POSIX. However, two concurrent `git w` processes writing to the same `.gitw` will race: the last rename wins and the earlier one's changes are silently dropped. No file locking is used.
- Safe modification: Acceptable for typical single-user interactive use, but could silently corrupt config under scripted parallel invocations. Document this limitation.
- Test coverage: No concurrency tests for config saving.

**Context Management in `pkg/workgroup/checkout.go`:**
- Files: `pkg/workgroup/checkout.go`
- Why fragile: `attachRemoteBranch` in `pkg/workgroup/common.go:145-155` performs a fetch then adds a worktree as separate non-atomic steps. If the process is interrupted between fetch and `AddWorktree`, the bare repo is updated but no worktree is created, leaving the workgroup in a partially initialized state.
- Safe modification: Check for existing worktree registration on retry rather than erroring.
- Test coverage: `pkg/workgroup/checkout_test.go` covers the happy path but not interrupted/partial states.

## Scaling Limits

**Workspace Size:**
- Current capacity: All repos and groups are loaded into a single `WorkspaceConfig` map for every command, including commands targeting a single repo.
- Limit: Practically fine for workspaces with hundreds of repos. The `validateRepoPaths` call on every `Save` iterates all repos — O(n) file I/O on every config write.
- Scaling path: No changes needed for typical developer workspaces. If workspace grows very large, consider lazy loading or caching validated paths.

**Parallel Worker Ceiling:**
- Current capacity: `parallel.MaxWorkers` caps at `runtime.NumCPU()` by default.
- Limit: For network-bound operations (fetch, clone, push), CPU count is the wrong bound — a user on a 4-core machine fetching 100 repos will serialize unnecessarily.
- Scaling path: Use a separate, higher configurable parallelism limit for network operations. The `-j` flag on `repo restore` is the right model.

## Dependencies at Risk

**`go-toml/v2` TOML Parsing:**
- Risk: The custom comment-preservation layer in `pkg/toml/preserve.go` duplicates structural TOML knowledge (section headers, key parsing) that diverges from what `go-toml` understands internally. Breaking changes in `go-toml`'s marshal output format (e.g., key ordering, quoting style) could silently break comment injection.
- Impact: Comments may be lost or misplaced after a `go-toml` version bump.
- Migration plan: Pin `go-toml` version carefully; add a golden-file test for marshal output shape.

**No Vulnerability Scanning in CI:**
- Risk: `pkg/go.sum` is present and dependencies are reasonably minimal, but the CI workflow (`.github/workflows/ci.yml`) does not run `govulncheck` or `go mod verify`.
- Impact: Known CVEs in dependencies would not be detected automatically.
- Migration plan: Add `govulncheck ./...` step to `ci.yml`.

## Missing Critical Features

**No Config Validation on `GIT_W_CONFIG` Override Path:**
- Problem: When `GIT_W_CONFIG` points to a nonexistent file, the error is a raw `os.ReadFile` error with the full path, not a user-friendly diagnostic.
- Blocks: Smooth onboarding in CI/CD environments where `GIT_W_CONFIG` is set externally.

**No `--dry-run` on Destructive Commands:**
- Problem: `worktree drop`, `workgroup drop`, `repo rm` have no dry-run mode. Only `git w commit` has `--dry-run`.
- Blocks: Safe inspection before removing worktrees with uncommitted changes.

## Test Coverage Gaps

**`pkg/cmd/root.go` — Zero Coverage:**
- What's not tested: `os.Exit(1)` path and cobra root setup in `pkg/cmd/root.go`. Coverage file shows 0 coverage on all lines.
- Files: `pkg/cmd/root.go`
- Risk: Command-line flag wiring (`--config`, `--no-color`, completion) could regress silently.
- Priority: Low (cobra wiring is stable).

**`pkg/worktree/common.go` and `pkg/workgroup/common.go` — No Direct Tests:**
- What's not tested: Shared helpers (`findByRepoName`, `bareAbsPath`, `defaultBranchAbsPath`, `runStep`, `recordStep`, `resolveExistingTreePath`) have no dedicated unit tests; they are exercised only indirectly through command tests.
- Files: `pkg/worktree/common.go`, `pkg/workgroup/common.go`
- Risk: Edge cases in `resolveExistingTreePath` (wrong branch on existing path) or `findByRepoName` (set/branch naming collisions) could go undetected.
- Priority: Medium.

**`pkg/git/executor.go` Context Cancellation — Not Tested:**
- What's not tested: The `buildContext` function always creates a `context.Background()` derivation. There are no tests verifying that a cancelled parent context actually stops in-flight git subprocesses.
- Files: `pkg/git/executor.go`
- Risk: Ctrl-C behavior is untested; silent hang on timeout edge cases.
- Priority: Medium.

**`pkg/config/loader.go` Concurrent Write Race — Not Tested:**
- What's not tested: Two concurrent `Save` calls to the same config path.
- Files: `pkg/config/loader.go`
- Risk: Silent config corruption under scripted parallel invocations.
- Priority: Low for interactive use, Medium for CI scripting.

---

*Concerns audit: 2026-04-01*
