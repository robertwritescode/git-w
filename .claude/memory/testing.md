# Testing Strategy

## Guiding Principles

- All non-trivial logic must have unit tests
- **Table-driven tests are required** (not optional) for any test that can be parameterised — any test with multiple input/output scenarios must use `[]struct{ name, input, want }`
- **testify/suite is required** in all test files — see pattern below
- No other test libraries (no gomock, no ginkgo)
- No mocking framework — use `t.TempDir()` and real subprocesses instead
- Subprocess-heavy code: **separate parsing from invocation** so parsing is purely functional and testable with fixture strings
- Filesystem tests: `t.TempDir()` for isolated, auto-cleaned directories
- Real git repos (created via `git init` in `t.TempDir()`) for integration-style unit tests where subprocess behavior must be verified

---

## Required Test Pattern: testify/suite + Table-Driven

Every test file **must** use the testify suite pattern. Bare `func TestXxx(t *testing.T)` functions are only acceptable as the single `suite.Run` entry point per suite.

```go
package workspace_test

import (
    "testing"

    "github.com/stretchr/testify/suite"
)

type LoaderSuite struct {
    suite.Suite
    tmpDir string
}

// SetupTest runs before each test method — use for per-test isolation
func (s *LoaderSuite) SetupTest() {
    s.tmpDir = s.T().TempDir()
}

// Single-case tests: method on suite, use s.Require() / s.Assert()
func (s *LoaderSuite) TestLoad_MissingFile() {
    _, err := Load("/nonexistent/.gitw")
    s.Require().Error(err)
}

// Multi-case tests: table-driven inside a suite method, use s.Run()
func (s *LoaderSuite) TestLoad_ValidToml() {
    cases := []struct {
        name  string
        input string
        want  string
    }{
        {"workspace name", `[workspace]\nname = "foo"`, "foo"},
        {"empty file", ``, ""},
    }
    for _, tc := range cases {
        s.Run(tc.name, func() {
            // write tc.input to s.tmpDir, load, assert
            s.Assert().Equal(tc.want, got.Workspace.Name)
        })
    }
}

// Entry point — one per file
func TestLoader(t *testing.T) {
    suite.Run(t, new(LoaderSuite))
}
```

**Key rules:**
- `s.Require()` — fatal assertions (setup steps, preconditions); stops the test immediately on failure
- `s.Assert()` — non-fatal assertions (value checks); test continues on failure
- `s.Run(tc.name, func() { ... })` — creates sub-tests within table-driven loops, giving clear failure output
- `SetupTest` / `TeardownTest` — per-test lifecycle hooks (prefer over `SetupSuite` unless truly shared state)
- `s.T().TempDir()` — temp directory scoped to the test, auto-cleaned

---

## Test File Placement

Standard Go pattern: `foo_test.go` alongside `foo.go` in the same directory.

- Library files in `pkg/workspace`, `pkg/repo`, `pkg/git`: use `package <domain>` (white-box) or `package <domain>_test` (black-box) as appropriate
- Command files in the same domain packages: use `package <domain>_test` (black-box); call via `s.ExecuteCmd()`
- `pkg/cmd`: completion test uses `package cmd` (white-box) because `registerCompletion` is private

---

## Shared Test Helpers (`pkg/testutil/`)

A package of reusable test fixtures and integration test infrastructure.

### `helpers.go` — Standalone helper functions (unexported; accessed via CmdSuite methods)
```go
makeGitRepo(t, remoteURL string) string          // git repo with initial commit in TempDir
makeGitRepoAt(t, base, sub, name string) string  // git repo at specific path
initBareGitRepo(t, dir string)                   // bare git repo for clone tests
makeWorkspace(t, dir, content string) string     // .gitw with TOML content
changeToDir(t, dir string)                       // chdir + restore on cleanup
setupWorkspaceDir(t) string                      // TempDir with minimal .gitw
appendGroup(t, wsDir, groupName, repoName string)
setActiveContext(t, wsDir, ctxName string)
createBareRepo(t) (dir, url string)
pushToRemote(t, repoDir string)
makeWorkspaceWithRepos(t, repos map[string]string) (string, map[string]string)
makeWorkspaceFromPaths(t, repos map[string]string) string
makeWorkspaceWithNLocalRepos(t, n int) (string, []string)
makeWorkspaceWithNRemoteRepos(t, n int) (string, []string)
makeWorkspaceWithRepoNames(t, repoNames []string, extraTOML string) string
```

