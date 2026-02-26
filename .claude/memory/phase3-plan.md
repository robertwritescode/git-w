# Phase 3: Parallel Execution Engine

**Goal**: Git commands run concurrently across repos with prefixed output.

**Exit criteria**: `git workspace fetch` runs in all repos concurrently with prefixed output.
All Phase 3 `_test.go` files pass `go test -race -count=1 ./...`.

---

## Files to Produce

| File | Dependency |
|---|---|
| `internal/executor/result.go` | none |
| `internal/executor/result_test.go` | result.go |
| `internal/executor/parallel.go` | result.go |
| `internal/executor/parallel_test.go` | parallel.go |
| `cmd/exec.go` | executor package |
| `cmd/git_cmds.go` | executor package |
| `cmd/exec_test.go` | exec.go |
| `cmd/git_cmds_test.go` | git_cmds.go |

---

## Execution Waves (Optimised for Parallel Workers)

```
Wave 1 (parallel):
  Worker A → result.go + result_test.go
  Worker B → git_cmds.go (stubs ok; compiles against executor API contract)

Wave 2 (parallel, after Wave 1):
  Worker A → parallel.go + parallel_test.go
  Worker B → exec.go (uses ExecOptions/ExecResult from result.go)

Wave 3 (parallel, after Wave 2):
  Worker A → exec_test.go
  Worker B → git_cmds_test.go

Wave 4 (sequential):
  go test -race -count=1 ./...
```

---

## Wave 1-A: `internal/executor/result.go` + `result_test.go`

### `result.go`

**Package**: `executor`

```go
// ExecResult holds the outcome of running a git command in one repo.
type ExecResult struct {
    RepoName string
    Stdout   []byte
    Stderr   []byte
    ExitCode int
    Err      error
}
```

Private helpers (all unexported — no godoc comments needed):

- `prefixLines(name string, b []byte) []byte` — prepend `[name] ` to each non-empty line. Split on `\n`, skip blank trailing line, rejoin.
- `combinedOutput(r ExecResult) []byte` — `append(r.Stdout, r.Stderr...)` for display when stdout and stderr need to be shown together.

**Coding standards**:
- No exported helpers beyond the `ExecResult` type itself — callers (`parallel.go`, `cmd/`) use `prefixLines` indirectly via output written to `io.Writer`.
- No inline comments that restate the code. Only add a comment if the behaviour is non-obvious (e.g. why blank trailing lines are skipped).
- Guard clauses: if `len(b) == 0`, return early.

### `result_test.go`

**Package**: `executor` (white-box — needs access to `prefixLines`)

**Suite**: `ResultSuite` → `func TestResult(t *testing.T) { suite.Run(t, new(ResultSuite)) }`

**Tests** (all table-driven where multiple cases exist):

```
TestPrefixLines:
  cases:
    - "empty input"          → input: nil,         want: nil
    - "single line"          → input: "hello\n",   want: "[repo] hello\n"
    - "multi-line"           → input: "a\nb\n",    want: "[repo] a\n[repo] b\n"
    - "no trailing newline"  → input: "hello",     want: "[repo] hello"
    - "blank middle line"    → input: "a\n\nb\n",  want: "[repo] a\n\n[repo] b\n"

TestCombinedOutput:
  cases:
    - "stdout only"      → Stdout: []byte("out"), Stderr: nil
    - "stderr only"      → Stdout: nil,           Stderr: []byte("err")
    - "both"             → Stdout: []byte("o"),   Stderr: []byte("e")
    - "both empty"       → Stdout: nil,           Stderr: nil

TestExecResult_NonZeroExit:
  Verify ExitCode field is preserved on the struct (simple value check, no helper needed).
```

Use `s.Assert().Equal` for non-fatal value checks; `s.Require()` only for setup.

---

## Wave 1-B: `cmd/git_cmds.go` (stub, finalised in Wave 2)

**Package**: `cmd`

