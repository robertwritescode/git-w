# Architecture

**Analysis Date:** 2026-04-01

## Pattern Overview

**Overall:** CLI Plugin with Domain-Oriented Package Architecture

**Key Characteristics:**
- Single binary CLI tool (`git-w`) designed to be invoked as `git w <cmd>` via git's plugin system
- Each feature domain is a self-contained package that registers its own subcommands onto a shared cobra root
- Commands follow a consistent pattern: load config → resolve repos → execute in parallel → collect reports → write output
- Configuration is file-driven (`.gitw` TOML file) discovered by walking up the filesystem tree from CWD
- Parallel execution is a first-class concern; most multi-repo operations use fan-out goroutines

## Layers

**Entry Point:**
- Purpose: Wires version and kicks off cobra execution
- Location: `main.go`
- Contains: Single `main()` delegating to `cmd.Execute(version)`
- Depends on: `pkg/cmd`
- Used by: OS/shell

**Command Registration Layer:**
- Purpose: Builds the cobra command tree by wiring domain packages
- Location: `pkg/cmd/root.go`
- Contains: `newRootCmd`, `Execute`, top-level flag (`--config`), completion command
- Depends on: All domain packages (`workspace`, `repo`, `worktree`, `branch`, `workgroup`, `git`)
- Used by: `main.go`

**Domain Command Packages:**
- Purpose: Implement specific feature areas; each exposes a `Register(*cobra.Command)` function
- Locations:
  - `pkg/workspace/` — `init`, `context`, `group` commands
  - `pkg/repo/` — `repo add/clone/list/rename/unlink/restore` commands
  - `pkg/worktree/` — `worktree clone/add/rm/drop/list` commands
  - `pkg/branch/` — `branch create/default/checkout` commands
  - `pkg/workgroup/` — `workgroup create/checkout/list/drop/push/path/add` commands
  - `pkg/git/` — `fetch/pull/push/status/sync/exec/info/commit` commands
- Contains: cobra `RunE` handlers, local `*Report`/`*Step`/`*Flags` structs, parallel orchestration
- Depends on: `config`, `repo`, `gitutil`, `parallel`, `output`, `display`, `cmdutil`
- Used by: `pkg/cmd`

**Configuration Layer:**
- Purpose: Loads, validates, saves, and provides query helpers for `.gitw`/`.gitw.local` TOML configs
- Location: `pkg/config/`
- Contains: `WorkspaceConfig` struct, `Load`, `Save`, `SaveLocal`, `Discover`, path resolution, worktree synthesis
- Depends on: `pkg/toml`
- Used by: All domain packages

**Repository Resolution Layer:**
- Purpose: Converts config entries into concrete `Repo` structs with absolute paths; resolves names/groups/context
- Location: `pkg/repo/`
- Contains: `Repo` struct, `FromConfig`, `FromNames`, `Filter`, `ForContext`, `ForGroup`
- Depends on: `pkg/config`
- Used by: All domain command packages

**Git Execution Layer:**
- Purpose: Runs git commands against one or more repos, serially or in parallel
- Location: `pkg/git/executor.go`, `pkg/git/result.go`
- Contains: `RunParallel`, `ExecResult`, `ExecOptions`, serial/async run helpers, output prefixing
- Depends on: `pkg/parallel`, `pkg/repo`
- Used by: `pkg/git` command handlers, indirectly by domain packages

**Git Utility Layer:**
- Purpose: Low-level single-repo git operations invoked as subprocesses
- Location: `pkg/gitutil/`
- Contains: Individual functions for clone, fetch, pull, push, branch manipulation, worktree management, `.gitignore` management
- Depends on: OS (`os/exec`)
- Used by: All domain command packages and `pkg/git`

**Parallel Execution Utility:**
- Purpose: Generic, bounded fan-out execution and failure aggregation
- Location: `pkg/parallel/`
- Contains: `RunFanOut[T,R]`, `MaxWorkers`, `FormatFailureError`
- Depends on: stdlib only
- Used by: `pkg/git`, `pkg/branch`, `pkg/workgroup`, `pkg/worktree`

**Display/Output Utilities:**
- Purpose: Formatted table rendering and terminal color output
- Locations:
  - `pkg/display/` — `RenderTable`, `RenderGroupedTable`, `RenderWorkgroupTable`, color helpers
  - `pkg/output/` — `Writef` (best-effort write to `io.Writer`)
- Depends on: `github.com/fatih/color`, `pkg/repo`
- Used by: Domain command packages

**Flag Utilities:**
- Purpose: Resolve mutually exclusive boolean flag pairs (e.g. `--push`/`--no-push`) against config defaults
- Location: `pkg/cmdutil/flags.go`
- Contains: `ResolveBoolFlag`
- Used by: `pkg/branch`, `pkg/workgroup`, `pkg/git`

**TOML Utilities:**
- Purpose: Thin wrapper around `go-toml/v2` adding comment-preserving round-trip serialization
- Location: `pkg/toml/`
- Used by: `pkg/config`

**Test Utilities:**
- Purpose: Shared test helpers and a testify suite base for command integration tests
- Location: `pkg/testutil/`
- Contains: `CmdSuite`, git repo scaffolding helpers, workspace setup helpers
- Used by: All `*_test.go` files in domain packages

## Data Flow

**Standard Multi-Repo Command (e.g. `git w fetch`):**

