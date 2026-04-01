# Technology Stack

**Analysis Date:** 2026-04-01

## Languages

**Primary:**
- Go 1.26.0 - All application code in `pkg/` and entry point `main.go`

**Secondary:**
- None — pure Go project

## Runtime

**Environment:**
- Go runtime 1.26.0 (specified in `go.mod`)

**Package Manager:**
- Go modules (`go mod`)
- Lockfile: `go.sum` present and committed

## Frameworks

**Core:**
- `github.com/spf13/cobra v1.10.2` — CLI command tree, flag parsing, subcommand dispatch (`pkg/cmd/root.go`)
- `github.com/spf13/pflag v1.0.9` — Enhanced POSIX flag handling (cobra dependency)

**Testing:**
- `github.com/stretchr/testify v1.11.1` — Assertions and test suite support (used throughout `pkg/**/*_test.go`)

**Build/Dev:**
- `github.com/magefile/mage v1.15.0` — Build task runner (`magefile.go`); replaces Makefile
- `golangci-lint v2.10.1` — Linter + formatter; pinned in `.github/workflows/ci.yml`
- `goreleaser v2` — Cross-platform release builds and Homebrew cask publishing (`.goreleaser.yaml`)
- `release-please` — Automated changelog and release PR management (`.release-please-config.json`)

## Key Dependencies

**Critical:**
- `github.com/pelletier/go-toml/v2 v2.2.4` — TOML parsing and marshaling for `.gitw` config files (`pkg/toml/`, `pkg/config/`)
- `github.com/fatih/color v1.18.0` — Terminal color output for status display (`pkg/display/colors.go`)
- `github.com/spf13/cobra v1.10.2` — Entire CLI surface, subcommand registration, help generation

**Infrastructure:**
- `github.com/mattn/go-colorable v0.1.13` — Windows-compatible ANSI color output (indirect, via fatih/color)
- `github.com/mattn/go-isatty v0.0.20` — TTY detection for color auto-disable (indirect, via fatih/color)
- `golang.org/x/sys v0.25.0` — OS-level syscalls (indirect)
- `gopkg.in/yaml.v3 v3.0.1` — YAML support (indirect, via testify)

## Configuration

**Application Config:**
- User-facing config: `.gitw` TOML file, discovered by walking up from CWD or via `GIT_W_CONFIG` env var
- Local overrides: `.gitw.local` TOML file (merged after main config; not committed to VCS)
- Config loading entrypoint: `pkg/config/loader.go`
- Discovery logic: `pkg/config/discovery.go`

**Build:**
- Build config: `magefile.go` (mage tasks: `All`, `Build`, `Test`, `TestFast`, `Lint`, `LintFix`, `Fmt`, `Cover`)
- Release config: `.goreleaser.yaml`
- Lint config: `.golangci.yml` (minimal; uses golangci-lint v2 defaults)
- Release-please config: `.release-please-config.json`

**Version Injection:**
- Version set at build time via `-ldflags "-X main.version=<tag>"` using `git describe --tags`
- Version exposed via `root.Version` in cobra command (`pkg/cmd/root.go`)

## Platform Requirements

**Development:**
- Go 1.26.0+
- `golangci-lint` (install separately; used by `mage Lint`)
- `git` binary on `$PATH` (the CLI wraps git commands)

**Production / Distribution:**
- Distributed as a static binary: `bin/git-w`
- Cross-compiled targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Distributed via GitHub Releases (tar.gz archives) and Homebrew cask (`robertwritescode/homebrew-tap`)
- Must be on `$PATH` as `git-w` to be invoked as `git w <cmd>` via git plugin system

---

*Stack analysis: 2026-04-01*
