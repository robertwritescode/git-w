# Phase 5 Plan: Advanced Features

## Overview

**Goal**: Clone, restore, recursive add, and shell completion.

**4 independent parallel streams** — no cross-stream dependencies. Assign one worker per stream.

**Exit criterion**: `go test -race -count=1 ./...` passes.

**Coding standards enforced throughout**:
- Every test file uses testify/suite (`suite.Suite` embed, `s.Require()` / `s.Assert()`)
- Every multi-case test is table-driven: `[]struct{ name, ... }` + `s.Run(tc.name, func() { ... })`
- Functions ≤ ~20 lines; extract private helpers for complex logic
- No comments that restate what code does
- No duplicated logic across 2+ functions

---

## Context: What Already Exists

- `ensureGitignore(dir, entry string) error` — in `cmd/init.go`, same package; callable from all new cmd files
- `addRepoToGroup(cfg, group, name string)` — in `cmd/add.go`, same package
- `detectRemoteURL(repoPath string) string` — in `cmd/add.go`, same package
- `isGitRepo(path string) bool` — in `cmd/add.go`, same package
- `computeRelPath(cfgPath, absPath string) (string, error)` — in `cmd/add.go`
- `resolveRepoName(cfg, absPath) (string, error)` — in `cmd/add.go`
- `autoGitignoreEnabled(cfg) bool` — in `cmd/root.go`
- `config.ConfigDir(cfgPath string) string` — in `internal/config/`
- `testutil.GitInitBare(t, dir)` — in `internal/testutil/helpers.go`
- Per-repo `Flags []string` in `RepoConfig` and wired into executor — **already complete**
- `URL string` field in `RepoConfig` — **already present**
- No `internal/gitignore/` package needed — `ensureGitignore` in cmd package is sufficient

---

## Stream A: `clone` Command

**Files**: `cmd/clone.go` (new), `cmd/clone_test.go` (new)

### A1 — `cmd/clone.go`

```
git workspace clone <url> [<path>] [-g <group>]
```

**Cobra command definition:**
```go
var cloneGroup string

var cloneCmd = &cobra.Command{
    Use:   "clone <url> [<path>]",
    Short: "Clone a remote repo and register it in the workspace",
    Args:  cobra.RangeArgs(1, 2),
    RunE:  runClone,
}

func init() {
    rootCmd.AddCommand(cloneCmd)
    cloneCmd.Flags().StringVarP(&cloneGroup, "group", "g", "", "add cloned repo to this group")
}
```

**Functions** (keep each ≤ 20 lines):

```go
func runClone(cmd *cobra.Command, args []string) error
// 1. loadConfig
// 2. resolve destPath (args[1] or deriveClonePath(args[0]))
// 3. resolveRepoName — error if already registered
// 4. gitClone(url, destPath)
// 5. computeRelPath + register in cfg.Repos
// 6. autoGitignoreEnabled → ensureGitignore (warn on error)
// 7. if cloneGroup != "" → addRepoToGroup
// 8. config.Save
// 9. print "Cloned <name> (<relPath>)"

func deriveClonePath(rawURL string) string
// uses path.Base(rawURL), strips trailing ".git"
// import "path" (not filepath) — URLs use forward slashes

func gitClone(url, destPath string) error
// exec.Command("git", "clone", url, destPath)
// CombinedOutput; wrap error with output
// NOTE: also called by cmd/restore.go (same package)
```

**Key rules**:
- `destPath` is resolved: if `args[1]` given, use `filepath.Abs(args[1])`; else `filepath.Join(cfgDir, deriveClonePath(args[0]))`
- Name derived via `filepath.Base(destPath)` — identical to `add` behavior
- `resolveRepoName` already errors if name is taken; reuse it
- URL stored in `RepoConfig.URL`

---

### A2 — `cmd/clone_test.go`

**Suite**: `CloneSuite` embeds `WorkspaceSuite` (defined in add_test.go).

**Entry point**: `func TestCloneSuite(t *testing.T) { suite.Run(t, new(CloneSuite)) }`

