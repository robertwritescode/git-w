# git-workspace Project Memory

## Project Identity
- **Name**: `git-workspace`
- **Binary name**: `git-workspace` (enables `git workspace <cmd>` via git plugin system)
- **Language**: Go 1.26
- **Inspired by**: [gita](https://github.com/nosarthur/gita) (Python multi-repo manager)
- **Purpose**: Manage multiple git repos defined in a local `.gitworkspace` TOML config

## Key Files & Docs
- [architecture.md](architecture.md) — Full architecture design, command inventory, internal design
- [decisions.md](decisions.md) — Design decisions and their rationale
- [release.md](release.md) — Build, versioning, CI/CD (GitHub Actions), GoReleaser, Homebrew tap
- [implementation.md](implementation.md) — Phased implementation plan and progress tracking
- [testing.md](testing.md) — Testing strategy, per-package patterns, testify suite pattern, CI flags
- [coding-standards.md](coding-standards.md) — Code quality: DRY, complexity, no unnecessary comments, self-review checklist

## Config File
- Name: `.gitworkspace` (TOML format)
- Location: workspace root; discovered by walking up from CWD (like `.git`)
- Env var override: `GIT_WORKSPACE_CONFIG`
- Defines repos with paths **relative to the config file's location**
- Repos, groups, and per-repo git flags live here (committed, shared)
- Active context lives in `.gitworkspace.local` (gitignored, per-developer; created by `init`)

## Tech Stack
- CLI: `github.com/spf13/cobra`
- Config parsing: `github.com/pelletier/go-toml/v2`
- Colors: `github.com/fatih/color`
- Concurrency: `golang.org/x/sync` (errgroup) + goroutine semaphore
- Testing: `github.com/stretchr/testify` (`assert` + `require`) — required in all test files
- Release: GoReleaser + GitHub Actions; primary distribution via Homebrew custom tap

## Architecture Status
- [x] Architecture designed (see architecture.md)
- [ ] User approved architecture
- [x] Phase 1: Scaffold + config + basic commands — **COMPLETE** (`go test -race ./...` passes)
- [x] Phase 2: Status detection + `info`/`ll` display — **COMPLETE** (`go test -race -count=1 ./...` passes)
- [ ] Phase 3: Parallel execution engine
- [ ] Phase 4: Groups + context
- [ ] Phase 5: Advanced (freeze/clone, recursive add, auto-context)

## Coding Standards (apply proactively)

- **testify/suite is required** in every test file — embed `suite.Suite`, use `s.Require()` / `s.Assert()`, register with `suite.Run(t, new(XxxSuite))`. Bare `func TestXxx(t *testing.T)` is only used as the `suite.Run` entry point.
- **Table-driven tests are required** (not optional) for any test with multiple input/output cases — use `[]struct{ name, ... }` + `s.Run(tc.name, func() { ... })`.
- **Extract to private functions** — public functions should read as high-level steps; complex logic goes in private helpers. Functions over ~20 lines need extraction.
- **No unnecessary comments** — do not restate what code does; only comment the *why* when non-obvious. Remove AI-generated boilerplate comments.
- **DRY** — if two functions share 2–3+ lines of identical logic, extract a shared helper.

See [coding-standards.md](coding-standards.md) for detail and examples.

## User Preferences
- Minimal dependencies
- No TUI framework
- Plain formatted output for `info`
- Single compiled binary (no runtime deps)
