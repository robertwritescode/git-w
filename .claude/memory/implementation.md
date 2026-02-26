# Implementation Plan

## Status: Architecture review pending

---

## Testing Approach

All non-trivial logic must be unit-tested. See `testing.md` for the full strategy.
Key points:
- Test files sit alongside source (`foo_test.go` next to `foo.go`)
- Parse functions in `status.go` are separated from subprocess calls — tested with fixture strings
- Filesystem tests use `t.TempDir()`; git repo tests use `testutil.MakeGitRepo`
- CI runs `go test -race -count=1 ./...` (both `ci.yml` and `release.yml`)
- `mage test` mirrors CI (`-race -count=1`); `mage cover` adds `-coverprofile`

---

## Phase 1: Scaffold + Config + Basic Commands

**Goal**: Binary builds, config file can be created and edited, repos can be listed.

### Tasks
- [ ] `go mod init github.com/robertwritescode/git-workspace`
- [ ] Directory structure (`cmd/`, `internal/config/`, `internal/repo/`, etc.)
- [ ] `Makefile` with `build` and `install` targets
- [ ] `internal/config/types.go` — WorkspaceConfig, RepoConfig, GroupConfig, ContextConfig
- [ ] `internal/config/loader.go` — TOML load/save, atomic write (write to tmp, rename)
- [ ] `internal/config/discovery.go` — walk up from CWD to find `.gitworkspace`
- [ ] `cmd/root.go` — cobra root, global `--config` flag, load config into context
- [ ] `cmd/init.go` — create `.gitworkspace` in CWD with minimal scaffold; add `.gitworkspace.local` to `.gitignore` (create if absent)
- [ ] `cmd/add.go` — validate path is a git repo, detect remote URL, add to config, auto-gitignore path
- [ ] `cmd/remove.go` — remove repo entry from config
- [ ] `cmd/rename.go` — rename repo key in config
- [ ] `cmd/list.go` — print repo names or path of single repo (cobra alias: `ls`)
- [ ] `internal/testutil/helpers.go` — `MakeGitRepo`, `MakeWorkspace` shared test helpers
- [ ] `internal/config/loader_test.go` — TOML round-trip, atomic write, missing-file error, malformed TOML, `.local` override
- [ ] `internal/config/discovery_test.go` — walk-up 0/1/2 levels, env var override, not-found sentinel
- [ ] `cmd/init_test.go` — creates `.gitworkspace`, errors if already exists, adds `.local` to `.gitignore`
- [ ] `cmd/add_test.go`, `cmd/remove_test.go`, `cmd/rename_test.go`, `cmd/list_test.go` — cobra integration tests

**Exit criteria**: `git workspace init`, `add`, `rm`, `rename`, `ls` all work. All Phase 1 `_test.go` files pass `go test -race ./...`.

---

## Phase 2: Status Detection + `info` Display

**Goal**: `git workspace info` shows a color-coded status table.

### Tasks
- [ ] `internal/repo/repo.go` — Repo type, AbsPath resolution, IsGitRepo check
- [ ] `internal/repo/status.go` — detect dirty/staged/untracked/stashed via `git` subprocess
- [ ] `internal/repo/status.go` — detect remote state (InSync/LocalAhead/RemoteAhead/Diverged/NoRemote)
- [ ] `internal/display/colors.go` — ANSI color helpers wrapping fatih/color
- [ ] `internal/display/table.go` — tabwriter-based status table renderer
- [ ] `cmd/info.go` — fetch status for all (or group) repos, render table (cobra alias: `ll`)
- [ ] `internal/repo/repo_test.go` — `AbsPath` resolution, `IsGitRepo` true/false
- [ ] `internal/repo/status_test.go` — all `parsePorcelainV1` / `parseBranchLine` / `parseStashCount` states via fixture strings; one smoke test against a real temp git repo
- [ ] `internal/display/colors_test.go` — `visualWidth` strips ANSI codes correctly
- [ ] `internal/display/table_test.go` — golden string comparison with `color.NoColor = true`
- [ ] `cmd/info_test.go` — cobra integration test with a temp workspace

**Exit criteria**: `git workspace info` and `git workspace ll` both show a formatted, colored table. All Phase 2 `_test.go` files pass `go test -race ./...`.

---

## Phase 3: Parallel Execution Engine

**Goal**: Git commands run concurrently across repos.