Register four cobra commands — `fetch`, `pull`, `push`, `status` (alias `st`) — all backed by the same private helper once `executor` is available. In Wave 1, write the full file with the function bodies calling a helper (`runGitCmd`) that can be wired to `executor.RunParallel` in Wave 2.

### Exported command vars

```go
var fetchCmd  = &cobra.Command{ Use: "fetch [repos...]",  Short: "Run git fetch in repos",  RunE: runFetch  }
var pullCmd   = &cobra.Command{ Use: "pull [repos...]",   Short: "Run git pull in repos",   RunE: runPull   }
var pushCmd   = &cobra.Command{ Use: "push [repos...]",   Short: "Run git push in repos",   RunE: runPush   }
var statusCmd = &cobra.Command{
    Use:     "status [repos...]",
    Aliases: []string{"st"},
    Short:   "Run git status -sb in repos",
    RunE:    runStatus,
}
```

Register all four in `init()` with `rootCmd.AddCommand(...)`.

### Private helpers

```go
// runFetch, runPull, runPush, runStatus all delegate to runGitCmd.
func runFetch(cmd *cobra.Command, args []string) error  { return runGitCmd(cmd, args, "fetch") }
func runPull(cmd *cobra.Command, args []string) error   { return runGitCmd(cmd, args, "pull") }
func runPush(cmd *cobra.Command, args []string) error   { return runGitCmd(cmd, args, "push") }
func runStatus(cmd *cobra.Command, args []string) error { return runGitCmd(cmd, args, "status", "-sb") }

// runGitCmd resolves repos, builds ExecOptions, calls executor.RunParallel,
// then streams prefixed output and returns a non-nil error if any repo failed.
func runGitCmd(cmd *cobra.Command, args []string, gitArgs ...string) error
```

**Argument semantics**: `args` is the optional `[repos...]` filter (repo names from config).
`gitArgs` is the fixed git subcommand. Repo-name args and git-args are never mixed for these commands.

**Output**: write each result's prefixed output to `cmd.OutOrStdout()`. Return a summary error listing all failures if any `ExecResult.ExitCode != 0` or `ExecResult.Err != nil`.

**Coding standards**:
- `runGitCmd` will exceed ~15 lines due to the resolve → execute → print → error-collect pipeline; extract `resolveTargetRepos`, `printResults`, `collectErrors` as private helpers.
- No comments restating what each line does.

---

## Wave 2-A: `internal/executor/parallel.go` + `parallel_test.go`

### `parallel.go`

**Package**: `executor`

```go
// ExecOptions configures a RunParallel invocation.
type ExecOptions struct {
    MaxConcurrency int           // 0 → runtime.NumCPU()
    Timeout        time.Duration // 0 → no timeout
    Async          bool          // false → single serial run, stdin passthrough
}

// RunParallel executes git args in each repo concurrently.
// Single-repo or Async=false: stdin passes through; output not prefixed.
// Multi-repo Async=true: stdin suppressed (os.DevNull); output prefixed "[name]".
func RunParallel(repos []repo.Repo, args []string, opts ExecOptions) []ExecResult
```

#### Implementation notes

**`RunParallel` logic** (high-level steps — keep the body ≤20 lines by extracting helpers):

```
1. resolve concurrency cap     → maxWorkers(opts)
2. build context with timeout  → buildContext(opts) returns (ctx, cancel)
3. dispatch: single-repo or async=false path → runSerial
           : multi-repo async path            → runConcurrent
4. return results
```

**`maxWorkers(opts ExecOptions) int`** — returns `opts.MaxConcurrency` if > 0, else `runtime.NumCPU()`.

**`buildContext(opts ExecOptions) (context.Context, context.CancelFunc)`** — returns `context.WithTimeout` if `opts.Timeout > 0`, else `context.WithCancel(context.Background())`.

**`runSerial(ctx context.Context, r repo.Repo, args []string) ExecResult`** — runs one repo with stdin wired to `os.Stdin`, stdout/stderr passed through to the process directly (no buffering). Used when `len(repos) == 1 || !opts.Async`.