### `cmd.go` + `suite.go` — Integration test suite base
```go
// cmd.go
type CmdSuite struct {
    suite.Suite
    Root *cobra.Command
}

func (s *CmdSuite) SetRoot(register func(*cobra.Command)) // creates fresh root cmd + registers command
func (s *CmdSuite) ExecuteCmd(args ...string) (string, error) // runs command, returns stdout

// suite.go — all helpers from helpers.go available as CmdSuite methods:
func (s *CmdSuite) MakeGitRepo(remoteURL string) string
func (s *CmdSuite) MakeGitRepoAt(base, sub, name string) string
func (s *CmdSuite) InitBareGitRepo(dir string)
func (s *CmdSuite) MakeWorkspace(dir, content string) string
func (s *CmdSuite) ChangeToDir(dir string)
func (s *CmdSuite) SetupWorkspaceDir() string
func (s *CmdSuite) AppendGroup(wsDir, groupName, repoName string)
func (s *CmdSuite) SetActiveContext(wsDir, ctxName string)
func (s *CmdSuite) CreateBareRepo() (string, string)
func (s *CmdSuite) PushToRemote(repoDir string)
func (s *CmdSuite) MakeWorkspaceWithRepos(repos map[string]string) (string, map[string]string)
func (s *CmdSuite) MakeWorkspaceFromPaths(repos map[string]string) string
func (s *CmdSuite) MakeWorkspaceWithNLocalRepos(n int) (string, []string)
func (s *CmdSuite) MakeWorkspaceWithNRemoteRepos(n int) (string, []string)
func (s *CmdSuite) MakeWorkspaceWithRepoNames(repoNames []string, extraTOML string) string
```

---

## Package-Level Testing Notes

### `pkg/workspace/`

| File | What to test |
|---|---|
| `loader.go` | TOML round-trip (load → mutate → save → load again); atomic write (`.tmp` file renamed on success); missing-file sentinel error; malformed TOML error; `.local` values override `.gitw` values after merge; `LoadConfig` reads `--config` flag and delegates to `LoadCWD` |
| `discovery.go` | Walk-up finds `.gitw` at 0, 1, and 2 directory levels above CWD; `GIT_W_CONFIG` env var override; not-found returns a distinct sentinel error |
| `init.go`, `context.go`, `group.go` | Cobra integration tests via `s.SetRoot(workspace.Register)` + `s.ExecuteCmd(args...)` |

Use `t.TempDir()` to create real directory trees with nested subdirectories. No git subprocess needed.

### `pkg/repo/`

| File | What to test |
|---|---|
| `repo.go` | `FromConfig` path resolution, sorted output; `FromNames` selective resolution; `IsGitRepo` true/false |
| `filter.go` | `Filter` with explicit names, active context, group expansion, dedup; `ForContext`, `ForGroup` |
| `status.go` | Parse functions take `[]byte` — test all status states with fixture strings; one smoke test against a real temp git repo |
| `add.go`, `clone.go`, `unlink.go`, `rename.go`, `restore.go`, `list.go` | Cobra integration tests via `s.SetRoot(repo.Register)` — commands are under `repo` subcommand (e.g. `s.ExecuteCmd("repo", "add", ...)`) except `restore` which is on root |

**Key pattern for `status.go`**: keep the `git` subprocess call as a thin, untested wrapper. All business logic lives in pure parse functions:

```go
// internal — each tested with fixture []byte strings
func parsePorcelainV1(stdout []byte) (dirty, staged, untracked bool)
func parseBranchLine(line string) (branch string, remote RemoteState)
func parseStashCount(stdout []byte) int
```

Fixture strings should cover: clean, dirty only, staged only, untracked only, stashed, all combined; and for remote state: in-sync, local-ahead, remote-ahead, diverged, no-remote, no-upstream-configured.

### `pkg/git/`

| File | What to test |
|---|---|
| `result.go` | `WriteResults` output formatting; `ExecErrors` error aggregation; `prefixLines` prefix insertion |
| `executor.go` | Run `echo` across multiple `Repo` values; verify all results collected; verify concurrency limit via atomic counter; verify output prefixed `[name]`; single-repo case has no prefix; timeout cancels goroutines |
| `commands.go`, `exec.go`, `info.go` | Cobra integration tests via `s.SetRoot(gitpkg.Register)` (use `gitpkg` alias) |

### `pkg/parallel/`

| File | What to test |
|---|---|
| `parallel.go` | `MaxWorkers` bounds (0 input → NumCPU, configured > total → total, minimum 1); `RunFanOut` result ordering, concurrency; `FormatFailureError` nil when empty, formatted message when non-empty |

### `pkg/display/`

| File | What to test |
|---|---|
| `colors.go` | `visualWidth()` strips ANSI escape codes and returns correct visual length for both plain and colored strings |
| `table.go` | Set `color.NoColor = true` in `SetupTest`; render table with fixture `TableEntry` values; compare output to golden strings |

### `pkg/gitutil/`

