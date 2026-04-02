# Pitfalls Research

**Domain:** Go CLI multi-repo management tool (git-w v2)
**Researched:** 2026-04-01
**Confidence:** HIGH

## Critical Pitfalls

### Pitfall 1: Worktree `.git` File Breakage After Migration Directory Moves

**What goes wrong:**
v2 migration moves repo directories from `~/code/repo` to `~/code/workspace/repo`. Git worktrees store absolute paths in their `.git` files (e.g., `gitdir: /Users/x/code/repo/.git/worktrees/feature`). After the parent directory moves, every worktree's `.git` file points to a stale path. The worktree appears broken — `git status` fails, commits fail, work is stranded.

**Why it happens:**
Git worktrees use a bidirectional link: the worktree's `.git` file points to the main repo's `.git/worktrees/<name>/` directory, and the main repo's `.git/worktrees/<name>/gitdir` file points back to the worktree. Moving either side without running `git worktree repair` breaks both directions.

**How to avoid:**
The migration spec (v2-migration.md) correctly includes `git worktree repair` as a post-move step. Enforce this as a hard requirement: the migration function MUST call `git worktree repair` after every directory move, and MUST verify the repair succeeded by checking that all worktree `.git` files resolve. Never treat repair as optional cleanup.

**Warning signs:**
- Tests that move directories but skip `git worktree repair`
- `git worktree list` showing worktrees with `[prunable]` status after migration
- Integration tests that only test repos without worktrees

**Phase to address:**
M2 (Config v2 + Migration) — migration implementation must include repair as atomic with the move

---

### Pitfall 2: TOML Comment Loss at Scale With v2 Schema Expansion

**What goes wrong:**
v1 already has a fragile 525-line custom TOML parser (`applySmartUpdate`) that silently swallows errors to preserve comments. v2 adds `[remotes]`, `[sync_pairs]`, `[workstreams]`, `[branch_rules]`, and nested tables like `[remotes.origin.branch_rules]`. Each new section type multiplies the surface area for comment preservation bugs. Users lose carefully maintained comments on config update, or worse, the parser silently corrupts config structure.

**Why it happens:**
TOML libraries parse-then-serialize, destroying comments. The custom smart-update approach does string-level manipulation to preserve them, but every new table type needs its own handling. The complexity grows combinatorially with nested tables.

**How to avoid:**
Two strategies, choose one early:
1. **Accept comment loss on programmatic writes.** Document that `git w` commands that modify config may strip comments. Keep a `.gitw.bak` before writes. This is the pragmatic choice.
2. **Use a comment-preserving TOML library** (e.g., `github.com/pelletier/go-toml/v2` with its AST-level manipulation). This is the correct long-term fix but requires replacing the custom parser.

Either way: never silently swallow parse errors. If the smart-update fails, abort the write and tell the user.

**Warning signs:**
- `applySmartUpdate` growing past 800+ lines with v2 sections
- Bug reports about lost comments appearing after sync operations
- Tests that only verify key-value correctness but never check comment preservation

**Phase to address:**
M1 (Foundation) — decide the comment strategy before M2 adds new config sections. If replacing the parser, do it in M1 so M2 builds on solid ground.

---

### Pitfall 3: Concurrent Config File Writes (Race Condition)

**What goes wrong:**
Two `git w sync` processes run simultaneously (e.g., user runs sync in two terminals, or a hook triggers sync while manual sync is running). Both read `.gitw`, both compute updates, both write. Last write wins — the first process's changes are silently lost. v2 makes this worse because `sync` now writes config state AND hook files AND potentially `.gitw-state.json`.

**Why it happens:**
v1 has no file locking on config writes. This was acceptable when writes were rare (manual `repo add`). v2 adds `sync` as a frequent operation that writes config as a side effect, and hooks that trigger sync-like behavior, dramatically increasing write frequency and concurrency risk.

**How to avoid:**
Implement advisory file locking using `flock(2)` (via `syscall.Flock` on Unix) on a `.gitw.lock` file. Acquire lock before any config read-modify-write cycle. Use a short timeout (5s) with a clear error message: "Another git-w process is updating config. Retry in a moment."

Keep the lock scope minimal — lock only around the read-modify-write, not the entire command execution. This prevents deadlocks from long-running operations.

**Warning signs:**
- Config files that "lose" repos intermittently
- State file (`gitw-state.json`) with stale timestamps after concurrent syncs
- Tests that never exercise concurrent command execution

