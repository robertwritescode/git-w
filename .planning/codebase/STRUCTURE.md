# Codebase Structure

**Analysis Date:** 2026-04-01

## Directory Layout

```
git-w/
├── main.go                  # Binary entry point; delegates to pkg/cmd
├── magefile.go              # Mage build tasks (build, test, lint, cover)
├── go.mod                   # Module declaration and dependencies
├── go.sum                   # Dependency checksums
├── .golangci.yml            # golangci-lint configuration
├── .goreleaser.yaml         # GoReleaser configuration for releases
├── bin/                     # Compiled binary output (git-w)
├── pkg/
│   ├── cmd/                 # Cobra root + command wiring (entry into feature packages)
│   ├── config/              # .gitw/.gitw.local config load/save/discovery
│   ├── repo/                # Repo struct, resolution from config, name/group/context filtering
│   ├── git/                 # Multi-repo git commands (fetch/pull/push/status/sync/exec/info/commit)
│   ├── gitutil/             # Low-level single-repo git subprocess wrappers
│   ├── workspace/           # workspace init/context/group commands
│   ├── worktree/            # worktree clone/add/rm/drop/list commands
│   ├── branch/              # branch create/default/checkout commands
│   ├── workgroup/           # workgroup create/checkout/list/drop/push/path/add commands
│   ├── parallel/            # Generic bounded fan-out (RunFanOut)
│   ├── display/             # Terminal table rendering + color formatting
│   ├── output/              # Writef helper (best-effort io.Writer writes)
│   ├── cmdutil/             # Flag utilities (ResolveBoolFlag)
│   ├── toml/                # Comment-preserving TOML marshal/unmarshal wrapper
│   └── testutil/            # Shared test helpers and CmdSuite base
├── .planning/               # GSD planning artifacts (not shipped)
├── .opencode/               # OpenCode agent configuration
├── .github/workflows/       # CI pipeline definitions
└── .vscode/                 # Editor settings
```

## Directory Purposes

**`pkg/cmd/`:**
- Purpose: Cobra root command construction and domain package wiring
- Contains: `root.go` (newRootCmd, Execute), `completion.go`
- Key files: `pkg/cmd/root.go`

**`pkg/config/`:**
- Purpose: All logic for reading/writing `.gitw` and `.gitw.local` TOML config files
- Contains: Config structs (`WorkspaceConfig`, `RepoConfig`, `WorktreeConfig`, etc.), discovery (walk-up search), loader (merge main + local), saver (atomic write with comment preservation), path resolution utilities
- Key files: `pkg/config/config.go`, `pkg/config/loader.go`, `pkg/config/discovery.go`

**`pkg/repo/`:**
- Purpose: Resolves raw config entries into `Repo` structs with absolute paths; filters by name, group, or active context
- Contains: `Repo` struct, `FromConfig`, `FromNames`, `Filter`, `ForContext`, `ForGroup`, `IsGitRepo`, repo status types, repo list/add/clone/unlink/rename/restore/status commands
- Key files: `pkg/repo/repo.go`, `pkg/repo/filter.go`, `pkg/repo/register.go`

**`pkg/git/`:**
- Purpose: Multi-repo git command execution (the core "run git across all repos" feature)
- Contains: `executor.go` (RunParallel, serial/async run), `result.go` (ExecResult, WriteResults, ExecErrors), `commands.go` (fetch/pull/push/status), `sync.go`, `exec.go`, `info.go`, `commit.go`, `runner.go`
- Key files: `pkg/git/executor.go`, `pkg/git/result.go`, `pkg/git/commands.go`

**`pkg/gitutil/`:**
- Purpose: Low-level git operations; each function wraps a single git subprocess call
- Contains: Clone, Pull, Fetch, Push, Branch operations, Worktree operations, EnsureGitignore
- Key files: `pkg/gitutil/gitutil.go`

**`pkg/workspace/`:**
- Purpose: Workspace lifecycle management (init, context switching, group management)
- Contains: `init.go`, `context.go`, `group.go`, `cmd_config.go`, `register.go`
- Key files: `pkg/workspace/register.go`, `pkg/workspace/init.go`, `pkg/workspace/context.go`

