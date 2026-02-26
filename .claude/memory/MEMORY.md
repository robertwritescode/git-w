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
- [testing.md](testing.md) — Testing strategy, per-package patterns, testify usage, CI flags

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
- [ ] Phase 1: Scaffold + config + basic commands
- [ ] Phase 2: Status detection + `info` display
- [ ] Phase 3: Parallel execution engine
- [ ] Phase 4: Groups + context
- [ ] Phase 5: Advanced (freeze/clone, recursive add, auto-context)

## User Preferences
- Minimal dependencies
- No TUI framework
- Plain formatted output for `info`
- Single compiled binary (no runtime deps)
