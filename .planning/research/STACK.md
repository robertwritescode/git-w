# Stack Research

**Domain:** Go CLI tool — multi-repo git orchestration (v2 new dependencies)
**Researched:** 2026-04-01
**Confidence:** HIGH

> Scope: Only NEW dependencies and patterns for v2 features. Existing stack
> (cobra, go-toml/v2, testify, text/tabwriter, os/exec git subprocess) is
> documented in `.planning/codebase/STACK.md` and not repeated here.

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `code.gitea.io/sdk/gitea` | v0.24.1 | Gitea API provider (`internal/provider/`) | First-party SDK maintained by the Gitea project. Provides typed `CreateRepo`, `GetRepo`, `ListRepos` methods that map directly to the v2 Provider interface (`RepoExists`, `CreateRepo`). Single import, no hand-rolled HTTP. Confirmed current on pkg.go.dev 2026-03-24. |
| `github.com/google/go-github/v84` | v84.0.0 | GitHub API provider (`internal/provider/`) | Google-maintained, tracks GitHub REST API changes within weeks. `RepositoriesService.Create`/`Get` for repo CRUD, `PullRequestsService.Create` for PR opening. v84 targets Go 1.26. Confirmed on GitHub releases 2026-02-27. |
| `github.com/bmatcuk/doublestar/v4` | v4.10.0 | Glob pattern matching for branch rules | `Match(pattern, name)` with `**` crossing `/` and `*` not crossing `/`, matching the branch-rule engine spec exactly. Cross-platform (no OS path separator quirks like `filepath.Match`). Zero dependencies. Confirmed on pkg.go.dev 2026-01-25. |
| `golang.org/x/sync/errgroup` | v0.20.0 | Bounded parallel fan-out (sync executor) | `errgroup.SetLimit(n)` provides bounded worker pools without manual semaphore+WaitGroup. Replaces the existing `parallel.RunFanOut` pattern for sync fan-out. Part of the official Go sub-repos. Confirmed on pkg.go.dev 2026-02-23. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` (stdlib) | Go 1.26 | JSON output contracts (`--json` flag) | All `--json` output. Simple struct marshaling to stdout. No `encoding/json/v2` in stdlib yet (confirmed via Go 1.26 release notes); stdlib v1 is sufficient for flat output structs. |
| `os` + `encoding/json` (stdlib) | Go 1.26 | State file read/write (`.git/git-w-state.json`) | Atomic write pattern: write to temp file, rename into place. No external library needed. |
| `go-toml/v2` (existing) | v2.2.3 | TOML config merge semantics | Already in go.mod. v2 merge logic (`.gitw` + `.gitw.local` layering) lives in `pkg/config`. No new dependency; just new merge functions. |
| `net/http` (stdlib) | Go 1.26 | HTTP client for API providers | Both Gitea SDK and go-github accept a custom `*http.Client`. Pass stdlib client with timeouts; no need for resty or similar. |
| `context` (stdlib) | Go 1.26 | Cancellation for API calls and errgroup | `errgroup.WithContext` propagates cancellation on first error. API provider calls use `context.Context` as first parameter per Go idioms. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `mage` (existing) | Build/test orchestration | No change. `mage test` / `mage testfast` / `mage lint` continue to work. |
| `golangci-lint` (existing) | Linting | No change. Existing config covers new packages. |
| `go test -race` (via mage) | Race detection for errgroup concurrency | Critical for the new errgroup-based fan-out. `mage test` already runs with `-race`. |

## Installation

```bash
# New runtime dependencies
go get code.gitea.io/sdk/gitea@v0.24.1
go get github.com/google/go-github/v84@v84.0.0
go get github.com/bmatcuk/doublestar/v4@v4.10.0
go get golang.org/x/sync@v0.20.0
```

No new dev dependencies required. No new build tooling.

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `go-github/v84` | `go-gitlab` | If GitLab support is added later. Same Provider interface pattern; swap implementation. Not needed for v2 scope. |
| `doublestar/v4` | Custom glob engine (`internal/glob/`) | Never. doublestar is zero-dep, battle-tested, and matches the exact `*`/`**` semantics the spec requires. Writing a custom engine adds maintenance burden for zero benefit. |
| `doublestar/v4` | `filepath.Match` (stdlib) | Never for branch rules. `filepath.Match` uses OS-specific path separators and does not support `**`. The spec explicitly warns against it. |
| `errgroup` | Manual semaphore + WaitGroup (current `parallel.RunFanOut`) | Keep the existing `parallel.RunFanOut` for the fire-and-forget async pattern (where all repos run regardless of errors). Use errgroup only for the NEW sync fan-out executor where first-error cancellation is desired. Both patterns coexist. |
| Gitea SDK | Raw HTTP calls to Gitea API | Never. The SDK handles pagination, auth token injection, and API versioning. Hand-rolling HTTP would duplicate what the SDK already does. |
| `encoding/json` (stdlib) | `json-iterator/go` or `segmentio/encoding` | Not needed. JSON output is small structs (repo status, command results). Marshaling performance is irrelevant at this scale. Extra dependency for zero user-visible benefit. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `go-git` (`github.com/go-git/go-git/v5`) | The v2 spec mentions "go-git only" for remote upsert, but the entire existing codebase uses `os/exec` subprocess calls via `pkg/gitutil/`. Introducing go-git creates two incompatible git execution paths, doubles the surface area for bugs, and adds a massive dependency tree (~30 transitive deps). | Continue using `os/exec` via `pkg/gitutil/` for `git remote add` / `git remote set-url`. Consistent with existing patterns, zero new dependencies, same behavior users expect from their git installation. |
| `bubbletea` / `lipgloss` / `charm` | Project constraint: no TUI frameworks. Plain formatted output via `text/tabwriter`. | `text/tabwriter` (existing) + `output.Writef` (existing) for all output. |
| `resty` / `go-retryablehttp` | Unnecessary HTTP abstraction layer. Both Gitea SDK and go-github handle their own HTTP transport. | Stdlib `net/http` with `context.Context` timeouts. Pass a configured `*http.Client` to each SDK. |
| `viper` | Heavyweight config framework. The project already has lean TOML loading via `go-toml/v2` + `pkg/config`. Viper would add 10+ transitive dependencies for features we don't need (env vars, remote config, YAML). | Existing `go-toml/v2` config loading with new merge functions in `pkg/config`. |
| `cobra-cli` (scaffolding) | Code generation produces boilerplate that violates project conventions (package-level vars, stuttering names). | Hand-write commands following the domain package convention in AGENTS.md §8. |
| `oauth2` / `golang.org/x/oauth2` | Over-engineered for this use case. API providers use static personal access tokens from config, not OAuth flows. | Read token from TOML config (`[providers.github].token`), pass directly to SDK client constructors. |

## Stack Patterns by Variant

**If adding a new API provider (e.g., GitLab):**
- Implement the `Provider` interface in `internal/provider/gitlab/`
- Add `go get github.com/xanzy/go-gitlab` (or successor)
- Register in the provider factory. No architectural changes needed.

**If sync fan-out needs fire-and-forget semantics:**
- Do NOT use errgroup for that case. errgroup cancels on first error.
- Keep `parallel.RunFanOut` for fire-and-forget (all repos execute regardless of errors).
- Use errgroup only when first-error cancellation is the desired behavior.

**If JSON output grows complex (nested objects, streaming):**
- stdlib `encoding/json` is still sufficient. Use `json.NewEncoder(w).Encode()` for streaming.
- Only reconsider if profiling shows marshaling as a bottleneck (extremely unlikely for CLI output).

**If branch rule patterns need negation (`!pattern`):**
- `doublestar/v4` does not support negation natively.
- Implement negation as a wrapper: check `!` prefix, strip it, call `doublestar.Match`, invert result.
- This is simpler than switching libraries or writing a custom engine.

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `go-github/v84` v84.0.0 | Go 1.26+ | v84 module path includes major version. Uses generics (Go 1.18+). |
| `gitea-sdk` v0.24.1 | Go 1.21+ | No generics dependency. Compatible with Go 1.26. |
| `doublestar/v4` v4.10.0 | Go 1.18+ | Minimal Go version requirement. Zero external dependencies. |
| `golang.org/x/sync` v0.20.0 | Go 1.22+ | `errgroup.SetLimit` available since x/sync v0.7.0 (2024). v0.20.0 is latest. |
| `go-github/v84` | `gitea-sdk` v0.24.1 | No conflict. Both use stdlib `net/http`. Independent provider implementations. |
| `errgroup` | existing `parallel.RunFanOut` | Coexist. Different use cases (cancelling vs. fire-and-forget). No conflict. |

## Sources

- pkg.go.dev `code.gitea.io/sdk/gitea` — confirmed v0.24.1 published 2026-03-24. HIGH confidence.
- GitHub releases `google/go-github` — confirmed v84.0.0 published 2026-02-27. HIGH confidence.
- pkg.go.dev `github.com/bmatcuk/doublestar/v4` — confirmed v4.10.0 published 2026-01-25. HIGH confidence.
- pkg.go.dev `golang.org/x/sync` — confirmed v0.20.0 published 2026-02-23. HIGH confidence.
- Go 1.26 release notes — confirmed no `encoding/json/v2` in stdlib. HIGH confidence.
- `pkg/parallel/parallel.go` source — confirmed current RunFanOut uses manual semaphore+WaitGroup. HIGH confidence.
- `.planning/v2/v2-remote-management.md` — spec for sync fan-out, provider interface, hook mechanism. HIGH confidence.
- `.planning/v2/v2-commands.md` — spec for `--json` output contracts, command surface. HIGH confidence.
- `.planning/v2/v2-schema.md` — spec for TOML merge semantics, branch rule globs. HIGH confidence.

---
*Stack research for: git-w v2 new dependencies*
*Researched: 2026-04-01*
