# git-w Project Memory

## Project Identity
- **Name**: `git-w`
- **Binary name**: `git-w` (enables `git w <cmd>` via git plugin system)
- **Language**: Go 1.26
- **Inspired by**: [gita](https://github.com/nosarthur/gita) (Python multi-repo manager)
- **Purpose**: Manage multiple git repos defined in a local `.gitw` TOML config
- **Status**: v0.1.0 MVP — all commands implemented, all tests passing

## Documentation Index
- [architecture.md](architecture.md) — Directory structure, type definitions, command inventory, config schema
- [decisions.md](decisions.md) — Design decisions and their rationale
- [release.md](release.md) — Build (Mage), versioning, CI/CD (GitHub Actions), GoReleaser, Homebrew tap
- [testing.md](testing.md) — Testing strategy, per-package patterns, testify suite pattern, pitfalls
- [coding-standards.md](coding-standards.md) — Code quality standards, self-review checklist, domain package pattern

## Config Files
- **`.gitw`** — committed, shared workspace definition (repos, groups, settings); TOML format
- **`.gitw.local`** — gitignored, per-developer state (`[context]` section only)
- Location: workspace root; discovered by walking up from CWD (like `.git`)
- Env var override: `GIT_W_CONFIG`; CLI override: `--config` flag
- Loader merges both files; `.local` values take precedence
- Repos stored with paths **relative to the config file's location**

## Tech Stack
- CLI: `github.com/spf13/cobra`
- Config parsing: `github.com/pelletier/go-toml/v2`
- Colors: `github.com/fatih/color`
- Concurrency: native goroutines + channels/WaitGroup (in `pkg/parallel`)
- Testing: `github.com/stretchr/testify` (`assert` + `require`) — required in all test files
- Build: Mage (`magefile.go`) — Go-based build tool
- Release: GoReleaser + GitHub Actions; primary distribution via Homebrew custom tap

## pkg/ Domain Layout
```
pkg/cmd/             — root cobra cmd, Execute(), completion (wires 4 domain Register funcs)
pkg/workspace/       — config types, loader, discovery, init/context/group commands
pkg/repo/            — repo types, filter, status, add/clone/unlink/rename/restore/list commands
pkg/git/             — executor, result, fetch/pull/push/status/exec/info commands
pkg/worktree/        — worktree set commands: clone/add/rm/drop/list; safety checks
pkg/gitutil/         — low-level git subprocess wrappers (Clone, CloneBare, AddWorktree, RemoveWorktree, FetchBare, EnsureGitignore)
pkg/parallel/        — generic concurrency primitives (RunFanOut, MaxWorkers, FormatFailureError)
pkg/display/         — terminal output formatting (RenderTable, ANSI colors)
pkg/output/          — standardized command output helpers
pkg/testutil/        — shared test infrastructure (CmdSuite, MakeGitRepo, MakeBareGitRepo, AddWorktreeToRepo, etc.)
```

Dependency graph (cycle-free):
```
workspace  → gitutil
repo       → workspace, gitutil
display    → repo
git        → repo, workspace, display, parallel
worktree   → workspace, repo, gitutil, parallel
output     → (none)
gitutil    → (none)
parallel   → (none)
testutil   → (none)
```

### Key architectural patterns
- Each domain package exports a single `Register(root *cobra.Command)` that calls private `register<Name>` functions
- **Repo lifecycle commands** live under a `repo` subcommand (alias `r`): `git w repo add/clone/unlink/rename/list`; `restore` is directly on root
- Config loaded via `workspace.LoadConfig(cmd)` (reads `--config` flag, calls `workspace.LoadCWD`)
- Repo filtering cascade: `repo.Filter` → `repo.ForContext` → `repo.ForGroup` in `pkg/repo/filter.go`
- `display.TableEntry.RemoteState` uses `repo.RemoteState` (single canonical enum, no duplication)
- `gitutil.Clone` / `gitutil.CloneContext` / `gitutil.EnsureGitignore` in `pkg/gitutil`; `EnsureGitignore` is mutex-protected for concurrent calls
- `pkg/git/runner.go` provides shared `runGitCmd` helper used by fetch/pull/push/status
- `pkg/parallel` provides `RunFanOut` (generic fan-out over goroutines) used by `pkg/git/executor.go`

## How to Add a New Command

Choose the appropriate domain package (`pkg/workspace`, `pkg/repo`, or `pkg/git`).

1. **Add file**: `pkg/<domain>/<name>.go` with `package <domain>`
2. **Define command and private register function**:
   ```go
   package workspace  // or repo, or git

   import (
       "github.com/spf13/cobra"
   )

   func registerName(root *cobra.Command) {
       root.AddCommand(&cobra.Command{
           Use:   "name",
           Short: "Description",
           RunE:  runName,
       })
   }

   func runName(cmd *cobra.Command, args []string) error {
       cfg, cfgPath, err := LoadConfig(cmd)  // same-package call in workspace
       // or: cfg, cfgPath, err := workspace.LoadConfig(cmd)  // in repo or git
       if err != nil {
           return err
       }
       // ... command logic
   }
   ```
3. **Wire in `pkg/<domain>/register.go`**: Add `registerName(root)` to the `Register` function
4. **Create test file**: `pkg/<domain>/<name>_test.go` — embed `testutil.CmdSuite`, use `s.SetRoot(<domain>.Register)` and `s.ExecuteCmd`
5. **Follow coding standards**: testify/suite, table-driven tests, ≤20-line functions, no restating comments

## Coding Standards (quick reference)

- **testify/suite required** in every test file — `suite.Suite` embed, `s.Require()` / `s.Assert()`
- **Table-driven tests required** for any test with multiple cases
- **Extract to private functions** — public functions ≤ ~20 lines; complex logic in helpers
- **No unnecessary comments** — never restate what code does; only comment the *why*
- **DRY** — extract shared helpers when 2+ functions share identical logic
- **Guard clauses** — early returns over nested conditionals

See [coding-standards.md](coding-standards.md) for detail and examples.

## User Preferences
- Minimal dependencies
- No TUI framework (no bubbletea/lipgloss)
- Plain formatted output for `info` (text/tabwriter)
- Single compiled binary (no runtime deps)