**`runConcurrent(ctx context.Context, repos []repo.Repo, args []string, workers int) []ExecResult`** — goroutine pool using `golang.org/x/sync/errgroup` + a semaphore channel of size `workers`. Collects all results into a slice (same length as `repos`, in same order). Each goroutine calls `runOne`.

**`runOne(ctx context.Context, r repo.Repo, args []string) ExecResult`** — builds `exec.CommandContext(ctx, "git", args...)`, sets `Cmd.Dir = r.AbsPath`, appends `r.Flags` before args, captures stdout+stderr separately, populates `ExecResult`. Stdin is `os.DevNull`.

**Result ordering**: pre-allocate `results := make([]ExecResult, len(repos))` and write by index — avoids a mutex on the slice.

**Semaphore pattern** (preferred over `errgroup` limit since we want all results even on error):

```go
sem := make(chan struct{}, workers)
var wg sync.WaitGroup
for i, r := range repos {
    wg.Add(1)
    sem <- struct{}{}
    go func(idx int, r repo.Repo) {
        defer wg.Done()
        defer func() { <-sem }()
        results[idx] = runOne(ctx, r, args)
    }(i, r)
}
wg.Wait()
```

This pattern (not `errgroup`) ensures we always collect all results and never bail early on a non-zero exit.

**Coding standards**:
- `RunParallel` body: delegates entirely to helpers; max ~10 lines.
- `runConcurrent`: the goroutine loop above is ~10 lines; acceptable.
- Guard clause in `runSerial`: check `ctx.Err()` before running.
- No comments restating variable assignments.

### `parallel_test.go`

**Package**: `executor` (white-box)

**Suite**: `ParallelSuite` → `func TestParallel(t *testing.T) { suite.Run(t, new(ParallelSuite)) }`

**`SetupTest`**: create 3 temp git repos using `testutil.MakeGitRepo(s.T(), "")` (no remote), store as `s.repos []repo.Repo`.

**Tests**:

```
TestRunParallel_MultiRepo_CollectsAll:
  Run "echo", "hello" across 3 repos.
  Assert len(results) == 3 and all ExitCode == 0.

TestRunParallel_MultiRepo_PrefixesOutput:
  Run "echo", "hi" across 3 repos with Async=true.
  For each result, assert strings.Contains(string(result.Stdout), "["+result.RepoName+"]").

TestRunParallel_SingleRepo_NoPrefix:
  Pass a single-element repos slice.
  Assert result.Stdout does NOT contain "[".

TestRunParallel_ConcurrencyLimit:
  Use an atomic int32 counter: start a script that increments, sleeps 50ms, decrements.
  Run with MaxConcurrency=2 across 4 repos.
  Assert peak concurrent count never exceeds 2.
  (Use /bin/sh -c "..." via exec, or a mock command — see note below.)

TestRunParallel_Timeout:
  Set Timeout=100ms, run "sleep 5" across 2 repos.
  Assert all results have non-nil Err or non-zero ExitCode within 500ms.

TestRunParallel_NonZeroExit:
  Run "git", "invalid-subcommand" across 2 repos.
  Assert all results have ExitCode != 0.
```

**Concurrency limit test note**: The simplest approach is to use `sh -c` with a shared temp file as a counter (increment, sleep, decrement). Alternatively, since this tests the semaphore cap rather than actual git behaviour, use a helper that spawns a known-slow process. Use `testutil.MakeGitRepo` repos as `cmd.Dir` targets.

**Async flag on multi-repo tests**: set `opts.Async = true`.

**Known pitfall**: `SetupTest` runs once per method, not per `s.Run` sub-test. Each test method that needs isolation must create its own repos inside the method body, not rely on `s.repos`. Use `s.repos` only for tests that don't mutate state.

---

## Wave 2-B: `cmd/exec.go`

**Package**: `cmd`