| File | What to test |
|---|---|
| `gitutil.go` | `RemoteURL` with/without remote; `EnsureGitignore` creates, appends, deduplicates, handles missing trailing newline; concurrent safety (mutex); `Clone`/`CloneContext` with `file://` URL |

### `pkg/cmd/`

| File | What to test |
|---|---|
| `completion.go` | All shell types return non-empty output; invalid shell returns error. Test uses `package cmd` (internal) to access private `registerCompletion` |

### Domain command integration tests

Each test suite embeds `testutil.CmdSuite` and calls `s.SetRoot(<domain>.Register)` — this registers ALL commands in the domain, not just one. `s.ExecuteCmd(args...)` exercises the full path.

| Command group | Domain | Subcommand path | Needs real git repo? |
|---|---|---|---|
| `init`, `context`, `group` | workspace | direct on root | No |
| `repo list`, `repo rename`, `repo unlink` | repo | under `repo` subcommand | No |
| `repo add`, `repo clone`, `restore` | repo | `add`/`clone` under `repo`; `restore` direct | Yes |
| `completion` | cmd | direct on root | No |
| `fetch`, `pull`, `push`, `status`, `exec`, `info` | git | direct on root | Yes |

---

## Known Pitfalls (learned during implementation)

### SetupTest does NOT run per s.Run sub-test

`SetupTest` fires once per top-level test *method*, not per `s.Run` closure. Table-driven
sub-tests that need isolated state (clean workspace, fresh CWD) must set up that state
themselves inside the closure. Pattern used in `info_test.go`:

```go
for _, tt := range tests {
    s.Run(tt.name, func() {
        // Do NOT rely on SetupTest having prepared a clean workspace.
        wsDir := s.T().TempDir()
        s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), ...))
        s.ChangeToDir(wsDir)
        // ... test body
    })
}
```

### Cobra/pflag flag vars persist between Execute() calls

pflag does not reset flag variables to their defaults between `Execute()` calls when the
flag is absent from args. A test that calls `repo add -g mygroup <repo1>`
then `repo add <repo2>` in the same workspace will silently register repo2 in mygroup too.

Workaround: avoid relying on flag-driven setup when testing downstream commands. Instead,
manipulate config directly after registering repos:

```go
cfg, _ := workspace.Load(cfgPath)
cfg.Groups["mygroup"] = workspace.GroupConfig{Repos: []string{repoName}}
workspace.Save(cfgPath, cfg)
```

Prefer inline `&cobra.Command{...}` in `register<Name>` functions (not package-level `var xxxCmd`)
to avoid flag state bleeding between test runs. Use named vars only when flags must be attached.

### Disable colors in display tests

`color.NoColor = true` in `SetupTest` makes `fatih/color` Sprint calls return plain
strings regardless of terminal type. Required for reliable `Contains` checks and
`visualWidth` assertions in `pkg/display/` tests.

### Verify plan specs against actual values

Spec documents can contain typos in expected values. Always verify before writing tests.
Example: Phase 2 plan stated `visualWidth("main ✓") == 7`; actual is 6 (6 runes).

### MarkFlagRequired + pflag state: don't test "missing required flag" via execCmd

`MarkFlagRequired` uses pflag's `Changed()` state. After any `execCmd` call that sets a flag,
the flag's `Changed` state persists into the next `execCmd` call. Testing "error when required
flag absent" via `execCmd` is therefore unreliable and should be omitted — it tests cobra's
built-in enforcement, not application logic.

### Multi-line command output invalidates `strings.Count == 1` assertions

When asserting deduplication (repo ran exactly once), `strings.Count(out, "[repo]") == 1` fails
if the git command produces multiple prefixed lines. Instead, establish a baseline with a single-repo
run and assert `strings.Count(dedup_out, "[repo]") == strings.Count(baseline_out, "[repo]")`.

### Never use execCmd to set up group/context state for downstream tests

Same-package global vars persist between `execCmd` calls. Always write `.gitw`
and `.gitw.local` directly via `os.WriteFile` or suite helper methods for test fixture setup.

### Repo command args include the "repo" subcommand prefix

Because `repo.Register` nests add/clone/unlink/rename/list under a `repo` parent command,
integration tests must include `"repo"` in the args:

```go
// correct
out, err := s.ExecuteCmd("repo", "add", "/path/to/repo")

// wrong — "add" is not a direct subcommand of root
out, err := s.ExecuteCmd("add", "/path/to/repo")
```

`restore` is the exception — it is added directly to root by `repo.Register`.

---

## CI Configuration

Both `ci.yml` (PR gate) and `goreleaser.yml` (release gate) run:

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
| `mage lint` | `golangci-lint fmt --diff` + `golangci-lint run` |