**`pkg/worktree/`:**
- Purpose: Git worktree set management (cloning bare repos, adding/removing worktrees)
- Contains: Per-operation files (`clone.go`, `add.go`, `rm.go`, `drop.go`, `list.go`), shared helpers (`common.go`), safety checks (`safety.go`)
- Key files: `pkg/worktree/register.go`, `pkg/worktree/clone.go`, `pkg/worktree/common.go`

**`pkg/branch/`:**
- Purpose: Branch management across multiple repos and worktree sets
- Contains: `create.go`, `checkout.go`, `default.go`, `register.go`
- Key files: `pkg/branch/register.go`, `pkg/branch/create.go`

**`pkg/workgroup/`:**
- Purpose: Local workgroup management (per-developer named sets of worktrees across repos)
- Contains: `create.go`, `checkout.go`, `drop.go`, `push.go`, `list.go`, `path.go`, `add.go`, `common.go`
- Key files: `pkg/workgroup/register.go`, `pkg/workgroup/common.go`, `pkg/workgroup/create.go`

**`pkg/parallel/`:**
- Purpose: Generic, bounded, order-preserving goroutine fan-out
- Contains: `RunFanOut[T,R]`, `MaxWorkers`, `FormatFailureError`
- Key files: `pkg/parallel/parallel.go`

**`pkg/display/`:**
- Purpose: Terminal table rendering with column alignment, color coding, and worktree grouping
- Contains: `table.go` (RenderTable, RenderGroupedTable, RenderWorkgroupTable), `colors.go`
- Key files: `pkg/display/table.go`, `pkg/display/colors.go`

**`pkg/output/`:**
- Purpose: Best-effort formatted writes to `io.Writer` (suppresses write errors for terminal output)
- Contains: `write.go` (Writef)
- Key files: `pkg/output/write.go`

**`pkg/cmdutil/`:**
- Purpose: Reusable flag helpers shared across domain packages
- Contains: `flags.go` (ResolveBoolFlag for `--flag`/`--no-flag` pairs)
- Key files: `pkg/cmdutil/flags.go`

**`pkg/toml/`:**
- Purpose: Wraps `go-toml/v2` with comment-preserving update capability
- Contains: Marshal, Unmarshal, UpdatePreservingComments
- Key files: `pkg/toml/`

**`pkg/testutil/`:**
- Purpose: Shared integration test infrastructure; avoids duplication across domain package tests
- Contains: `CmdSuite` (testify suite with cobra root helpers), `helpers.go` (git repo scaffolding, workspace setup), `cmd.go` (command execution helpers)
- Key files: `pkg/testutil/suite.go`, `pkg/testutil/helpers.go`

## Key File Locations

**Entry Points:**
- `main.go`: Binary entry point
- `pkg/cmd/root.go`: Root cobra command, version wiring, domain package registration

**Configuration:**
- `pkg/config/config.go`: All config struct definitions and query methods
- `pkg/config/loader.go`: Load, Save, SaveLocal, SaveLocalWorkgroup, atomic write logic
- `pkg/config/discovery.go`: `.gitw` discovery (CWD walk-up, `GIT_W_CONFIG` env override)

**Core Logic:**
- `pkg/repo/repo.go`: `Repo` struct and `FromConfig`/`FromNames`
- `pkg/repo/filter.go`: `Filter`, `ForContext`, `ForGroup` — the primary repo selection API
- `pkg/git/executor.go`: `RunParallel` — multi-repo git execution engine
- `pkg/parallel/parallel.go`: `RunFanOut` — generic bounded concurrency primitive

**Build:**
- `magefile.go`: Build tasks — `Build`, `Test`, `TestFast`, `Cover`, `Lint`, `LintFix`, `Fmt`
- `go.mod`: Module path `github.com/robertwritescode/git-w`