1. `main.go` → `cmd.Execute(version)` → cobra dispatches to `RunE`
2. Handler calls `config.LoadConfig(cmd)` → discovers `.gitw` by walking up from CWD → merges `.gitw.local`
3. Handler calls `repo.Filter(cfg, cfgPath, args)` → resolves names/groups/context → returns `[]repo.Repo` with absolute paths
4. Handler calls `RunParallel(repos, gitArgs, ExecOptions{Async: len(repos)>1})` → fan-out via `parallel.RunFanOut`
5. Each goroutine calls `buildCmd(ctx, repo, args)` → `exec.CommandContext` → captures stdout/stderr
6. Results collected as `[]ExecResult` → `WriteResults` streams to `cmd.OutOrStdout()` → `ExecErrors` returns combined error

**Config-Mutating Command (e.g. `git w repo add`):**

1. cobra dispatches to `RunE`
2. `config.LoadConfig(cmd)` → loads current state
3. Business logic modifies in-memory `*WorkspaceConfig`
4. `config.Save(cfgPath, cfg)` → TOML marshal with comment preservation → atomic write (temp file + rename)

**Workgroup/Branch Multi-Step Command:**

1. cobra dispatches to `RunE`
2. Config and repos resolved as above
3. Repos classified into plain repos and worktree sets → assembled as `[]unit` structs
4. `parallel.RunFanOut(units, workers, fn)` dispatches each unit concurrently
5. Each unit fn executes sequential steps (e.g. fetch → pull → push), recording each as a `*Step` into a `*Report`
6. Reports collected → `write*Reports` emits `[repo] step` lines to stdout/stderr → `write*Summary` emits final count
7. `*ReportsError` returns combined failure error if any reports failed

**Config Discovery:**

1. Checks `GIT_W_CONFIG` env var first (override)
2. Falls back to `--config` root flag
3. Falls back to walking up from CWD looking for `.gitw` file
4. Merges `.gitw.local` (context + workgroups) after loading `.gitw`

**State Management:**
- No in-process state; all state is read from `.gitw`/`.gitw.local` on every command invocation
- `.gitw` is the shared workspace config (committed to git)
- `.gitw.local` is machine-local state (context, workgroups — gitignored by default)
- Writes use atomic rename-from-temp to avoid partial writes

## Key Abstractions

**`repo.Repo`:**
- Purpose: A resolved, ready-to-use repository with an absolute filesystem path
- Examples: `pkg/repo/repo.go`
- Pattern: Value struct (`Name string`, `AbsPath string`, `Flags []string`); created by `FromConfig`/`FromNames`/`Filter`

**`config.WorkspaceConfig`:**
- Purpose: The complete in-memory representation of a workspace (repos, groups, worktrees, workgroups, context)
- Examples: `pkg/config/config.go`
- Pattern: Loaded by `config.Load`, modified in-memory, written back by `config.Save`/`config.SaveLocal`

**`git.ExecResult` / `*Report`/`*Step` structs:**
- Purpose: Per-repo result containers used to collect outcomes of parallel operations and emit structured output
- Examples: `pkg/git/result.go`, `pkg/branch/create.go` (`branchReport`), `pkg/workgroup/common.go` (`workReport`), `pkg/git/sync.go` (`syncReport`)
- Pattern: Populated during parallel execution, then iterated for output and error aggregation

**`parallel.RunFanOut[T,R]`:**
- Purpose: Generic bounded goroutine fan-out preserving input order in results
- Examples: `pkg/parallel/parallel.go`
- Pattern: Used throughout domain packages whenever len(items) > 1

**`Register(*cobra.Command)` convention:**
- Purpose: Each domain package exposes one `Register` function that wires all its subcommands into the provided parent
- Examples: `pkg/workspace/register.go`, `pkg/repo/register.go`, `pkg/worktree/register.go`
- Pattern: Called once from `pkg/cmd/root.go`; subcommands call internal `register*` functions

**Worktree Synthesis:**
- Purpose: Worktree sets in `.gitw` are automatically expanded into synthetic `repo` entries and `group` entries at config load time
- Examples: `pkg/config/loader.go` (`synthesizeWorktreeTargets`)
- Pattern: Synthesized names follow `{setName}-{branch}` convention (e.g. `infra-dev`)

## Entry Points

**`main.go`:**
- Location: `main.go`
- Triggers: `go run .` or `bin/git-w <args>`
- Responsibilities: Receives build-time `version` ldflags variable; delegates entirely to `cmd.Execute(version)`

**`cmd.Execute(version string)`:**
- Location: `pkg/cmd/root.go`
- Triggers: Called by `main`
- Responsibilities: Builds root cobra command, wires all domain packages, sets version, calls `root.Execute()`

## Error Handling

**Strategy:** Errors propagate up through `error` return values; cobra prints them to stderr and exits non-zero. Parallel operations accumulate failures into `[]string` and format them via `parallel.FormatFailureError`.

**Patterns:**
- `RunE` handlers return `error`; cobra handles printing and exit code
- `config.LoadConfig` wraps OS/parse errors with `fmt.Errorf("...%w",...)`
- Multi-repo operations use `ExecErrors(results)` / `workReportsError(reports, op)` / `branchReportsError(reports, op)` to aggregate failures
- Partial success is supported: some repos can succeed while others fail; summary line always printed
- `output.Writef` for terminal output intentionally ignores write errors (best-effort)

## Cross-Cutting Concerns

**Logging:** None (no structured logging); operations emit `[repoName] stepName` lines directly to `cmd.OutOrStdout()` / `cmd.ErrOrStderr()`

**Validation:** Config paths validated at load time (`validateRepoPaths`, `validateWorktreePaths`); flag conflicts validated at command start (e.g. `--push`/`--no-push`)

**Authentication:** Delegated entirely to git; no auth logic in `git-w`

**Concurrency:** `parallel.RunFanOut` with `runtime.NumCPU()` default worker count; `gitutil.EnsureGitignore` uses a package-level mutex for safe concurrent `.gitignore` writes

---

*Architecture analysis: 2026-04-01*