**Phase to address:**
M1 (Foundation) — file locking is infrastructure that every subsequent milestone depends on

---

### Pitfall 4: Hook Scripts That Break in Worktree Context

**What goes wrong:**
The `pre-push` hook installed in `.git/hooks/` applies to ALL worktrees from that repo. But `$GIT_DIR` inside a worktree points to `.git/worktrees/<name>/`, not the main `.git/`. If the hook script uses `$GIT_DIR` to find config or resolve paths, it gets the wrong location in worktree contexts. The hook silently does nothing, or worse, errors out and blocks pushes.

**How to avoid:**
The hook script must use `git rev-parse --git-common-dir` to find the shared `.git/` directory (where hooks actually live), and `git rev-parse --show-toplevel` to find the working tree root. Never rely on `$GIT_DIR` for path resolution in hook scripts.

Test hooks explicitly in worktree contexts — create a repo, add a worktree, install hooks, push from the worktree. This must be an integration test, not just a unit test of the hook template.

**Why it happens:**
Developers test hooks in the main working tree where `$GIT_DIR` == `.git/` and everything works. The worktree case is only discovered in production when a user pushes from a worktree for the first time.

**Warning signs:**
- Hook script using `$GIT_DIR` or hardcoded `.git/` paths
- No integration tests that exercise hooks from within worktrees
- Users reporting "hook does nothing" but only when pushing from worktrees

**Phase to address:**
M5 (Hook Management) — hook implementation must test in worktree context from day one

---

### Pitfall 5: Migration Partial Failure Leaves Inconsistent State

**What goes wrong:**
Migration moves directories, rewrites config, repairs worktrees, and installs hooks across potentially dozens of repos. If it fails midway (disk full, permission error, power loss), some repos are in the new v2 layout and some are in v1. The config may reference paths that don't exist. The tool can't operate in either v1 or v2 mode.

**Why it happens:**
The migration spec describes 5 distinct path cases and 2 abort conditions, but filesystem operations aren't transactional. `os.Rename` succeeds or fails per-directory, not atomically across all repos.

**How to avoid:**
Implement migration as a two-phase process:
1. **Plan phase:** Validate all moves are possible (check disk space, permissions, path collisions, bare repos). Write a migration plan file (`.gitw-migration.json`) listing all operations.
2. **Execute phase:** Perform moves one-by-one, updating the plan file with completion status after each. If interrupted, re-running migration reads the plan and resumes from where it left off.

The plan file acts as a write-ahead log. `git w` startup should check for an incomplete migration plan and prompt the user to resume or rollback.

**Warning signs:**
- Migration function that does moves in a loop without tracking progress
- No resume capability after interruption
- Tests that only test successful full migration, never partial failure

**Phase to address:**
M2 (Config v2 + Migration) — migration implementation must be resumable from day one

---

### Pitfall 6: Provider API Differences Leaking Through the Abstraction

**What goes wrong:**
The `Provider` interface abstracts GitHub and Gitea, but their APIs differ in fundamental ways: GitHub uses org-scoped endpoints (`/orgs/{org}/repos`), Gitea uses user-scoped. GitHub has fine-grained rate limiting with `X-RateLimit-*` headers, Gitea has simpler throttling. GitHub returns different error codes. Pagination differs (GitHub uses Link headers, Gitea supports both Link headers and `?page=N`). The abstraction either becomes so thin it's useless, or so thick it hides important differences.

**Why it happens:**
The initial Provider interface is designed around one provider's API shape (usually GitHub, since it's more familiar), then the other provider is awkwardly shoe-horned in. Provider-specific behavior leaks through error messages, rate limiting strategies, and pagination patterns.

**How to avoid:**
Design the `Provider` interface from the Gitea side first — it's the simpler API. GitHub can always be constrained to match Gitea's capabilities. The interface should expose:
- `ListRepos(ctx, owner string) ([]Repo, error)` — provider handles org vs user distinction internally
- `GetRepo(ctx, owner, name string) (Repo, error)`
- Rate limiting should be internal to each provider implementation, not exposed in the interface

Use a `ProviderError` type that wraps provider-specific errors with a uniform code (NotFound, RateLimited, AuthFailed, NetworkError). Never expose raw HTTP status codes through the interface.