```go
var execCmd = &cobra.Command{
    Use:   "exec [repos...] -- <git-args>",
    Short: "Execute an arbitrary git command across repos",
    Long: `Runs any git command in each registered repo concurrently.
Repo names before '--' filter targets; everything after '--' is passed to git.`,
    RunE: runExec,
    DisableFlagParsing: false,
}
```

Register in `init()` with `rootCmd.AddCommand(execCmd)`.

### Argument parsing

`exec [repos...] -- <git-args>` — cobra does not natively split on `--`. Parse `args` manually:

```go
func splitExecArgs(args []string) (repoNames, gitArgs []string) {
    for i, a := range args {
        if a == "--" {
            return args[:i], args[i+1:]
        }
    }
    return nil, args  // no "--": treat everything as git args, no repo filter
}
```

### `runExec` body (≤20 lines, extracted helpers):

```go
func runExec(cmd *cobra.Command, args []string) error {
    repoNames, gitArgs := splitExecArgs(args)
    if len(gitArgs) == 0 {
        return fmt.Errorf("no git command specified; use: exec [repos...] -- <git-args>")
    }
    cfg, cfgPath, err := loadConfig()
    if err != nil { return err }
    repos, err := filterRepos(cfg, cfgPath, repoNames)
    if err != nil { return err }
    opts := executor.ExecOptions{Async: len(repos) > 1}
    results := executor.RunParallel(repos, gitArgs, opts)
    writeResults(cmd.OutOrStdout(), results)
    return execErrors(results)
}
```

Private helpers:

- `filterRepos(cfg, cfgPath, names []string) ([]repo.Repo, error)` — if `names` empty, return all repos; else look up each name and return an error for any that don't exist.
- `writeResults(w io.Writer, results []executor.ExecResult)` — iterate results, write prefixed stdout then prefixed stderr to `w`.
- `execErrors(results []executor.ExecResult) error` — iterate all results, collect failures (non-zero ExitCode or non-nil Err); if none, return nil; otherwise return a formatted error: `"N of M repos failed:\n  [repo]: <message>"` where `<message>` is `Err.Error()` or `"exit <code>"`.

**Coding standards**:
- `filterRepos`: guard clause for empty names.
- `execErrors`: iterate all results before returning; no early exit — every repo's outcome is inspected.
- No comments on obvious lines.

---

## Wave 3-A: `cmd/exec_test.go`

**Package**: `cmd_test`

**Suite**: `ExecSuite` → `func TestExec(t *testing.T) { suite.Run(t, new(ExecSuite)) }`

Each sub-test creates its own workspace + repos inside the `s.Run` closure (do not rely on `SetupTest`).

```
TestExec_RunsInAllRepos:
  MakeGitRepo x2, MakeWorkspace referencing both.
  runCmd(wsDir, "exec", "--", "git", "status") → exit 0, output contains both repo names prefixed.

TestExec_FilterByRepoName:
  MakeGitRepo x2, exec with one repo name before "--".
  Output contains only that repo's prefix.

TestExec_UnknownRepo_Error:
  exec "nonexistent", "--", "status" → non-zero exit, stderr contains "nonexistent".

TestExec_MissingDashDash_TreatsAllAsGitArgs:
  exec "status" (no "--") → runs "git status" in all repos (no repo-name filter).

TestExec_NonZeroGitExit_PropagatesError:
  exec "--", "invalid-subcommand" → runCmd returns non-zero exit code.
```

Use the `runCmd` helper pattern established in Phase 2 (`cmd/info_test.go`):
```go
func runCmd(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int)
```
Reuse or re-export this helper; do not duplicate.

---

## Wave 3-B: `cmd/git_cmds_test.go`

**Package**: `cmd_test`

**Suite**: `GitCmdsSuite` → `func TestGitCmds(t *testing.T) { suite.Run(t, new(GitCmdsSuite)) }`

Each sub-test is table-driven where multiple commands share the same shape:

