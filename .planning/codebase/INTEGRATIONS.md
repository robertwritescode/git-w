# External Integrations

**Analysis Date:** 2026-04-01

## APIs & External Services

**Git (Local Process):**
- The application's primary integration is with the local `git` binary via `os/exec`
- All git operations are performed by spawning `git` subprocesses in each repo's directory
- Client: `os/exec.CommandContext` in `pkg/git/executor.go`
- No remote API calls to GitHub, GitLab, or any git hosting service
- Concurrency model: fan-out goroutine pool in `pkg/parallel/parallel.go`

**No other external APIs detected.**

## Data Storage

**Databases:**
- None — no database, no ORM, no connection strings

**File Storage:**
- Local filesystem only
- Config file: `.gitw` (TOML) discovered by walking from CWD upward (`pkg/config/discovery.go`)
- Local overrides: `.gitw.local` (TOML), same directory as `.gitw`, not VCS-committed
- Workgroup worktrees stored under `.workgroup/<name>/<repo>/` relative to workspace root
- Atomic writes: temp file + `os.Rename` pattern in `pkg/config/loader.go` (`atomicWriteFile`)

**Caching:**
- None

## Authentication & Identity

**Auth Provider:**
- None — the tool inherits whatever git credentials are configured on the host (SSH keys, credential helpers, etc.)
- No authentication tokens managed by the application itself

## Monitoring & Observability

**Error Tracking:**
- None

**Logs:**
- No structured logging framework; errors are returned via Go's `error` interface and printed to `stderr` via `fmt.Fprintln(os.Stderr, err)` in `main.go`
- Command output (stdout/stderr from git subprocesses) is streamed or buffered depending on execution mode (serial vs async) in `pkg/git/executor.go`

## CI/CD & Deployment

**Hosting:**
- GitHub Releases — binary archives (`tar.gz`) for each platform
- Homebrew tap: `robertwritescode/homebrew-tap` (Casks/git-w.rb)

**CI Pipeline:**
- GitHub Actions
  - `ci.yml`: lint (`golangci-lint`) + test (`go test -race`) + build, runs on every push
  - `release.yml`: triggered on push to `main`; uses `release-please` for changelog/version PR, then `goreleaser` for multi-platform builds and Homebrew cask publish

**Release Automation:**
- `googleapis/release-please-action@v4` — creates release PRs, manages `CHANGELOG.md` and version manifest (`.release-please-manifest.json`)
- `goreleaser/goreleaser-action@v6` — builds binaries, creates GitHub Release assets, pushes Homebrew cask to tap repo

## Environment Configuration

**Required env vars:**
- `GIT_W_CONFIG` — optional; overrides `.gitw` config file path (bypasses CWD discovery); documented in `pkg/config/discovery.go`

**CI-only secrets (not needed for development):**
- `GITHUB_TOKEN` — standard GitHub Actions token for creating releases
- `TAP_GITHUB_TOKEN` — personal access token for pushing to `robertwritescode/homebrew-tap`

**Secrets location:**
- GitHub repository secrets (used only in CI workflows)
- No `.env` file or secrets file in repository

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

---

*Integration audit: 2026-04-01*
