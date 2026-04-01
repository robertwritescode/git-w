# Coding Conventions

**Analysis Date:** 2026-04-01

## Naming Patterns

**Files:**
- One file per operation: `add.go`, `drop.go`, `clone.go`, `list.go`
- Pairing: every source file has a co-located `_test.go`: `add.go` / `add_test.go`
- Registration: each package exposes a single `register.go` that wires sub-commands into cobra
- Common/shared helpers: `common.go` for intra-package shared utilities

**Functions:**
- Exported: `PascalCase` — e.g. `FromConfig`, `GetStatus`, `RunFanOut`, `EnsureGitignore`
- Unexported: `camelCase` — e.g. `runAdd`, `prepareAddOperation`, `parseBranchLine`
- Cobra RunE handlers: always named `run<Command>` — e.g. `runAdd`, `runCreate`, `runDrop`
- Sub-command registration: `register<Command>` — e.g. `registerAdd`, `registerCreate`
- Prepare/execute/finalize pattern for multi-step operations: `prepareAddOperation`, `executeAddOperation`, `finalizeAddOperation`

**Variables:**
- `camelCase` everywhere; short names for local scope
- Error variables: bare `err` in single-error context; descriptive (e.g. `loadErr`, `existsErr`) when multiple errors in scope

**Types:**
- Structs: `PascalCase` nouns — e.g. `WorkspaceConfig`, `RepoStatus`, `branchReport`
- Unexported operation structs: lowercase — e.g. `addOperation`, `branchUnit`, `branchFlags`
- Enums via `iota` with type alias: `RemoteState int`, constants `Unknown`, `InSync`, `LocalAhead` etc.
- Config structs tagged with `toml:"..."` field tags

**Test Types:**
- Suite types: `<Feature>Suite` — e.g. `WorktreeAddSuite`, `BranchCreateSuite`, `StatusSuite`
- Table-driven test case structs: lowercase with `name string` as first field — e.g. `branchAccessorCase`, `filterCase`, `flagConflictCase`

## Code Style

**Formatting:**
- `golangci-lint fmt` (gofmt-compatible), enforced via `mage Lint`
- Enforced in CI via `golangci-lint fmt --diff ./...`

**Linting:**
- `golangci-lint run` with config in `.golangci.yml`
- Max issues per linter: unlimited (`max-issues-per-linter: 0`, `max-same-issues: 0`)

## Import Organization

**Order (goimports style):**
1. Standard library packages
2. Third-party packages (cobra, testify, etc.)
3. Internal packages (`github.com/robertwritescode/git-w/pkg/...`)

**Path Aliases:**
- Used only to resolve name collisions: `gitpkg "github.com/robertwritescode/git-w/pkg/git"` in `pkg/cmd/root.go`
- Avoid aliases unless necessary

## Error Handling

**Patterns:**
- Return `error` as the last value; never panic for expected errors
- Wrap errors with context using `fmt.Errorf("doing X: %w", err)` — error wrapping is pervasive
- Propagate config/IO errors immediately; no silent swallowing unless intentional and commented
- Intentional error discard: assign to `_` with explanatory comment when used in `defer` or cleanup (e.g. `_ = os.Chdir(orig)`)
- Commands (`RunE` handlers): return errors directly for cobra to handle; use `output.Writef(cmd.ErrOrStderr(), ...)` for non-fatal warnings
- `errors.Is` for sentinel comparison (e.g. `errors.Is(err, os.ErrNotExist)`)
- `errors.As` for type-specific extraction (e.g. extracting `*exec.ExitError`)

**Error Message Style:**
- Lowercase, no trailing period: `"worktree set %q not found"`, `"path must be relative"`
- Include quoted identifiers with `%q` for user-facing names
- Git subprocess errors include the raw output: `fmt.Errorf("git clone: %w\n%s", err, out)`

## Logging / Output

**Framework:** `pkg/output` package with `output.Writef` — thin wrapper over `fmt.Fprintf`

**Pattern:**
- Stdout via `cmd.OutOrStdout()` for success/progress: `output.Writef(cmd.OutOrStdout(), "[%s] %s\n", repoName, step)`
- Stderr via `cmd.ErrOrStderr()` for warnings: `output.Writef(cmd.ErrOrStderr(), "warning: %v\n", err)`
- Errors silently discarded in `output.Writef` (best-effort terminal output)
- No structured logging; no third-party logging library

## Comments

**When to Comment:**
- Every exported function/type/constant gets a doc comment (`// FunctionName does X`)
- Non-obvious unexported helpers also commented
- File-level comments rarely used; package-level `// package foo` doc comment not used beyond the package declaration

**Style:**
- Complete sentences starting with the identifier name: `// Output runs a git command in repoPath and returns its stdout.`
- Multi-line comments use `//` per line, not `/* */`
- Inline comments for non-obvious logic choices (e.g. `// Clear cache to ensure fresh run`)

## Function Design

**Size:** Functions kept small and focused — large operations split into `prepare/execute/finalize` or named step functions

**Parameters:**
- `context.Context` as first parameter for any function doing I/O or git operations
- `testing.TB` (not `*testing.T`) for helper functions that work with both `T` and `B`
- `*cobra.Command` passed through for access to flags and output writers

**Return Values:**
- `(result, error)` standard pair
- Named returns avoided; the `closeWithError` pattern uses a named `err` pointer for deferred error capture

## Module Design

**Exports:**
- Each feature package exports only a `Register(*cobra.Command)` entry point and necessary types
- Internal operations are unexported; tests use the `_test` package suffix (`package repo_test`) for black-box testing, or the internal package name (`package repo`) for white-box tests of unexported helpers

**Barrel Files:**
- Not used; each package exposes a focused public API
- `register.go` is the conventional entry point per feature package

## Context Pattern

- `context.Background()` used in non-command code paths (e.g. `safetyViolations`, direct test calls)
- `cmd.Context()` used inside cobra `RunE` handlers
- Context passed all the way through to git subprocess calls via `exec.CommandContext`

---

*Convention analysis: 2026-04-01*