**Test setup helper** (inline in sub-tests, not in SetupTest — see testing.md pitfall):
```go
// createBareRepo creates a local bare git repo; returns its absolute path and file:// URL.
func createBareRepo(t *testing.T) (dir, url string)
```

**Table-driven: `TestClone`**
```go
tests := []struct {
    name      string
    args      []string  // args after "clone" (url + optional path)
    wantGroup string
}{
    {name: "derives path from URL",    args: []string{"file://<bareDir>"}},
    {name: "uses explicit path",       args: []string{"file://<bareDir>", "myrepo"}},
    {name: "strips .git from URL",     args: []string{"file://<bareDir.git suffix>"}},
    {name: "adds to group",            args: []string{"file://<bareDir>", "-g", "web"}, wantGroup: "web"},
    {name: "updates gitignore",        args: []string{"file://<bareDir>"}},
}
```
Each sub-test:
1. Creates fresh workspace + bare repo in `s.T().TempDir()` (sub-test scope)
2. Runs `execCmd(s.T(), append([]string{"clone"}, tt.args...)...)`
3. Asserts: config has repo, URL matches, path exists as git repo, group membership if `tt.wantGroup != ""`
4. Asserts gitignore contains relPath for the "updates gitignore" case

**Single-case tests**:
```go
func (s *CloneSuite) TestCloneErrorAlreadyRegistered()
// clone same bare repo twice → second clone returns error containing "already registered"

func (s *CloneSuite) TestCloneErrorNoURL()
// execCmd("clone") → error (cobra validates RangeArgs(1,2))
```

