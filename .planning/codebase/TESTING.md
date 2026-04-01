# Testing Patterns

**Analysis Date:** 2026-04-01

## Test Framework

**Runner:**
- Go's built-in `testing` package
- `testify/suite` v1.11.1 — all suites use `suite.Suite` embedding
- Config: no separate config file; run via `go test` or `mage Test`/`mage TestFast`

**Assertion Library:**
- `github.com/stretchr/testify` — `require` (fail immediately) and `assert` (continue after failure)

**Run Commands:**
```bash
mage Test              # Full test suite: clean cache + race detector (-race -count=1 -p=8)
mage TestFast          # Fast local dev: no race detector (-p=8)
mage Cover             # Coverage report: generates coverage.out and opens HTML
go test ./...          # Direct Go test invocation
go test -run TestFoo   # Single test
```

## Test File Organization

**Location:**
- Co-located with source in the same package directory
- Example: `pkg/worktree/add.go` ↔ `pkg/worktree/add_test.go`

**Naming:**
- `<operation>_test.go` matching source file: `add_test.go`, `drop_test.go`, `clone_test.go`
- Package-level shared fixtures: `fixtures_test.go`
- Package-level shared assertions: `assertions_test.go`

**Package Declaration:**
- Black-box tests (command integration): `package repo_test` — external package view
- White-box tests (unexported functions): `package repo` (same package name, no `_test` suffix)
  - Example: `pkg/repo/status_test.go` uses `package repo` to access `parsePorcelainV1`, `parseBranchLine`
  - Example: `pkg/worktree/safety_test.go` uses `package worktree` to access `safetyViolations`
  - Example: `pkg/parallel/parallel_test.go` uses `package parallel`
  - Example: `pkg/display/table_test.go` uses `package display`

**Structure:**
```
pkg/<feature>/
├── add.go
├── add_test.go          # black-box: package <feature>_test
├── drop.go
├── drop_test.go
├── fixtures_test.go     # shared setup helpers for the package's tests
├── assertions_test.go   # shared assertion helpers
└── register.go
```

## Test Structure

**Suite Organization:**
```go
type WorktreeAddSuite struct {
    testutil.CmdSuite   // embeds suite.Suite + helpers
}

func TestWorktreeAddSuite(t *testing.T) {
    s := new(WorktreeAddSuite)
    s.InitRoot(worktree.Register)   // register the cobra subcommand under test
    testutil.RunSuite(t, s)
}

func (s *WorktreeAddSuite) TestAddBranch() {
    // arrange
    wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev"})
    s.Require().NoError(err)

    // act
    _, err = s.ExecuteCmd("worktree", "add", "infra", "test")
    s.Require().NoError(err)

    // assert
    cfg, err := config.Load(filepath.Join(wsDir, ".gitw"))
    s.Require().NoError(err)
    s.Assert().Equal("infra/test", cfg.Worktrees["infra"].Branches["test"])
}
```

**Patterns:**
- Suite's `SetupTest()` resets the cobra command tree for each test via `CmdSuite.SetupTest()`
- Custom `SetupTest()` calls `s.CmdSuite.SetupTest()` before additional per-test setup
- `s.Require()` for setup steps — test cannot continue if these fail
- `s.Assert()` for assertions — collect multiple failures within a test
- Sub-tests via `s.Run(tt.name, func() { ... })` for table-driven test cases

## Mocking

**Framework:** None — no mock library used

**Philosophy:** The codebase uses **real filesystem and real git operations** rather than mocks. Integration tests run actual `git` commands in temp directories.

**What to Mock:**
- Terminal colors: `color.NoColor = true` in `SetupTest()` to disable ANSI codes in output assertions

**What NOT to Mock:**
- Git operations — always use real git repos via `testutil` helpers
- Config file reads/writes — use real temp files
- Filesystem operations — use `t.TempDir()` for isolation

## Fixtures and Factories

**Centralized Test Infrastructure: `pkg/testutil/`**

`pkg/testutil/helpers.go` — unexported primitives, `pkg/testutil/suite.go` — exported `CmdSuite` methods wrapping them:

