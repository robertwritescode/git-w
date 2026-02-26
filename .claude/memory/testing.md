# Testing Strategy

## Guiding Principles

- All non-trivial logic must have unit tests
- Table-driven tests (`[]struct{ name, input, want }`) preferred throughout
- **testify** (`github.com/stretchr/testify`) used in all test files:
  - `require.NoError` / `require.Equal` — fatal setup assertions (stop test immediately)
  - `assert.Equal` / `assert.Contains` — non-fatal value checks
- No other test libraries (no gomock, no ginkgo)
- No mocking framework — use `t.TempDir()` and real subprocesses instead
- Subprocess-heavy code: **separate parsing from invocation** so parsing is purely functional and testable with fixture strings
- Filesystem tests: `t.TempDir()` for isolated, auto-cleaned directories
- Real git repos (created via `git init` in `t.TempDir()`) for integration-style unit tests where subprocess behavior must be verified

---

## Test File Placement

Standard Go pattern: `foo_test.go` alongside `foo.go` in the same directory.

- `internal/` packages: use `package config` (white-box) to access unexported helpers where necessary
- `cmd/` package: use `package cmd_test` (black-box); call `Execute()` with captured stdout/stderr

---

## Shared Test Helpers (`internal/testutil/`)

A small package of reusable test fixtures — only helpers used by 2+ packages.

```go
// MakeGitRepo initialises a bare git repo with one commit in a new t.TempDir()
// subdirectory. Optionally sets a remote URL. Returns the absolute path.
func MakeGitRepo(t testing.TB, remoteURL string) string

// MakeWorkspace creates a temp dir containing a minimal .gitworkspace TOML file
// with the provided content. Returns the workspace root path.
func MakeWorkspace(t testing.TB, toml string) string
```

---

## Package-Level Testing Notes

### `internal/config/`

| File | What to test |
|---|---|
| `loader.go` | TOML round-trip (load → mutate → save → load again); atomic write (`.tmp` file renamed on success); missing-file sentinel error; malformed TOML error; `.local` values override `.gitworkspace` values after merge |
| `discovery.go` | Walk-up finds `.gitworkspace` at 0, 1, and 2 directory levels above CWD; `GIT_WORKSPACE_CONFIG` env var override; not-found returns a distinct sentinel error |

Use `t.TempDir()` to create real directory trees with nested subdirectories. No git subprocess needed.

### `internal/repo/`

| File | What to test |
|---|---|
| `repo.go` | `AbsPath` correctly joins config root + relative path; `IsGitRepo` returns true for a dir containing `.git/`, false otherwise |
| `status.go` | Parse functions take `[]byte` stdout — test all status states with fixture strings; one integration smoke test using a real temp git repo |

**Key pattern for `status.go`**: keep the `git` subprocess call as a thin, untested wrapper. All business logic lives in pure parse functions:

```go
// internal — each tested with fixture []byte strings
func parsePorcelainV1(stdout []byte) (dirty, staged, untracked bool)
func parseBranchLine(line string) (branch string, remote RemoteState)
func parseStashCount(stdout []byte) int
```

Fixture strings should cover: clean, dirty only, staged only, untracked only, stashed, all combined; and for remote state: in-sync, local-ahead, remote-ahead, diverged, no-remote, no-upstream-configured.

### `internal/executor/`

| File | What to test |
|---|---|
| `result.go` | Output formatting helpers; prefix insertion; non-zero exit code representation |
| `parallel.go` | Run `echo` across multiple `Repo` values; verify all results collected; verify concurrency limit via an atomic counter in test command; verify output is prefixed `[name]`; single-repo case has no prefix; timeout cancels in-flight goroutines |

### `internal/display/`

| File | What to test |
|---|---|
| `colors.go` | `visualWidth()` strips ANSI escape codes and returns correct visual length for both plain and colored strings |
| `table.go` | Set `color.NoColor = true` in `TestMain` or per-test; render table with fixture `RepoStatus` values; compare output to a golden string stored inline |

### `cmd/`

Cobra integration tests exercise the full command path through a real temp workspace.

```go
// runCmd sets working dir, constructs root cmd, captures stdout/stderr, calls Execute().
func runCmd(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int)
```

| Command group | Needs real git repo? |
|---|---|
| `init`, `ls`, `rename`, `rm`, `group`, `context` | No — static config file only |
| `add`, `status`, `fetch`, `pull`, `push`, `exec`, `restore` | Yes — `testutil.MakeGitRepo` |

---

## CI Configuration

Both `ci.yml` (PR gate) and `release.yml` (release gate) run:

```
go test -race -count=1 ./...
```

- `-race` — data race detector (required; catches goroutine bugs in executor)
- `-count=1` — disables test result caching (ensures tests actually run in CI)

See `release.md` for exact workflow YAML.

## Local Development (Mage Targets)

| Target | Command |
|---|---|
| `mage test` | `go test -race -count=1 ./...` |
| `mage cover` | `go test -coverprofile=coverage.out ./...` then open HTML report |
| `mage check` | `mg.Deps(Lint, Test)` — vet + test |
