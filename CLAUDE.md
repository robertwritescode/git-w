<!-- GSD:project-start source:PROJECT.md -->
## Project

**git-w v2**

A major version upgrade of git-w, the Go CLI tool that manages multiple git repos via `git w <cmd>`. v2 replaces the workgroup model with a two-level workspace/workstream hierarchy, adds multi-destination remote management with push protection, supports flexible infra repo patterns (branch-per-env and folder-per-env), and introduces an agent interop layer so AI coding tools can operate within git-w-managed environments. Ships as a single compiled binary with a `git w migrate` upgrade path from v1.

**Core Value:** Multi-repo orchestration that keeps repos organized, synced, and safe from accidental pushes while giving AI agents full visibility into the workspace structure.

### Constraints

- **Tech stack**: Go 1.26, cobra, go-toml/v2. No new runtime dependencies without justification. Single compiled binary.
- **No TUI**: Plain formatted output via `text/tabwriter`. No bubbletea/lipgloss.
- **Breaking changes**: v2.0 is a major version. Workgroup retirement, command surface reduction, directory layout migration all require `git w migrate` path.
- **Compatibility**: v1 configs must be detected at load time with actionable migration instructions.
- **Output**: `output.Writef` for stdout/stderr. No `fmt.Fprintf` directly.
- **Testing**: testify suites for shared setup/teardown, table-driven tests for combinatorial cases, `mage test` with race detector before marking work complete.
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.26.0 - All application code in `pkg/` and entry point `main.go`
- None — pure Go project
## Runtime
- Go runtime 1.26.0 (specified in `go.mod`)
- Go modules (`go mod`)
- Lockfile: `go.sum` present and committed
## Frameworks
- `github.com/spf13/cobra v1.10.2` — CLI command tree, flag parsing, subcommand dispatch (`pkg/cmd/root.go`)
- `github.com/spf13/pflag v1.0.9` — Enhanced POSIX flag handling (cobra dependency)
- `github.com/stretchr/testify v1.11.1` — Assertions and test suite support (used throughout `pkg/**/*_test.go`)
- `github.com/magefile/mage v1.15.0` — Build task runner (`magefile.go`); replaces Makefile
- `golangci-lint v2.10.1` — Linter + formatter; pinned in `.github/workflows/ci.yml`
- `goreleaser v2` — Cross-platform release builds and Homebrew cask publishing (`.goreleaser.yaml`)
- `release-please` — Automated changelog and release PR management (`.release-please-config.json`)
## Key Dependencies
- `github.com/pelletier/go-toml/v2 v2.2.4` — TOML parsing and marshaling for `.gitw` config files (`pkg/toml/`, `pkg/config/`)
- `github.com/fatih/color v1.18.0` — Terminal color output for status display (`pkg/display/colors.go`)
- `github.com/spf13/cobra v1.10.2` — Entire CLI surface, subcommand registration, help generation
- `github.com/mattn/go-colorable v0.1.13` — Windows-compatible ANSI color output (indirect, via fatih/color)
- `github.com/mattn/go-isatty v0.0.20` — TTY detection for color auto-disable (indirect, via fatih/color)
- `golang.org/x/sys v0.25.0` — OS-level syscalls (indirect)
- `gopkg.in/yaml.v3 v3.0.1` — YAML support (indirect, via testify)
## Configuration
- User-facing config: `.gitw` TOML file, discovered by walking up from CWD or via `GIT_W_CONFIG` env var
- Local overrides: `.gitw.local` TOML file (merged after main config; not committed to VCS)
- Config loading entrypoint: `pkg/config/loader.go`
- Discovery logic: `pkg/config/discovery.go`
- Build config: `magefile.go` (mage tasks: `All`, `Build`, `Test`, `TestFast`, `Lint`, `LintFix`, `Fmt`, `Cover`)
- Release config: `.goreleaser.yaml`
- Lint config: `.golangci.yml` (minimal; uses golangci-lint v2 defaults)
- Release-please config: `.release-please-config.json`
- Version set at build time via `-ldflags "-X main.version=<tag>"` using `git describe --tags`
- Version exposed via `root.Version` in cobra command (`pkg/cmd/root.go`)
## Platform Requirements
- Go 1.26.0+
- `golangci-lint` (install separately; used by `mage Lint`)
- `git` binary on `$PATH` (the CLI wraps git commands)
- Distributed as a static binary: `bin/git-w`
- Cross-compiled targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Distributed via GitHub Releases (tar.gz archives) and Homebrew cask (`robertwritescode/homebrew-tap`)
- Must be on `$PATH` as `git-w` to be invoked as `git w <cmd>` via git plugin system
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming Patterns
- One file per operation: `add.go`, `drop.go`, `clone.go`, `list.go`
- Pairing: every source file has a co-located `_test.go`: `add.go` / `add_test.go`
- Registration: each package exposes a single `register.go` that wires sub-commands into cobra
- Common/shared helpers: `common.go` for intra-package shared utilities
- Exported: `PascalCase` — e.g. `FromConfig`, `GetStatus`, `RunFanOut`, `EnsureGitignore`
- Unexported: `camelCase` — e.g. `runAdd`, `prepareAddOperation`, `parseBranchLine`
- Cobra RunE handlers: always named `run<Command>` — e.g. `runAdd`, `runCreate`, `runDrop`
- Sub-command registration: `register<Command>` — e.g. `registerAdd`, `registerCreate`
- Prepare/execute/finalize pattern for multi-step operations: `prepareAddOperation`, `executeAddOperation`, `finalizeAddOperation`
- `camelCase` everywhere; short names for local scope
- Error variables: bare `err` in single-error context; descriptive (e.g. `loadErr`, `existsErr`) when multiple errors in scope
- Structs: `PascalCase` nouns — e.g. `WorkspaceConfig`, `RepoStatus`, `branchReport`
- Unexported operation structs: lowercase — e.g. `addOperation`, `branchUnit`, `branchFlags`
- Enums via `iota` with type alias: `RemoteState int`, constants `Unknown`, `InSync`, `LocalAhead` etc.
- Config structs tagged with `toml:"..."` field tags
- Suite types: `<Feature>Suite` — e.g. `WorktreeAddSuite`, `BranchCreateSuite`, `StatusSuite`
- Table-driven test case structs: lowercase with `name string` as first field — e.g. `branchAccessorCase`, `filterCase`, `flagConflictCase`
## Code Style
- `golangci-lint fmt` (gofmt-compatible), enforced via `mage Lint`
- Enforced in CI via `golangci-lint fmt --diff ./...`
- `golangci-lint run` with config in `.golangci.yml`
- Max issues per linter: unlimited (`max-issues-per-linter: 0`, `max-same-issues: 0`)
## Import Organization
- Used only to resolve name collisions: `gitpkg "github.com/robertwritescode/git-w/pkg/git"` in `pkg/cmd/root.go`
- Avoid aliases unless necessary
## Error Handling
- Return `error` as the last value; never panic for expected errors
- Wrap errors with context using `fmt.Errorf("doing X: %w", err)` — error wrapping is pervasive
- Propagate config/IO errors immediately; no silent swallowing unless intentional and commented
- Intentional error discard: assign to `_` with explanatory comment when used in `defer` or cleanup (e.g. `_ = os.Chdir(orig)`)
- Commands (`RunE` handlers): return errors directly for cobra to handle; use `output.Writef(cmd.ErrOrStderr(), ...)` for non-fatal warnings
- `errors.Is` for sentinel comparison (e.g. `errors.Is(err, os.ErrNotExist)`)
- `errors.As` for type-specific extraction (e.g. extracting `*exec.ExitError`)
- Lowercase, no trailing period: `"worktree set %q not found"`, `"path must be relative"`
- Include quoted identifiers with `%q` for user-facing names
- Git subprocess errors include the raw output: `fmt.Errorf("git clone: %w\n%s", err, out)`
## Logging / Output
- Stdout via `cmd.OutOrStdout()` for success/progress: `output.Writef(cmd.OutOrStdout(), "[%s] %s\n", repoName, step)`
- Stderr via `cmd.ErrOrStderr()` for warnings: `output.Writef(cmd.ErrOrStderr(), "warning: %v\n", err)`
- Errors silently discarded in `output.Writef` (best-effort terminal output)
- No structured logging; no third-party logging library
## Comments
- Every exported function/type/constant gets a doc comment (`// FunctionName does X`)
- Non-obvious unexported helpers also commented
- File-level comments rarely used; package-level `// package foo` doc comment not used beyond the package declaration
- Complete sentences starting with the identifier name: `// Output runs a git command in repoPath and returns its stdout.`
- Multi-line comments use `//` per line, not `/* */`
- Inline comments for non-obvious logic choices (e.g. `// Clear cache to ensure fresh run`)
## Function Design
- `context.Context` as first parameter for any function doing I/O or git operations
- `testing.TB` (not `*testing.T`) for helper functions that work with both `T` and `B`
- `*cobra.Command` passed through for access to flags and output writers
- `(result, error)` standard pair
- Named returns avoided; the `closeWithError` pattern uses a named `err` pointer for deferred error capture
## Module Design
- Each feature package exports only a `Register(*cobra.Command)` entry point and necessary types
- Internal operations are unexported; tests use the `_test` package suffix (`package repo_test`) for black-box testing, or the internal package name (`package repo`) for white-box tests of unexported helpers
- Not used; each package exposes a focused public API
- `register.go` is the conventional entry point per feature package
## Context Pattern
- `context.Background()` used in non-command code paths (e.g. `safetyViolations`, direct test calls)
- `cmd.Context()` used inside cobra `RunE` handlers
- Context passed all the way through to git subprocess calls via `exec.CommandContext`
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- Single binary CLI tool (`git-w`) designed to be invoked as `git w <cmd>` via git's plugin system
- Each feature domain is a self-contained package that registers its own subcommands onto a shared cobra root
- Commands follow a consistent pattern: load config → resolve repos → execute in parallel → collect reports → write output
- Configuration is file-driven (`.gitw` TOML file) discovered by walking up the filesystem tree from CWD
- Parallel execution is a first-class concern; most multi-repo operations use fan-out goroutines
## Layers
- Purpose: Wires version and kicks off cobra execution
- Location: `main.go`
- Contains: Single `main()` delegating to `cmd.Execute(version)`
- Depends on: `pkg/cmd`
- Used by: OS/shell
- Purpose: Builds the cobra command tree by wiring domain packages
- Location: `pkg/cmd/root.go`
- Contains: `newRootCmd`, `Execute`, top-level flag (`--config`), completion command
- Depends on: All domain packages (`workspace`, `repo`, `worktree`, `branch`, `workgroup`, `git`)
- Used by: `main.go`
- Purpose: Implement specific feature areas; each exposes a `Register(*cobra.Command)` function
- Locations:
- Contains: cobra `RunE` handlers, local `*Report`/`*Step`/`*Flags` structs, parallel orchestration
- Depends on: `config`, `repo`, `gitutil`, `parallel`, `output`, `display`, `cmdutil`
- Used by: `pkg/cmd`
- Purpose: Loads, validates, saves, and provides query helpers for `.gitw`/`.gitw.local` TOML configs
- Location: `pkg/config/`
- Contains: `WorkspaceConfig` struct, `Load`, `Save`, `SaveLocal`, `Discover`, path resolution, worktree synthesis
- Depends on: `pkg/toml`
- Used by: All domain packages
- Purpose: Converts config entries into concrete `Repo` structs with absolute paths; resolves names/groups/context
- Location: `pkg/repo/`
- Contains: `Repo` struct, `FromConfig`, `FromNames`, `Filter`, `ForContext`, `ForGroup`
- Depends on: `pkg/config`
- Used by: All domain command packages
- Purpose: Runs git commands against one or more repos, serially or in parallel
- Location: `pkg/git/executor.go`, `pkg/git/result.go`
- Contains: `RunParallel`, `ExecResult`, `ExecOptions`, serial/async run helpers, output prefixing
- Depends on: `pkg/parallel`, `pkg/repo`
- Used by: `pkg/git` command handlers, indirectly by domain packages
- Purpose: Low-level single-repo git operations invoked as subprocesses
- Location: `pkg/gitutil/`
- Contains: Individual functions for clone, fetch, pull, push, branch manipulation, worktree management, `.gitignore` management
- Depends on: OS (`os/exec`)
- Used by: All domain command packages and `pkg/git`
- Purpose: Generic, bounded fan-out execution and failure aggregation
- Location: `pkg/parallel/`
- Contains: `RunFanOut[T,R]`, `MaxWorkers`, `FormatFailureError`
- Depends on: stdlib only
- Used by: `pkg/git`, `pkg/branch`, `pkg/workgroup`, `pkg/worktree`
- Purpose: Formatted table rendering and terminal color output
- Locations:
- Depends on: `github.com/fatih/color`, `pkg/repo`
- Used by: Domain command packages
- Purpose: Resolve mutually exclusive boolean flag pairs (e.g. `--push`/`--no-push`) against config defaults
- Location: `pkg/cmdutil/flags.go`
- Contains: `ResolveBoolFlag`
- Used by: `pkg/branch`, `pkg/workgroup`, `pkg/git`
- Purpose: Thin wrapper around `go-toml/v2` adding comment-preserving round-trip serialization
- Location: `pkg/toml/`
- Used by: `pkg/config`
- Purpose: Shared test helpers and a testify suite base for command integration tests
- Location: `pkg/testutil/`
- Contains: `CmdSuite`, git repo scaffolding helpers, workspace setup helpers
- Used by: All `*_test.go` files in domain packages
## Data Flow
- No in-process state; all state is read from `.gitw`/`.gitw.local` on every command invocation
- `.gitw` is the shared workspace config (committed to git)
- `.gitw.local` is machine-local state (context, workgroups — gitignored by default)
- Writes use atomic rename-from-temp to avoid partial writes
## Key Abstractions
- Purpose: A resolved, ready-to-use repository with an absolute filesystem path
- Examples: `pkg/repo/repo.go`
- Pattern: Value struct (`Name string`, `AbsPath string`, `Flags []string`); created by `FromConfig`/`FromNames`/`Filter`
- Purpose: The complete in-memory representation of a workspace (repos, groups, worktrees, workgroups, context)
- Examples: `pkg/config/config.go`
- Pattern: Loaded by `config.Load`, modified in-memory, written back by `config.Save`/`config.SaveLocal`
- Purpose: Per-repo result containers used to collect outcomes of parallel operations and emit structured output
- Examples: `pkg/git/result.go`, `pkg/branch/create.go` (`branchReport`), `pkg/workgroup/common.go` (`workReport`), `pkg/git/sync.go` (`syncReport`)
- Pattern: Populated during parallel execution, then iterated for output and error aggregation
- Purpose: Generic bounded goroutine fan-out preserving input order in results
- Examples: `pkg/parallel/parallel.go`
- Pattern: Used throughout domain packages whenever len(items) > 1
- Purpose: Each domain package exposes one `Register` function that wires all its subcommands into the provided parent
- Examples: `pkg/workspace/register.go`, `pkg/repo/register.go`, `pkg/worktree/register.go`
- Pattern: Called once from `pkg/cmd/root.go`; subcommands call internal `register*` functions
- Purpose: Worktree sets in `.gitw` are automatically expanded into synthetic `repo` entries and `group` entries at config load time
- Examples: `pkg/config/loader.go` (`synthesizeWorktreeTargets`)
- Pattern: Synthesized names follow `{setName}-{branch}` convention (e.g. `infra-dev`)
## Entry Points
- Location: `main.go`
- Triggers: `go run .` or `bin/git-w <args>`
- Responsibilities: Receives build-time `version` ldflags variable; delegates entirely to `cmd.Execute(version)`
- Location: `pkg/cmd/root.go`
- Triggers: Called by `main`
- Responsibilities: Builds root cobra command, wires all domain packages, sets version, calls `root.Execute()`
## Error Handling
- `RunE` handlers return `error`; cobra handles printing and exit code
- `config.LoadConfig` wraps OS/parse errors with `fmt.Errorf("...%w",...)`
- Multi-repo operations use `ExecErrors(results)` / `workReportsError(reports, op)` / `branchReportsError(reports, op)` to aggregate failures
- Partial success is supported: some repos can succeed while others fail; summary line always printed
- `output.Writef` for terminal output intentionally ignores write errors (best-effort)
## Cross-Cutting Concerns
<!-- GSD:architecture-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd:quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd:debug` for investigation and bug fixing
- `/gsd:execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd:profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