**Warning signs:**
- Interface methods that take GitHub-specific parameters (e.g., `orgName` vs `userName`)
- Error handling that switches on HTTP status codes outside the provider package
- Rate limiting logic in the command layer instead of the provider layer

**Phase to address:**
M6 (Remote Management: Sync) — design the Provider interface, but M10 (Gitea + GitHub Providers) implements both. The interface must be validated against both APIs before M6 ships.

---

### Pitfall 7: Context Cancellation Ignored in Network Operations

**What goes wrong:**
v1 already uses `context.Background()` in places where `cmd.Context()` should be used (documented in CONCERNS.md). v2 adds long-running network operations: API calls to GitHub/Gitea for repo listing, sync fan-out across multiple remotes, hook reconciliation. If these operations ignore context cancellation, `Ctrl+C` hangs the CLI while HTTP requests complete, or worse, partial writes occur without cleanup.

**Why it happens:**
`context.Background()` works fine in tests and short operations. The problem only surfaces when operations take seconds (network calls) or when users interrupt mid-operation. Since v1 was all local git operations (fast), this was a minor annoyance. v2's network operations make it a UX disaster.

**How to avoid:**
Establish a rule in M1: every function that does I/O or subprocess calls takes `context.Context` as its first parameter (this is already in AGENTS.md coding standards). Enforce with a linter or code review checklist. Specifically:
- All `http.NewRequest` calls must use `http.NewRequestWithContext`
- All `exec.Command` calls must use `exec.CommandContext`
- Replace existing `context.Background()` usages in v1 code as encountered

**Warning signs:**
- `context.Background()` appearing in new code
- `http.NewRequest` without `WithContext` variant
- Integration tests that don't test cancellation mid-operation

**Phase to address:**
M1 (Foundation) — fix existing context issues, establish the pattern. Every subsequent milestone must follow it.

---

### Pitfall 8: `.gitw-stream` Manifest Drift From Disk Reality

**What goes wrong:**
`.gitw-stream` manifests are committed to the repo (they describe workstream composition), but the worktrees they reference are ephemeral local state. A user clones a repo with a `.gitw-stream`, but the worktrees described in it don't exist on their machine. Or a user deletes worktrees manually without updating the manifest. `git w restore` must reconcile this, but if the manifest is treated as truth, restore tries to create worktrees that conflict with existing directories, or fails silently when source branches don't exist.

**Why it happens:**
The manifest conflates "what this workstream should look like" (intent) with "what exists on disk" (state). Without a clear separation, code treats the manifest as both, leading to confusion about whether to create, skip, or error when reality doesn't match.