```
TestGitCmd_RunsInAllRepos (table-driven):
  cases: fetch, pull, status
  For each: MakeGitRepo with a remote (testutil.MakeGitRepo(s.T(), "file:///...")),
  runCmd(wsDir, cmdName) → exit 0, output contains repo name.

TestPush_RequiresRemote:
  MakeGitRepo with no remote, run push → non-zero exit or specific error message.

TestStatus_AliasWorks:
  runCmd(wsDir, "st") → same output as "status".
```

**Note on fetch/pull/push**: these require a reachable remote. Use a bare git repo as the remote:
```go
// inside test — create a bare repo to serve as remote
remoteDir := s.T().TempDir()
gitInitBare(s.T(), remoteDir)          // git init --bare
repoDir := testutil.MakeGitRepo(s.T(), "file://"+remoteDir)
```

Add `gitInitBare(t testing.TB, dir string)` to `internal/testutil/helpers.go` if not already present (shared helper used by 2+ test files → belongs in testutil).

---

## Wave 4: Verify

```
go test -race -count=1 ./...
```

All tests must pass. Fix any data races surfaced by `-race`. Common sources:
- Results slice written by goroutines without proper index isolation (use pre-allocated slice + index, not append).
- Shared `cmd` flag vars in cobra integration tests (see known pitfall in `testing.md`).

---

## Coding Standards Checklist (apply to every file)

Before marking any file complete:

- [ ] No function exceeds ~20 lines without extraction
- [ ] No inline comments restating what the code does
- [ ] No duplicated logic that could be a shared private helper
- [ ] Exported symbols have godoc; unexported helpers do not need comments
- [ ] Test files use `testify/suite` — not bare `func TestXxx(t *testing.T)` (except the `suite.Run` entry point)
- [ ] Every multi-case test uses table-driven `[]struct{ name, ... }` + `s.Run(tc.name, ...)`
- [ ] Every test file is evaluated to determine if consolidating tests to larger table-driven tests is reasonable and possible.
- [ ] `s.Require()` for setup/fatal, `s.Assert()` for value checks
- [ ] `SetupTest` state is NOT shared across `s.Run` sub-tests that need isolation

---

## Resolved Decisions

1. **`execErrors` behaviour**: collect and format all failures — `"N of M repos failed:\n  [repo]: <message>"`. Never returns on first error; every result is inspected. Implemented in the `execErrors` private helper in `cmd/exec.go` and shared via `runGitCmd` in `cmd/git_cmds.go`.

2. **Output ordering**: `runConcurrent` collects by index (preserves config order). Output is written after all goroutines finish (not streamed) to avoid interleaved output. Streaming is deferred to Phase 5 if desired.

3. **`git_cmds.go` stdin**: fetch/pull/push pass `os.DevNull` as stdin when Async=true (multi-repo). For a single-repo target, stdin passes through. Matches `exec.go` semantics.

---

## Phase 3 Implementation Notes (post-implementation)

### Deviations from plan

1. **`RunParallel` serial condition**: Changed from `len(repos)==1 || !opts.Async` to just `!opts.Async`. This allows `exec` (which always sets `Async:true`) to capture output even when filtered to a single repo.

2. **`execCommand` variable name**: Named `execCommand` (not `execCmd`) to avoid collision with the test helper function `execCmd(t, args...)` in `cmd/init_test.go`. Both are in `package cmd`.

3. **`DisableFlagParsing: true` on execCommand**: Cobra strips `--` from positional args during normal flag parsing. Setting DisableFlagParsing preserves `--` in args so `splitExecArgs` can split on it correctly.

4. **Timeout test**: Used `exec sleep 10` (not `sleep 10`) in fake git so SIGKILL to the shell process terminates sleep via exec-replacement, not as orphaned child.

5. **`testutil.MakeGitRepo` signature**: Changed from `(t *testing.T, dir string)` to `(t testing.TB, remoteURL string)`. All callers updated. Added `GitInitBare(t testing.TB, dir string)`.

6. **fetch test assertion**: `git fetch` with nothing to fetch produces NO output. Test for fetch only asserts exit 0, not output content.