**Pitfalls**:
- Each sub-test creates its own workspace dir + bare repo (don't share state between sub-tests)
- Bare repo: use `testutil.GitInitBare(s.T(), bareDir)`, URL = `"file://" + bareDir`
- `changeToDir(s.T(), wsDir)` at start of each sub-test

---

## Stream B: `restore` Command

**Files**: `cmd/restore.go` (new), `cmd/restore_test.go` (new)

### B1 — `cmd/restore.go`

```
git workspace restore
```

**Cobra command definition:**
```go
var restoreCmd = &cobra.Command{
    Use:   "restore",
    Short: "Materialize all repos: clone missing, pull existing",
    Args:  cobra.NoArgs,
    RunE:  runRestore,
}

func init() {
    rootCmd.AddCommand(restoreCmd)
}
```

**Imports needed**: `"context"`, `"sync"`, `"golang.org/x/sync/errgroup"`

**Functions** (keep each ≤ 20 lines):

```go
func runRestore(cmd *cobra.Command, args []string) error
// 1. loadConfig
// 2. restoreAll(cmd, cfg, cfgPath)

func restoreAll(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string) error
// goroutine per repo using errgroup; never return non-nil from goroutine (collect failures)
// mutex-protected: print status per repo, append to failures slice
// at end: if failures → return multi-line error

func restoreRepo(ctx context.Context, name string, rc config.RepoConfig, absPath string) (msg string, err error)
// if isGitRepo(absPath) → restorePull(ctx, absPath) → msg = trimmed output
// elif rc.URL == "" → return "skipped: no URL configured", nil
// else → gitClone(rc.URL, absPath) → msg = "cloned"

func restorePull(ctx context.Context, absPath string) (string, error)
// exec.CommandContext(ctx, "git", "-C", absPath, "pull")
// returns trimmed stdout on success
```

**Output format** (per-repo, prefixed):
```
[frontend] cloned
[backend] Already up to date.
[infra] skipped: no URL configured
```
Errors go to stderr: `[broken] error: git clone: exit status 128`

**gitClone**: defined in `cmd/clone.go` (Stream A). `restore.go` calls it directly (same package). Stream B worker must reference Stream A's signature: `gitClone(url, destPath string) error`.

**absPath computation** in `restoreAll`:
```go
absPath := filepath.Join(config.ConfigDir(cfgPath), rc.Path)
```

**gitignore** applied in `restoreAll` after `restoreRepo` returns (regardless of clone/pull), on success only:
```go
if err == nil && autoGitignoreEnabled(cfg) {
    if giErr := ensureGitignore(config.ConfigDir(cfgPath), rc.Path); giErr != nil {
        fmt.Fprintf(cmd.ErrOrStderr(), "[%s] warning: .gitignore: %v\n", name, giErr)
    }
}
```

---

### B2 — `cmd/restore_test.go`

**Suite**: `RestoreSuite` embeds `WorkspaceSuite`.

**Entry point**: `func TestRestoreSuite(t *testing.T) { suite.Run(t, new(RestoreSuite)) }`

**Shared sub-test helper** (reuse pattern from add_test.go):
```go
// Each sub-test sets up its own wsDir + bare repo in s.T().TempDir()
```

**Table-driven: `TestRestore`**
```go
tests := []struct {
    name           string
    repoExists     bool   // pre-create the target path as a cloned repo
    hasURL         bool
    wantOutput     string // substring to check in stdout
    wantErr        bool
}{
    {name: "clones missing repo",   repoExists: false, hasURL: true,  wantOutput: "cloned"},
    {name: "pulls existing repo",   repoExists: true,  hasURL: true,  wantOutput: "up to date"},
    {name: "skips no-URL repo",     repoExists: false, hasURL: false, wantOutput: "skipped"},
    {name: "gitignore updated",     repoExists: false, hasURL: true},
}
```

Each sub-test:
1. `wsDir := s.T().TempDir()` + write `.gitworkspace` with `changeToDir`
2. For `hasURL=true`: `testutil.GitInitBare(s.T(), bareDir)`, `url = "file://" + bareDir`
3. For `repoExists=true`: pre-clone the bare repo into `destPath` using `exec.Command("git", "clone", url, destPath)`
4. Write config TOML with `[repos.myrepo]` path + URL fields as appropriate
5. Run `execCmd(s.T(), "restore")`
6. Assert output contains `tt.wantOutput` and `!tt.wantErr`
7. For "clones missing repo": assert `isGitRepo(filepath.Join(wsDir, "myrepo"))`
8. For "gitignore updated": assert `.gitignore` contains `"myrepo"`

**Single-case tests**:
```go
func (s *RestoreSuite) TestRestoreIdempotent()
// clone, then restore again → no error, path still a valid git repo

func (s *RestoreSuite) TestRestoreEmpty()
// workspace with no repos → no error, no output
```

---

## Stream C: `add -r` Recursive Flag

**Files**: modify `cmd/add.go`, extend `cmd/add_test.go`

### C1 — Modify `cmd/add.go`

**New package-level var**:
```go
var addRecursiveDir string
```

**In `init()`** — add after existing flag:
```go
addCmd.Flags().StringVarP(&addRecursiveDir, "recursive", "r", "", "recursively add all git repos in <dir>; defaults to CWD if -r given without value")
addCmd.Flag("recursive").NoOptDefVal = "."  // -r alone → "."
```

**Change `Args`** from `cobra.ExactArgs(1)` to:
```go
Args: func(cmd *cobra.Command, args []string) error {
    if cmd.Flags().Changed("recursive") {
        return cobra.NoArgs(cmd, args)
    }
    return cobra.ExactArgs(1)(cmd, args)
},
```

**Modify `runAdd`** — dispatch at top:
```go
func runAdd(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil {
        return err
    }
    if addRecursiveDir != "" {
        return runAddRecursive(cmd, cfg, cfgPath)
    }
    return runAddSingle(cmd, cfg, cfgPath, args[0])
}
```

Rename current `runAdd` body to `runAddSingle(cmd, cfg, cfgPath, pathArg string) error`.

**New functions**:

```go
func runAddRecursive(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string) error
// 1. resolve walkDir: if addRecursiveDir == "." → os.Getwd(); else filepath.Abs(addRecursiveDir)
// 2. walkGitRepos(walkDir) → []string of abs paths
// 3. for each found path: call registerRepo (skip if already registered, no error)
// 4. config.Save; report how many added

func walkGitRepos(root string) ([]string, error)
// filepath.WalkDir; when entry is a dir and filepath.Join(path, ".git") exists:
//   append path; return fs.SkipDir to prevent nesting
// skip hidden dirs (name starts with ".")
// return sorted list of abs paths

func autoGroupName(repoAbsPath, walkRoot string) string
// rel = must be computed: filepath.Rel(walkRoot, repoAbsPath)
// parts = strings.Split(rel, string(os.PathSeparator))
// if len(parts) <= 1 → return "" (repo is directly in walkRoot, no group)
// return parts[0]  (immediate subdirectory name)

func registerRepo(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath, absPath, groupName string) (added bool, err error)
// 1. if !isGitRepo(absPath) → return false, nil (skip silently)
// 2. name = filepath.Base(absPath)
// 3. if _, exists := cfg.Repos[name]; exists → return false, nil (skip, no error)
// 4. relPath, err = computeRelPath(cfgPath, absPath)
// 5. cfg.Repos[name] = RepoConfig{Path: relPath, URL: detectRemoteURL(absPath)}
// 6. if autoGitignoreEnabled(cfg) → ensureGitignore (warn)
// 7. if groupName != "" → addRepoToGroup(cfg, groupName, name)
// 8. return true, nil
```

**DRY**: `runAddSingle` now calls `registerRepo` internally (or keeps the existing inline logic — whichever keeps it under 20 lines). Evaluate during implementation.

---

### C2 — Extend `cmd/add_test.go`

Add a new table-driven test to `AddSuite`:

```go
func (s *AddSuite) TestRecursiveAdd() {
    tests := []struct {
        name        string
        setup       func(t *testing.T, wsDir string) (dir string)  // returns walkDir
        args        []string  // args to pass after "add"
        wantNames   []string  // repo names expected in config
        wantGroups  map[string][]string  // groupName → repos expected
        wantSkipped []string  // repos not expected (already registered or not git)
    }{
        {
            name: "finds single repo",
            // create one git repo at walkDir/myrepo
            wantNames: []string{"myrepo"},
        },
        {
            name: "auto-groups repos by parent dir",
            // create: walkDir/apps/frontend, walkDir/apps/backend
            wantNames:  []string{"frontend", "backend"},
            wantGroups: map[string][]string{"apps": {"frontend", "backend"}},
        },
        {
            name: "skips non-git directories",
            // create: walkDir/notarepo (no .git), walkDir/realrepo (.git)
            wantNames:   []string{"realrepo"},
            wantSkipped: []string{"notarepo"},
        },
        {
            name: "skips already-registered repos without error",
            // pre-register "myrepo" in config, then recursive-add same dir
            wantNames: []string{"myrepo"},  // still in config, no duplicate, no error
        },
        {
            name: "non-nesting: inner git repo not found",
            // create: outer/.git and outer/inner/.git
            // walk outer → only "outer" registered, "inner" skipped
            wantNames:   []string{"outer"},
            wantSkipped: []string{"inner"},
        },
        {
            name: "uses CWD when -r given without value",
            // create git repo in wsDir itself
            // args: []string{"-r"}  (no dir arg; NoOptDefVal = ".")
            // change into a subdirectory that has a git repo
        },
    }
    for _, tt := range tests {
        s.Run(tt.name, func() {
            wsDir := s.T().TempDir()
            s.Require().NoError(os.WriteFile(
                filepath.Join(wsDir, ".gitworkspace"),
                []byte("[workspace]\nname = \"testws\"\n"), 0o644,
            ))
            changeToDir(s.T(), wsDir)
            // ... setup per test case ...
            _, err := execCmd(s.T(), tt.args...)
            s.Require().NoError(err)
            cfg, err := config.Load(filepath.Join(wsDir, ".gitworkspace"))
            s.Require().NoError(err)
            for _, name := range tt.wantNames {
                s.Assert().Contains(cfg.Repos, name)
            }
            for group, repos := range tt.wantGroups {
                for _, repo := range repos {
                    s.Assert().Contains(cfg.Groups[group].Repos, repo)
                }
            }
            for _, name := range tt.wantSkipped {
                s.Assert().NotContains(cfg.Repos, name)
            }
        })
    }
}
```

**Note on pflag pitfall**: `addRecursiveDir` persists between `execCmd` calls (same package-global). Each sub-test must be self-contained and not rely on flag state from prior sub-tests. Use fresh workspaces.

---

## Stream D: Shell Completion

**Files**: `cmd/completion.go` (new), `cmd/completion_test.go` (new)

### D1 — `cmd/completion.go`

```go
var completionCmd = &cobra.Command{
    Use:       "completion [bash|zsh|fish|powershell]",
    Short:     "Generate a shell completion script",
    ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
    Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
    RunE:      runCompletion,
}

func init() {
    rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) error {
    switch args[0] {
    case "bash":
        return rootCmd.GenBashCompletion(cmd.OutOrStdout())
    case "zsh":
        return rootCmd.GenZshCompletion(cmd.OutOrStdout())
    case "fish":
        return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
    case "powershell":
        return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
    default:
        return fmt.Errorf("unsupported shell: %s", args[0])
    }
}
```

**No Long description needed** — short is clear.

---

### D2 — `cmd/completion_test.go`

**Suite**: `CompletionSuite` embeds `WorkspaceSuite`.

**Entry point**: `func TestCompletionSuite(t *testing.T) { suite.Run(t, new(CompletionSuite)) }`

**Table-driven: `TestCompletion`**
```go
func (s *CompletionSuite) TestCompletion() {
    tests := []struct {
        name  string
        shell string
    }{
        {name: "bash",       shell: "bash"},
        {name: "zsh",        shell: "zsh"},
        {name: "fish",       shell: "fish"},
        {name: "powershell", shell: "powershell"},
    }
    for _, tt := range tests {
        s.Run(tt.name, func() {
            out, err := execCmd(s.T(), "completion", tt.shell)
            s.Require().NoError(err)
            s.Assert().NotEmpty(out)
        })
    }
}
```

**Single-case test**:
```go
func (s *CompletionSuite) TestCompletionErrorInvalidShell()
// execCmd("completion", "fish-sauce") → error
```

---

## Parallel Execution Summary

| Stream | Worker | Files | Depends on |
|--------|--------|-------|------------|
| A | Worker 1 | `cmd/clone.go`, `cmd/clone_test.go` | nothing (uses existing helpers) |
| B | Worker 2 | `cmd/restore.go`, `cmd/restore_test.go` | `gitClone` from Stream A (same package) |
| C | Worker 3 | modify `cmd/add.go`, extend `cmd/add_test.go` | nothing |
| D | Worker 4 | `cmd/completion.go`, `cmd/completion_test.go` | nothing |

**Coordination point**: Stream B's `restore.go` calls `gitClone(url, destPath string) error` defined in Stream A's `clone.go`. Workers A and B agree on this signature before starting. Stream B worker writes the call; both workers finish independently.

If workers A and B must not block on each other: Stream B can define a local `cloneToPath(url, dest string) error` function and then in the final integration step, deduplicate by having one call the other.

---

## Final Gate (after all streams complete)

1. `go test -race -count=1 ./...` — all tests must pass
2. Manually verify:
   - `git workspace clone <url>` clones and registers
   - `git workspace restore` clones missing, pulls existing
   - `git workspace add -r` finds repos recursively
   - `git workspace completion bash | head -5` outputs completion script

---

## Self-Check Before Marking Each File Done

- [ ] No function exceeds ~20 lines without a clear reason
- [ ] No inline comments that restate what the code does
- [ ] No duplicated logic that could be a shared private helper
- [ ] Exported symbols have godoc; unexported helpers do not
- [ ] Test file uses `testify/suite` — not bare `func TestXxx(t *testing.T)`
- [ ] Every multi-case test uses table-driven `[]struct{ name, ... }` + `s.Run(tc.name, ...)`
- [ ] Each table-driven sub-test sets up its own isolated state (does not rely on `SetupTest`)
- [ ] Flag vars are not shared between sub-tests (fresh workspace per sub-test)