**How to avoid:**
Treat the manifest as pure intent (declarative), and disk state as separate. `git w restore` computes a diff between intent and reality:
- Worktree in manifest but not on disk → create it
- Worktree on disk but not in manifest → leave it (warn, don't delete)
- Worktree in manifest AND on disk → verify paths match, repair if needed

Never delete worktrees that exist on disk just because the manifest changed. Deletion is always an explicit user action.

**Warning signs:**
- `restore` implementation that doesn't check disk state before creating worktrees
- Tests that always start from a clean state (no pre-existing worktrees)
- No handling for "branch doesn't exist yet" when restoring

**Phase to address:**
M4 (Worktree Lifecycle) — restore implementation must handle all mismatch cases

---

### Pitfall 9: Branch Rule Evaluation Order Ambiguity

**What goes wrong:**
Branch rules have three levels (repo-level overrides → remote-level rules → default allow) and three criteria per rule (pattern, untracked, explicit). The spec says "first-match-wins in declaration order." But TOML maps are unordered — `[remotes.origin.branch_rules]` as a TOML map has no guaranteed key order. If rules are stored as a map, "declaration order" is meaningless. Users write rules expecting top-to-bottom evaluation but get random order.

**Why it happens:**
TOML tables (maps) are unordered by spec. Developers assume the parser preserves insertion order (some do, some don't, and it's not guaranteed). The v2 schema uses `[[branch_rules]]` (array of tables) which IS ordered, but the nesting under `[remotes.origin]` could lead to confusion about where the array boundary is.

**How to avoid:**
Use `[[remotes.origin.branch_rules]]` (array of tables, not map) and verify that the TOML library preserves array order (go-toml/v2 does). Document explicitly that rules are evaluated in file order. Add a `git w config validate` command that prints rule evaluation order so users can verify their intent.

Write tests that specifically verify evaluation order with 3+ rules where order matters (e.g., rule 1 allows `main`, rule 2 denies `*` — reversing them changes behavior).

**Warning signs:**
- Branch rules stored as `map[string]Rule` instead of `[]Rule`
- Tests with only 1-2 rules that don't exercise ordering
- No way for users to see effective rule evaluation order

**Phase to address:**
M3 (Schema Expansion) — schema must use ordered arrays. M7 (Branch Rules) — evaluation engine must respect order and provide debugging tools.

---

### Pitfall 10: Two-File Config Merge Produces Surprising Overrides

**What goes wrong:**
`.gitw` (shared) and `.git/.gitw` (private) merge at field level, matched by `name` key. A user adds `private = true` to a remote in `.gitw` instead of `.git/.gitw`. The remote's URL is now committed to version control. Or worse: `.git/.gitw` overrides a field that the user expected to come from `.gitw`, and they can't figure out why their shared config change isn't taking effect.

**Why it happens:**
Two-file merge is inherently confusing. Users don't have a mental model for which file "wins" for which fields. The merge happens at load time with no visibility into the merge result. Debugging requires manually reading both files and mentally computing the merge.

**How to avoid:**
1. **Enforce `private` placement at load time:** If a remote has `private = true`, it MUST be in `.git/.gitw`. If found in `.gitw`, emit a warning: "Private remote 'origin' should be in .git/.gitw to avoid committing credentials."
2. **Add `git w config show --merged`:** Display the effective merged config with annotations showing which file each value came from.
3. **Document the merge rules prominently:** Not buried in a spec — in the `git w config --help` output.

**Warning signs:**
- No validation of which file contains `private` remotes
- No way to see the merged config result
- User confusion reports about "config changes not taking effect"

**Phase to address:**
M3 (Schema Expansion) — implement merge validation. M6 (Remote Management) — enforce `private` placement.

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip file locking on config writes | Faster M1 delivery | Silent data loss in concurrent usage; debugging nightmare | Never — implement in M1 |
| Use `context.Background()` for new network code | Less plumbing | Ctrl+C hangs, partial writes, untestable timeouts | Never — always pass `cmd.Context()` |
| Hardcode GitHub as the only provider initially | Ship sync faster | Provider interface shaped to GitHub's API; Gitea integration requires rewrite | Acceptable if interface is designed for both, only GitHub implemented |
| Skip comment preservation (always rewrite TOML) | Eliminate 525-line parser | Users lose config comments on every write; frustration and manual fixups | Acceptable if `.gitw.bak` is created before writes |
| Store branch rules as maps instead of arrays | Simpler parsing | Evaluation order is undefined; subtle bugs in rule matching | Never — use `[[array_of_tables]]` from the start |
| Embed provider credentials in config instead of credential helper | Quick setup | Credentials in plaintext on disk; security liability | Never — use git credential helper or keychain |
| Test hooks only in main worktree | Faster test suite | Hooks break silently in worktree contexts; users discover in production | Only during initial development; worktree tests required before release |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| GitHub REST API | Using unauthenticated requests (60 req/hour limit) | Always require a token; detect rate limiting via `X-RateLimit-Remaining` header and back off proactively |
| Gitea API | Assuming same endpoints as GitHub | Gitea's repo listing is `/api/v1/repos/search?owner=X`, not `/orgs/X/repos`; verify each endpoint against Gitea's Swagger docs |
| Git worktree repair | Calling repair on the worktree directory | `git worktree repair` must be run from the main working tree, not from inside the worktree being repaired |
| Git hooks (pre-push) | Forgarding only `$@` args | Pre-push hook receives remote name + URL as args AND ref data on stdin; must forward both `"$@"` and stdin to the handler |
| TOML config merge | Assuming deep merge (nested table override) | Merge is field-level per block matched by `name` key; a `.git/.gitw` block replaces the entire matching block, not individual nested fields |
| Git credential helper | Storing tokens directly in config | Use `git credential fill` to retrieve tokens at runtime; never persist tokens in `.gitw` or `.git/.gitw` |
| Agent interop (`.gitw-stream`) | Assuming agent reads the same config as CLI | Agents read `.gitw-stream` manifest only; CLI must ensure manifest is self-contained with all info the agent needs |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Sequential API calls during sync fan-out | `git w sync` takes 30+ seconds with 5 remotes | Parallelize API calls per-remote with `errgroup`; respect per-provider rate limits | >3 remotes or >50 repos per remote |
| Full repo list fetch on every sync | Sync re-fetches all repos from API even if nothing changed | Use `If-None-Match` / ETag headers; cache in `.gitw-state.json` with TTL | >100 repos across remotes |
| Spawning git subprocess per-repo for status checks | `git w status` across 30 repos takes 10+ seconds | Batch status checks; use `git -C <path> status --porcelain` which is faster than full status | >20 repos in workspace |
| Loading full config on every command invocation | Startup latency grows with config size | Lazy-load config sections; only parse `[repos]` for repo commands, not `[remotes]` and `[branch_rules]` | >50 repos with complex branch rules |
| Re-running `git worktree repair` on every sync | Repair is slow when many worktrees exist | Only repair after directory moves; track "needs repair" flag in migration state | >10 worktrees per repo |
| Unbounded concurrent git subprocess spawning | OS file descriptor exhaustion; system slowdown | Use a semaphore (bounded `chan struct{}`) to limit concurrent git operations (already in v1 as `RunParallel`) | >50 concurrent repos |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing API tokens in `.gitw` (shared config) | Tokens committed to version control; exposed to anyone with repo access | Tokens must only live in `.git/.gitw` (private) or OS keychain; validate at load time |
| Hook scripts with world-writable permissions | Malicious local user injects code into hook; runs on next push | Install hooks with `0755` permissions; verify permissions on hook load |
| Sync fetching repos over HTTP instead of HTTPS | Credentials transmitted in plaintext | Default to HTTPS; warn if any remote URL uses `http://`; refuse to sync over HTTP without `--insecure` flag |
| Shell injection in hook scripts via repo names | Repo name containing `$(...)` or backticks executes arbitrary code in hook | Always quote variables in hook scripts; use `exec` instead of shell evaluation where possible |
| Migration moving directories outside workspace root | Path traversal via crafted repo paths (e.g., `../../etc/passwd`) | Validate all target paths are within the workspace root before any move; reject paths containing `..` |
| Provider error messages exposing tokens | Error log contains `Authorization: Bearer <token>` from failed HTTP request | Scrub authorization headers from error messages; never log raw HTTP requests |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Migration with no dry-run | User runs migration on 30 repos and can't predict what will happen | `git w migrate --dry-run` shows all planned moves; require confirmation for actual migration |
| Silent hook installation | User doesn't know hooks were installed; surprised when push is intercepted | Print "Installed pre-push hook for [repo]" during sync; `git w hook list` to see installed hooks |
| Sync deletes repos removed from remote | User loses local work in repos that were archived upstream | Sync never deletes local repos; only adds new ones. Show "Remote removed: [repo] (still local)" |
| Branch rule blocks push with no explanation | User gets "push rejected" with no indication of which rule matched | Error message must include: which rule matched, from which file, and how to override (`--force-push` or adjust rules) |
| Config validation errors reference TOML line numbers | Line numbers are meaningless to non-technical users | Reference by repo name and field: "Remote 'origin' in repo 'api' is missing 'url'" |
| `git w restore` recreates worktrees without asking | User had intentionally removed worktrees; restore puts them back | `restore` shows diff between manifest and disk, asks for confirmation; `--yes` flag for automation |

## "Looks Done But Isn't" Checklist

- [ ] **Migration:** Often missing resume-after-interruption — verify migration can be re-run safely after partial failure
- [ ] **Hook installation:** Often missing worktree context testing — verify hooks work when pushing from a worktree, not just the main working tree
- [ ] **Provider interface:** Often missing rate limit handling — verify sync degrades gracefully when rate-limited, not just when API returns 200
- [ ] **Config merge:** Often missing conflict visibility — verify users can see which file each config value came from
- [ ] **Branch rules:** Often missing order-dependent test cases — verify 3+ rules where declaration order changes behavior
- [ ] **Sync fan-out:** Often missing partial failure handling — verify that one failed remote doesn't abort sync for all remotes
- [ ] **Worktree restore:** Often missing "branch doesn't exist" handling — verify restore behavior when manifest references branches that were deleted upstream
- [ ] **State file (`.gitw-state.json`):** Often missing concurrent write protection — verify state file has same locking as config
- [ ] **Error messages:** Often missing actionable next steps — verify every error tells the user what to do, not just what went wrong
- [ ] **Context cancellation:** Often missing cleanup on cancel — verify that Ctrl+C during sync doesn't leave config in a half-written state

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Broken worktree links after migration | LOW | Run `git worktree repair` from main working tree; `git worktree prune` to clean stale entries |
| Lost comments in TOML config | LOW | Restore from `.gitw.bak` (if strategy implemented) or `git checkout -- .gitw` if committed |
| Concurrent write data loss | MEDIUM | Restore `.gitw` from git history; re-run `git w sync` to regenerate state; cannot recover uncommitted local-only changes |
| Partial migration failure | HIGH | Read `.gitw-migration.json` plan file; manually move remaining repos or re-run migration with resume; verify all paths with `git worktree list` |
| Hook script causing push failures | LOW | `git push --no-verify` as immediate workaround; `git w hook uninstall` to remove; debug hook with `GIT_TRACE=1 git push` |
| Provider token exposed in logs | HIGH | Rotate token immediately on provider (GitHub/Gitea); audit git history for committed tokens; force-push to remove if committed |
| Branch rules blocking legitimate pushes | LOW | `git push --no-verify` bypasses pre-push hook; edit `.gitw` to adjust rules; `git w config validate` to check rule order |
| Manifest/disk drift in workstreams | MEDIUM | `git w restore --dry-run` to see diff; manually reconcile by either updating manifest or creating missing worktrees |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| TOML comment loss at scale | M1 (Foundation) | Decide comment strategy; test with v2 schema sections; verify no silent error swallowing |
| Concurrent config writes | M1 (Foundation) | File locking implemented; test two simultaneous writes don't lose data |
| Context cancellation ignored | M1 (Foundation) | Lint or review check for `context.Background()` in new code; Ctrl+C test for every network operation |
| Worktree `.git` breakage on migration | M2 (Migration) | Integration test: move repo dir, verify worktree repair, verify worktree is functional |
| Migration partial failure | M2 (Migration) | Test: interrupt migration mid-way, re-run, verify clean completion |
| Two-file config merge surprises | M3 (Schema Expansion) | `git w config show --merged` works; `private` in `.gitw` produces warning |
| Branch rule evaluation order | M3 (Schema) + M7 (Branch Rules) | Ordered array in schema (M3); order-dependent tests pass (M7) |
| Manifest/disk drift | M4 (Worktree Lifecycle) | `restore` handles: missing worktrees, extra worktrees, missing branches, path conflicts |
| Hook scripts break in worktrees | M5 (Hook Management) | Integration test: install hook on main repo, push from worktree, verify hook executes correctly |
| Provider API differences | M6 (Sync) + M10 (Providers) | Interface designed (M6); both GitHub and Gitea pass same integration test suite (M10) |
| Sync deleting local repos | M6 (Remote Sync) | Test: remove repo from remote, sync, verify local repo still exists with warning |
| Branch rules block with no explanation | M7 (Branch Rules) | Blocked push error message includes: rule text, source file, override instructions |
| Shell injection via repo names | M5 (Hook Management) | Test: repo name containing `$(rm -rf /)` doesn't execute; all hook vars quoted |
| Provider token exposure in logs | M10 (Providers) | Test: failed API call with token doesn't include token in error output |
| Agent interop manifest completeness | M11 (Workstreams) | Agent can reconstruct workspace from `.gitw-stream` alone without reading `.gitw` |

## Sources

- `.planning/codebase/CONCERNS.md` — documented v1 tech debt (comment preservation, context.Background usage)
- `.planning/v2/v2-migration.md` — migration spec with path cases and abort conditions
- `.planning/v2/v2-remote-management.md` — sync fan-out, hook mechanism, provider interface design
- `.planning/v2/v2-schema.md` — config schema including branch_rules and workstream manifests
- `.planning/v2/v2-milestones.md` — milestone definitions M1-M12
- Git official documentation: `git-worktree(1)` — worktree path resolution, repair semantics
- Git official documentation: `githooks(5)` — pre-push hook contract (args + stdin)
- Gitea SDK documentation — API endpoint differences from GitHub
- GitHub REST API documentation — rate limiting, pagination, org-scoped endpoints

---
*Pitfalls research for: git-w v2 (Go CLI multi-repo management)*
*Researched: 2026-04-01*