```go
// Creates temp git repo with initial commit, optional origin remote
wsDir := s.MakeGitRepo("file:///path/to/remote")

// Creates temp workspace dir with minimal .gitw config, chdir into it
wsDir := s.SetupWorkspaceDir()

// Creates N local repos + workspace config, returns wsDir and repo names
wsDir, names := s.MakeWorkspaceWithNLocalRepos(3)

// Creates N repos each backed by a bare remote
wsDir, names := s.MakeWorkspaceWithNRemoteRepos(2)

// Creates bare remote with specified branches pushed
remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})

// Creates workspace from a name→path map
wsDir := s.MakeWorkspaceFromPaths(map[string]string{"svc-a": "/abs/path"})

// Appends a [groups.<name>] section to the workspace config
s.AppendGroup(wsDir, "web", "frontend")

// Sets active context in .gitw.local
s.SetActiveContext(wsDir, "web")

// Changes CWD to dir, restores after test
s.ChangeToDir(wsDir)

// Execute cobra command, returns combined output and error
out, err := s.ExecuteCmd("repo", "add", repoDir)
```

**Test Identity:**
- All real git operations in tests set a fixed identity via `withTestGitIdentity`:
  ```
  GIT_AUTHOR_NAME=Test User
  GIT_AUTHOR_EMAIL=test@example.com
  ```

**Fixture Isolation:**
- Every test gets its own `t.TempDir()` — cleaned up automatically
- `t.Cleanup()` used for any non-TempDir resource (e.g. extra directories within workspace)
- CWD changes restored via `t.Cleanup(func() { _ = os.Chdir(orig) })`

**Per-Package Fixture Helpers:**
- `pkg/worktree/fixtures_test.go`: `setupClonedWorktreeSet(t, s, setName, remoteBranches, cloneBranches) (wsDir, remoteURL, error)`
- `pkg/workgroup/helpers_test.go`: `makeWorkspaceWithLocalRepos(s, n)`, `makeWorkspaceWithRemoteRepos(s, n)`
- `pkg/branch/create_test.go`: inline helpers (`setupInfraWorktreeSet`, `setupRemoteWorkspaceWithDefaultBranch`, etc.)

## Coverage

**Requirements:** No enforced coverage threshold

**View Coverage:**
```bash
mage Cover             # generates coverage.out and opens in browser
go tool cover -html=coverage.out  # view existing report
```
Coverage output file: `coverage.out` (in repo root, not committed)

## Test Types

**Unit Tests:**
- Pure function tests without filesystem: `config_test.go` (WorkspaceConfig accessors, `parsePorcelainV1`, `parseBranchLine`, `parseStashCount`)
- In-package (`package foo`) to access unexported functions
- Table-driven with `s.Run(tt.name, ...)` and case structs

**Integration Tests:**
- All command tests — create real git repos, run real git operations, verify filesystem and config state
- No separation into different directories; unit and integration tests are co-located
- Race detector enabled in CI (`-race`); all tests expected to pass under `-race`

**E2E Tests:**
- Not used as a separate layer; command tests in `pkg/<feature>/*_test.go` are effectively E2E through the cobra CLI

## Common Patterns

**Table-Driven Tests:**
```go
// Case struct defined at bottom of file or as package-level function
type statusCase struct {
    name  string
    input string
    dirty bool
}

tests := []struct {
    name  string
    input string
    want  int
}{
    {"empty", "", 0},
    {"one entry", "stash@{0}: WIP on main: abc\n", 1},
}

for _, tt := range tests {
    s.Run(tt.name, func() {
        s.Assert().Equal(tt.want, parseStashCount([]byte(tt.input)))
    })
}
```

**Error Testing:**
```go
// Expect a specific error
_, err := s.ExecuteCmd("repo", "add", notARepo)
s.Require().Error(err)
s.Assert().Contains(err.Error(), "not a git repository")

// Use errors.Is for sentinel errors
_, err := config.Load(cfgPath)
s.Require().Error(err)
s.Assert().ErrorIs(err, os.ErrNotExist)

// Package-level assertion helper for refusal errors
assertSafetyRefusal(s.T(), err, "uncommitted")  // checks err != nil && contains "refusing"
```

**Async / Parallel Testing:**
```go
// Concurrency test pattern using atomic counters
var peak atomic.Int32
var current atomic.Int32
RunFanOut(items, workers, func(_ int) int {
    cur := current.Add(1)
    // CAS loop to track peak
    current.Add(-1)
    return 0
})
assert.LessOrEqual(t, int(peak.Load()), workers)
```

**Workspace Setup Pattern:**
```go
func (s *AddSuite) SetupTest() {
    s.CmdSuite.SetupTest()        // always call super first
    s.wsDir = s.SetupWorkspaceDir()
}
```

**Circular Dependency Workaround:**
- `pkg/output/write_test.go` uses `suite.Run` directly instead of `testutil.RunSuite` to avoid `output → testutil → repo → output` circular import

---

*Testing analysis: 2026-04-01*
