# git-w Project Memory

## Project Identity
- **Name**: `git-w`
- **Binary name**: `git-w` (enables `git w <cmd>` via git plugin system)
- **Language**: Go 1.26
- **Inspired by**: [gita](https://github.com/nosarthur/gita) (Python multi-repo manager)
- **Purpose**: Manage multiple git repos defined in a local `.gitw` TOML config
- **Status**: Active development — workgroup feature in progress on branch `23-feature-workgroup`

## Documentation Index
- [architecture.md](architecture.md) — Directory structure, type definitions, command inventory, config schema
- [decisions.md](decisions.md) — Design decisions and their rationale
- [release.md](release.md) — Build (Mage), versioning, CI/CD (GitHub Actions), GoReleaser, Homebrew tap
- [testing.md](testing.md) — Testing strategy, per-package patterns, testify suite pattern, pitfalls
- [coding-standards.md](coding-standards.md) — Code quality standards, self-review checklist, domain package pattern

## Config Files
- **`.gitw`** — committed, shared workspace definition (repos, groups, settings); TOML format
- **`.gitw.local`** — gitignored, per-developer state (`[context]` section + `[workgroup.*]` entries)
- Location: workspace root; discovered by walking up from CWD (like `.git`)
- Env var override: `GIT_W_CONFIG`; CLI override: `--config` flag
- Loader merges both files; `.local` values take precedence
- Repos stored with paths **relative to the config file's location**

## Tech Stack
- CLI: `github.com/spf13/cobra`
- Config parsing: `github.com/pelletier/go-toml/v2` (wrapped in `pkg/toml` for comment preservation)
- Colors: `github.com/fatih/color`
- Concurrency: native goroutines + channels/WaitGroup (in `pkg/parallel`)
- Testing: `github.com/stretchr/testify` (`assert` + `require`) — required in all test files
- Build: Mage (`magefile.go`) — Go-based build tool
- Release: GoReleaser + GitHub Actions; primary distribution via Homebrew custom tap

## pkg/ Domain Layout
```
pkg/cmd/             — root cobra cmd, Execute(), completion (wires domain Register funcs)
pkg/config/          — ALL config types (WorkspaceConfig etc.), loader, discovery
pkg/toml/            — TOML parsing wrapper with comment preservation (wraps go-toml/v2)
pkg/cmdutil/         — shared CLI flag helpers (ResolveBoolFlag for on/off flag pairs)
pkg/workspace/       — init/context/group commands; cmd_config helpers
pkg/repo/            — repo types, filter, status, add/clone/unlink/rename/restore/list commands; SafetyViolations
pkg/git/             — executor, result, fetch/pull/push/status/exec/info/sync commands
pkg/branch/          — branch create (and future branch subcommands); register.go + create.go
pkg/worktree/        — worktree set commands: clone/add/rm/drop/list; safety checks
pkg/workgroup/       — workgroup commands: create/checkout/add/drop/push/list/path; common helpers
pkg/gitutil/         — low-level git subprocess wrappers; ALL functions take context.Context
pkg/parallel/        — generic concurrency primitives (RunFanOut, MaxWorkers, FormatFailureError)
pkg/display/         — terminal output formatting (RenderTable, ANSI colors)
pkg/output/          — standardized command output helpers
pkg/testutil/        — shared test infrastructure (CmdSuite, MakeGitRepo, MakeBareGitRepo, AddWorktreeToRepo, etc.)
```

Dependency graph (cycle-free):
```
config     → (none)
cmdutil    → (none)
workspace  → config, gitutil
repo       → config, gitutil
display    → repo
git        → repo, config, display, parallel
branch     → config, repo, gitutil, parallel, output, cmdutil
worktree   → config, repo, gitutil, parallel
workgroup  → config, repo, gitutil, parallel, output, cmdutil
output     → (none)
gitutil    → (none)
parallel   → (none)
testutil   → (none)
```

### Key architectural patterns
- Each domain package exports a single `Register(root *cobra.Command)` that calls private `register<Name>` functions
- **Repo lifecycle commands** live under a `repo` subcommand (alias `r`): `git w repo add/clone/unlink/rename/list`; `restore` is directly on root
- Config loaded via `config.LoadConfig(cmd)` (reads `--config` flag, calls `config.LoadCWD`)
- Repo filtering cascade: `repo.Filter` → `repo.ForContext` → `repo.ForGroup` in `pkg/repo/filter.go`
- `display.TableEntry.RemoteState` uses `repo.RemoteState` (single canonical enum, no duplication)
- `gitutil.Clone` / `gitutil.CloneContext` / `gitutil.EnsureGitignore` in `pkg/gitutil`; `EnsureGitignore` is mutex-protected for concurrent calls
- `pkg/git/runner.go` provides shared `runGitCmd` helper used by fetch/pull/push/status
- `pkg/parallel` provides `RunFanOut` (generic fan-out over goroutines) used by `pkg/git/executor.go`
- `cmdutil.ResolveBoolFlag` handles `--flag` / `--no-flag` pairs with config default fallback (used by branch and workgroup)
- `repo.SafetyViolations` is the canonical safety check (uncommitted + unpushed); used by both `pkg/worktree` and `pkg/workgroup`

## How to Add a New Command

Choose the appropriate domain package (`pkg/workspace`, `pkg/repo`, `pkg/git`, `pkg/branch`, or `pkg/workgroup`).

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
       cfg, cfgPath, err := config.LoadConfig(cmd)  // pkg/config in all packages
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