**Testing:**
- `pkg/testutil/suite.go`: `CmdSuite` base type used by all integration test suites
- `pkg/testutil/helpers.go`: `MakeGitRepo`, `MakeWorkspace`, `SetActiveContext`, etc.

## Naming Conventions

**Files:**
- One operation per file: `add.go`, `clone.go`, `list.go`, `drop.go` — the filename matches the subcommand
- Test files co-located: `add_test.go` lives next to `add.go`
- Shared helpers in `common.go` or named utility files (`fixtures_test.go`, `assertions_test.go`, `helpers_test.go`)
- Registration entry point always `register.go` in each domain package

**Packages:**
- Short, lowercase, single-word package names matching directory name
- `gitutil` for the low-level git subprocess wrapper (distinguished from `git` which is the command package)

**Functions:**
- `Register(*cobra.Command)` — exported, called from `pkg/cmd/root.go`
- `register<Op>(parent *cobra.Command)` — unexported, wires one subcommand
- `run<Op>(cmd *cobra.Command, args []string) error` — cobra `RunE` handler
- `resolve<X>` / `load<X>` / `collect<X>` / `write<X>` — internal orchestration helpers
- `<op>InRepo` / `<op>InWorktreeSet` — per-repo execution functions

**Types:**
- `*Report` structs (e.g. `branchReport`, `workReport`, `syncReport`) — per-repo result container
- `*Step` structs (e.g. `branchStep`, `workStep`, `syncStep`) — per-operation step within a report
- `*Flags` / `*flags` structs (e.g. `branchFlags`, `workFlags`) — resolved flag + config defaults
- `*Unit` / `*unit` structs (e.g. `branchUnit`, `syncUnit`) — work item for parallel dispatch

## Where to Add New Code

**New Top-Level Subcommand (e.g. `git w tag`):**
1. Create `pkg/tag/` directory
2. Add `register.go` with exported `Register(*cobra.Command)`
3. Add per-operation files: `create.go`, `list.go`, etc.
4. Add `register.go` internal `registerCreate(parent)` / `registerList(parent)` calls
5. Call `tag.Register(root)` in `pkg/cmd/root.go`

**New Subcommand to Existing Domain (e.g. `git w branch rename`):**
1. Create `pkg/branch/rename.go` with `registerRename(branchCmd)` and `runRename` handler
2. Call `registerRename(branchCmd)` from `pkg/branch/register.go`
3. Create `pkg/branch/rename_test.go` with test suite

**New Config Field:**
1. Add field to appropriate struct in `pkg/config/config.go`
2. Add TOML tag
3. Add accessor method on `WorkspaceConfig` if it has a nil-means-default pattern
4. Update `diskConfig` / `localDiskConfig` in `pkg/config/loader.go` as needed

**Shared Test Helpers:**
- Add to `pkg/testutil/helpers.go` (for workspace/git repo scaffolding)
- Add method on `CmdSuite` in `pkg/testutil/suite.go` (for suite-level helpers)
- Package-local helpers belong in `helpers_test.go` or `fixtures_test.go` within the domain package

**New Utility Function:**
- Single-repo git subprocess: `pkg/gitutil/gitutil.go`
- Generic concurrency: `pkg/parallel/parallel.go`
- Flag resolution: `pkg/cmdutil/flags.go`
- Terminal output: `pkg/output/write.go`
- Table display: `pkg/display/table.go` or `pkg/display/colors.go`

## Special Directories

**`bin/`:**
- Purpose: Local build output target
- Generated: Yes (`mage build` writes `bin/git-w`)
- Committed: No (gitignored)

**`.planning/`:**
- Purpose: GSD planning artifacts — phase plans, codebase analysis documents
- Generated: By GSD agent commands
- Committed: Up to developer preference; typically yes for shared planning

**`.opencode/`:**
- Purpose: OpenCode agent skill and command configuration
- Generated: No (authored)
- Committed: Yes

**`.github/workflows/`:**
- Purpose: GitHub Actions CI pipelines
- Generated: No
- Committed: Yes

---

*Structure analysis: 2026-04-01*