### Tasks
- [ ] `internal/executor/result.go` — ExecResult type
- [ ] `internal/executor/parallel.go` — goroutine pool with semaphore (runtime.NumCPU())
- [ ] `internal/executor/parallel.go` — single-repo path: stdin passthrough, no prefix
- [ ] `internal/executor/parallel.go` — multi-repo path: stdin suppressed, output prefixed `[name]`
- [ ] `cmd/git_cmds.go` — register fetch, pull, push, status as cobra commands
- [ ] `cmd/exec.go` — `exec [repos...] -- <git-args>`
- [ ] `internal/executor/result_test.go` — output formatting, prefix insertion, non-zero exit representation
- [ ] `internal/executor/parallel_test.go` — `echo` command across multiple repos; all results collected; concurrency limit verified via atomic counter; output prefixed `[name]`; single-repo path has no prefix; timeout cancels goroutines
- [ ] `cmd/exec_test.go`, `cmd/git_cmds_test.go` — cobra integration tests with real temp git repos

**Exit criteria**: `git workspace fetch` runs in all repos concurrently with prefixed output. All Phase 3 `_test.go` files pass `go test -race ./...`.

---

## Phase 4: Groups + Context

**Goal**: Full group management and context-scoped command execution.
**Plan**: See `phase4-plan.md` for full parallel-stream breakdown, function signatures, and test specs.

### Tasks (3 parallel streams — see phase4-plan.md)

**Stream A: group subcommand**
- [ ] `cmd/group.go` — group subcommand tree (add/rm/rename/rmrepo/list/info); alias `g`
- [ ] `cmd/group_test.go` — GroupSuite; table-driven for all subcommands

**Stream B: context subcommand**
- [ ] `cmd/context.go` — context set/auto/none/show; writes `.gitworkspace.local`
- [ ] `cmd/context_test.go` — ContextSuite; table-driven show/set/auto/clear

**Stream C: context integration**
- [ ] `cmd/exec.go` — replace `filterRepos` with context-aware + group-name-expanding version
- [ ] `cmd/exec_test.go` — add tests: active-context scoping, group-name filter, dedup
- [ ] `cmd/git_cmds_test.go` — add table-driven active-context scoping test
- [ ] `cmd/info.go` — audit; update if it bypasses filterRepos and needs context awareness

**Exit criteria**: `git workspace group add frontend backend -n web` works;
`git workspace context web` scopes subsequent commands to that group. All Phase 4 `_test.go` files pass `go test -race ./...`.

---

## Phase 5: Advanced Features

**Goal**: Restore, recursive add, auto-context, custom git flags, shell completion.

### Tasks
- [ ] `cmd/clone.go` — `git clone <url> [<path>]`, register in config, auto-gitignore path, also support -g / --group for adding to a group
- [ ] `cmd/restore.go` — for each repo in `.gitworkspace`: clone if missing (requires `url`), pull if present; auto-gitignore each path
- [ ] `cmd/add.go` — `-r <dir>` flag: walk directory, find `.git` dirs (non-nesting), detect remote URL via `git remote get-url origin`, register each, auto-gitignore, auto-group by parent path, passing `-r` without an argument should use the current working directory
- [ ] `internal/config/types.go` — add `URL` to `RepoConfig`; add `AutoGitignore *bool` to `WorkspaceMeta`
- [ ] `internal/gitignore/` (or helper in `config/`) — `IsIgnored(root, path)` using `git check-ignore -q`; `EnsureIgnored(root, path)` to append if needed
- [ ] Per-repo `flags` wired into all git subcommand invocations
- [ ] Shell completion scaffolding via cobra's `GenBashCompletion` etc.
- [ ] `cmd/clone_test.go`, `cmd/restore_test.go` — clone into temp dir, verify config written, verify idempotent re-run
- [ ] `internal/gitignore/gitignore_test.go` — `IsIgnored` and `EnsureIgnored` with a real temp git repo

**Exit criteria**: `git workspace restore` clones all repos; re-running is idempotent; `.gitignore` is enforced. All Phase 5 `_test.go` files pass `go test -race ./...`.

---

## Notes

- Atomic config writes: write to `<file>.tmp`, then `os.Rename` to prevent corruption
- Config is always loaded fresh per invocation (no daemon, no caching)
- All git operations invoked as subprocesses via `os/exec` — no go-git library
  (keeps behavior identical to user's installed git version)
- Testing library: `github.com/stretchr/testify` (`assert` + `require`) throughout
  — `require` for fatal setup assertions, `assert` for non-fatal value checks
- See `testing.md` for detailed per-package testing notes and patterns
