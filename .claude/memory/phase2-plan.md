# Phase 2 Plan: Status Detection + `info` Display

**Goal**: `git workspace info` (alias `ll`) shows a formatted, color-coded status table for all repos (or a group).

**Coding standards apply throughout**: testify/suite + table-driven tests required; no inline comments restating what code does; functions ≤ ~20 lines; DRY; guard clauses over nesting; exported symbols get godoc.

---

## Dependency Graph

```
Wave 1 (parallel)
  ├── A: internal/repo/repo.go + repo_test.go
  └── B: internal/display/colors.go + colors_test.go

Wave 2 (after A)
  └── C: internal/repo/status.go + status_test.go

Wave 3 (after B + C)
  └── D: internal/display/table.go + table_test.go

Wave 4 (after D)
  └── E: cmd/info.go + info_test.go

Wave 5 (after all)
  └── F: go test -race -count=1 ./... passes
```

---

## Wave 1 — Task A: `internal/repo/repo.go` + `repo_test.go`

### File: `internal/repo/repo.go`

**Package**: `package repo`

**Imports**: `path/filepath`, `github.com/robertwritescode/git-workspace/internal/config`

**Types and exported functions**:

```go
// Repo is a resolved, ready-to-use repository with an absolute path.
type Repo struct {
    Name    string
    AbsPath string
    Flags   []string
}

// FromConfig returns a slice of Repos resolved from cfg, using cfgPath as the
// base for relative path resolution. Repos are returned in sorted name order.
func FromConfig(cfg *config.WorkspaceConfig, cfgPath string) []Repo

// IsGitRepo reports whether path is a git repository (contains a .git entry).
func IsGitRepo(path string) bool
```

**Implementation notes**:
- `FromConfig`: collect keys, sort them (deterministic output), then for each call `absPath(config.ConfigDir(cfgPath), rc.Path)` to resolve. Return `[]Repo`.
- `absPath` (private): `filepath.Join(root, relPath)` — simple join; no need to call `filepath.Abs` since root is already absolute (comes from `ConfigDir` which calls `filepath.Dir` on the already-abs config path).
- `IsGitRepo`: `os.Open(filepath.Join(path, ".git"))` — same pattern as `isGitRepo` in `cmd/add.go`. **Note**: this is the exported package-level version; the private `isGitRepo` in `cmd/add.go` can stay since it's in a different package and already in use. Do not modify `cmd/add.go`.
- `FromConfig` must not exceed ~20 lines; extract `sortedKeys(m map[string]config.RepoConfig) []string` if needed.

### File: `internal/repo/repo_test.go`

**Package**: `package repo` (white-box — tests `absPath` indirectly via `FromConfig`)

**Suite**: `RepoSuite` embedding `suite.Suite`

**Test cases for `FromConfig`** (table-driven):
- `"empty config"` — no repos → empty slice
- `"single repo"` — one repo, verify `Name`, `AbsPath = configDir + "/" + relPath`, `Flags` nil/empty
- `"multiple repos"` — three repos, verify results are sorted by name
- `"with flags"` — repo with `Flags: ["--bare"]`, verify propagated

**Test cases for `IsGitRepo`** (table-driven):
- `"valid git repo"` — `testutil.MakeGitRepo(s.T(), s.T().TempDir())` → true
- `"plain directory"` — bare `t.TempDir()` with no `.git` → false
- `"nonexistent path"` → false

---

## Wave 1 — Task B: `internal/display/colors.go` + `colors_test.go`

### File: `internal/display/colors.go`

**Package**: `package display`

**Imports**: `regexp`, `github.com/fatih/color`

**Exported color variables** (package-level; `*color.Color` values):
```go
var (
    ColorInSync   = color.New(color.FgGreen)
    ColorAhead    = color.New(color.FgHiMagenta)
    ColorBehind   = color.New(color.FgYellow)
    ColorDiverged = color.New(color.FgRed)
    ColorNoRemote = color.New(color.FgWhite)
)
```

**Private helper**:
```go
// ansiEscape matches ANSI CSI escape sequences used by fatih/color.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// visualWidth returns the visible character width of s, ignoring ANSI codes.
func visualWidth(s string) int
```

`visualWidth`: strip `ansiEscape` from s with `ReplaceAllString(s, "")`, then return `len([]rune(result))` for correct Unicode width handling.

### File: `internal/display/colors_test.go`

**Package**: `package display` (white-box — tests `visualWidth`)

**Suite**: `ColorsSuite` embedding `suite.Suite`

**Test cases for `visualWidth`** (table-driven, required — multiple cases):
- `"plain string"` — `"hello"` → 5
- `"empty string"` — `""` → 0
- `"green colored"` — `ColorInSync.Sprint("main ✓")` → `len("main ✓")` (visual width = 7, including ✓ as 1 rune)
- `"bold red"` — `color.New(color.Bold, color.FgRed).Sprint("x")` → 1
- `"no codes"` — `"abc"` → 3
- `"unicode symbol"` — `"✓"` → 1

**Setup**: `color.NoColor = true` in `SetupTest` and `TeardownTest` restores it, so tests are deterministic regardless of terminal type.

Actually: use `s.T().Setenv` is not available; instead in `SetupTest`:
```go
func (s *ColorsSuite) SetupTest() {
    color.NoColor = true
}
```
This means `Sprint` returns plain strings, making `visualWidth` tests consistent.

---

## Wave 2 — Task C: `internal/repo/status.go` + `status_test.go`

Depends on: Task A (`Repo` type must be defined)

### File: `internal/repo/status.go`

**Package**: `package repo`

**Imports**: `bytes`, `os/exec`, `strings`, `strconv`

**Types**:

```go
// RemoteState describes how the local branch relates to its upstream.
type RemoteState int

const (
    RemoteUnknown RemoteState = iota
    InSync
    LocalAhead
    RemoteAhead
    Diverged
    NoRemote
)

// RepoStatus holds the current state of a repository.
type RepoStatus struct {
    Branch      string
    RemoteState RemoteState
    Dirty       bool   // unstaged changes in working tree
    Staged      bool   // changes staged for commit
    Untracked   bool   // untracked files present
    Stashed     bool   // one or more stash entries exist
    LastCommit  string // subject line of HEAD commit
}
```

**Exported function**:
```go
// GetStatus returns the current status of r by running git subprocesses.
// Returns an error if the directory does not exist or git cannot run.
func GetStatus(r Repo) (RepoStatus, error)
```

**Private parse functions** (each tested exhaustively with fixture strings):
```go
func parsePorcelainV1(stdout []byte) (dirty, staged, untracked bool)
func parseBranchLine(line string) (branch string, remote RemoteState)
func parseStashCount(stdout []byte) int
```

**`GetStatus` implementation** — runs three git subprocesses:
1. `git -C <AbsPath> status -b --porcelain` → porcelain v1 with branch line; parse with `parsePorcelainV1` and `parseBranchLine`
2. `git -C <AbsPath> stash list` → count lines with `parseStashCount`
3. `git -C <AbsPath> log -1 --format=%s` → trim and store as `LastCommit`

Function body reads as high-level steps; each subprocess call is extracted to a private helper that returns `([]byte, error)`:
```go
func gitOutput(repoPath string, args ...string) ([]byte, error) {
    out, err := exec.Command("git", append([]string{"-C", repoPath}, args...)...).Output()
    if err != nil {
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) {
            return exitErr.Stderr, err
        }
        return nil, err
    }
    return out, nil
}
```

**`parsePorcelainV1`** — iterates lines; skips the `## ` branch line; for each XY status line:
- XY[0] (index/X): `M`, `A`, `D`, `R`, `C` → staged = true
- XY[1] (worktree/Y): `M`, `D` → dirty = true
- XY == `??` → untracked = true
- Returns early once all three are true (short-circuit optimization)

**`parseBranchLine`** — parses the first line of porcelain v1 output that starts with `## `:
- Input: `## main...origin/main [ahead 2, behind 1]`
- Split on `...`: left part (after `## `) = branch name
- If no `...`: branch is everything after `## `; remote = `NoRemote`
- Right part: check for `ahead` and/or `behind`:
  - both → `Diverged`
  - ahead only → `LocalAhead`
  - behind only → `RemoteAhead`
  - neither → `InSync`
- Edge cases: `## HEAD (no branch)` → branch = "HEAD", remote = `RemoteUnknown`; `## No commits yet on main` → extract name after last space, remote = `NoRemote`

**`parseStashCount`** — count newline-terminated lines in output; empty output → 0.

### File: `internal/repo/status_test.go`

**Package**: `package repo` (white-box — tests unexported parse functions)

**Suite**: `StatusSuite` embedding `suite.Suite`

**`TestParsePorcelainV1`** — table-driven, all status combinations:
- `"clean"` — `""` → all false
- `"dirty only"` — `" M file.go\n"` → dirty true, others false
- `"staged only"` — `"M  file.go\n"` → staged true, others false
- `"untracked only"` — `"?? newfile.go\n"` → untracked true, others false
- `"staged and dirty"` — `"MM file.go\n"` → dirty + staged true
- `"all three"` — `"MM a.go\n?? b.go\n"` → all true
- `"new file staged"` — `"A  newfile.go\n"` → staged true
- `"deleted unstaged"` — `" D file.go\n"` → dirty true
- `"renamed staged"` — `"R  old.go -> new.go\n"` → staged true
- (branch line present but skipped): `"## main\n M file.go\n"` → dirty true

**`TestParseBranchLine`** — table-driven, all remote states:
- `"in sync"` — `"## main...origin/main"` → ("main", InSync)
- `"local ahead"` — `"## main...origin/main [ahead 2]"` → ("main", LocalAhead)
- `"remote ahead"` — `"## main...origin/main [behind 3]"` → ("main", RemoteAhead)
- `"diverged"` — `"## main...origin/main [ahead 1, behind 1]"` → ("main", Diverged)
- `"no remote"` — `"## main"` → ("main", NoRemote)
- `"feature branch"` — `"## feature/auth...origin/feature/auth"` → ("feature/auth", InSync)
- `"detached HEAD"` — `"## HEAD (no branch)"` → ("HEAD", RemoteUnknown)
- `"fresh repo"` — `"## No commits yet on main"` → ("main", NoRemote)

**`TestParseStashCount`** — table-driven:
- `"empty"` — `""` → 0
- `"one entry"` — `"stash@{0}: ...\n"` → 1
- `"three entries"` — three lines → 3

**`TestGetStatus_Smoke`** — single integration test against a real git repo:
- Create repo with `testutil.MakeGitRepo`
- Call `GetStatus(Repo{Name: "x", AbsPath: dir})`
- Assert no error, branch is non-empty (likely "main" or "master"), remote = `NoRemote`, dirty/staged/untracked all false (clean after init commit), LastCommit contains "init"

---

## Wave 3 — Task D: `internal/display/table.go` + `table_test.go`

Depends on: Task B (`visualWidth`, color vars) + Task C (`RepoStatus`, `RemoteState`)

### File: `internal/display/table.go`

**Package**: `package display`

**Imports**: `fmt`, `io`, `strings`, `text/tabwriter` (NOT used for alignment — see note), `github.com/robertwritescode/git-workspace/internal/repo`

**Note on alignment**: ANSI escape codes inflate byte width seen by `text/tabwriter`. Instead, compute max visual column widths manually, then pad each cell using `visualWidth`. Do NOT use `tabwriter`.

**Types**:
```go
// TableEntry pairs a repo name with its current status for table rendering.
type TableEntry struct {
    Name   string
    Status repo.RepoStatus
}
```

**Exported function**:
```go
// RenderTable writes a formatted, color-coded status table to w.
// Each entry occupies one row. Columns are aligned by visual width.
func RenderTable(w io.Writer, entries []TableEntry)
```

**Column layout**:
```
REPO          BRANCH          STATUS  COMMIT
frontend      main ✓          *+?     feat: add login page
```
Four columns: REPO, BRANCH, STATUS, COMMIT. One blank line separates header from data.

No blank line — header then rows. Columns separated by two spaces.

**Branch rendering** (`formatBranch(branch string, state repo.RemoteState) string`):
- Append sync symbol: `✓` (InSync), `↑` (LocalAhead), `↓` (RemoteAhead), `⇕` (Diverged), `∅` (NoRemote/Unknown)
- Color the whole string with the corresponding color var

**Status rendering** (`formatStatus(s repo.RepoStatus) string`):
- Build a string: `*` if Dirty, `+` if Staged, `?` if Untracked, `$` if Stashed
- Empty string if all false (clean)

**Column width computation** (`columnWidths(entries []TableEntry) [4]int`):
- Start with header widths: `[4]int{4, 6, 6, 6}` (REPO, BRANCH, STATUS, COMMIT)
- For each entry, compute visual widths of each column value; take max
- Returns final column widths

**Row padding** (`padTo(s string, width int) string`):
- `s + strings.Repeat(" ", max(0, width - visualWidth(s)))`
- Use `max` builtin (Go 1.21+) or `if/else`; Go 1.26 is in use, so `max` is available

**`RenderTable` body** (high-level):
```go
func RenderTable(w io.Writer, entries []TableEntry) {
    widths := columnWidths(entries)
    writeHeader(w, widths)
    for _, e := range entries {
        writeRow(w, e, widths)
    }
}
```

**`writeHeader`** — writes `REPO`, `BRANCH`, `STATUS`, `COMMIT` padded and joined with two spaces.

**`writeRow`** — formats branch and status, then writes padded columns.

### File: `internal/display/table_test.go`

**Package**: `package display`

**Suite**: `TableSuite` embedding `suite.Suite`

**`SetupTest`**: `color.NoColor = true`

**`TestRenderTable`** — table-driven with golden string comparison:

Fixture entries:
```go
entries := []TableEntry{
    {Name: "frontend", Status: repo.RepoStatus{Branch: "main", RemoteState: repo.InSync, Dirty: true, Staged: true}},
    {Name: "backend",  Status: repo.RepoStatus{Branch: "feature/auth", RemoteState: repo.LocalAhead, Staged: true, LastCommit: "fix: token"}},
    {Name: "infra",    Status: repo.RepoStatus{Branch: "main", RemoteState: repo.RemoteAhead, Untracked: true, LastCommit: "chore: bump"}},
}
```

Test cases:
- `"header present"` — output contains `"REPO"`, `"BRANCH"`, `"STATUS"`, `"COMMIT"`
- `"repo names appear"` — output contains `"frontend"`, `"backend"`, `"infra"`
- `"status symbols"` — frontend row contains `*+`, backend row contains `+`, infra row contains `?`
- `"sync symbols"` — with `color.NoColor = true`, symbols `✓`, `↑`, `↓` appear in expected rows
- `"column alignment"` — each row has same number of fields (split on 2+ spaces); verify widths consistent

**`TestRenderTable_Empty`** — zero entries: output is just the header line, no crash.

**`TestRenderTable_SingleEntry`** — one entry: header + one data row.

**`TestFormatBranch`** — table-driven (white-box test of `formatBranch`):
- With `color.NoColor = true`, verify each symbol appears for each `RemoteState`

**`TestFormatStatus`** — table-driven (white-box test of `formatStatus`):
- All 16 combinations of dirty/staged/untracked/stashed as bitmask; verify correct symbol string

---

## Wave 4 — Task E: `cmd/info.go` + `cmd/info_test.go`

Depends on: Task D (all internal packages complete)

### File: `cmd/info.go`

**Package**: `package cmd`

**Imports**: `fmt`, `github.com/robertwritescode/git-workspace/internal/display`, `github.com/robertwritescode/git-workspace/internal/repo`, `github.com/spf13/cobra`

```go
var infoCmd = &cobra.Command{
    Use:     "info [group]",
    Aliases: []string{"ll"},
    Short:   "Show status table for all repos (or a group)",
    Args:    cobra.MaximumNArgs(1),
    RunE:    runInfo,
}

func init() {
    rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil {
        return err
    }

    repos, err := resolveRepos(cfg, cfgPath, args)
    if err != nil {
        return err
    }

    entries := collectStatuses(repos)
    display.RenderTable(cmd.OutOrStdout(), entries)
    return nil
}
```

**`resolveRepos(cfg, cfgPath string, args []string) ([]repo.Repo, error)`**:
- If `len(args) == 0`: return `repo.FromConfig(cfg, cfgPath)`
- If `len(args) == 1`: interpret as group name; look up `cfg.Groups[args[0]]`; if not found return error `"group %q not found"`; build `[]repo.Repo` from the group's repo names

**`collectStatuses(repos []repo.Repo) []display.TableEntry`**:
- For each repo, call `repo.GetStatus(r)` — sequential (parallel execution is Phase 3)
- If error, use a zero `RepoStatus` with `LastCommit = "(error)"` — do not fail the whole command
- Return `[]display.TableEntry`

**Implementation note**: `collectStatuses` iterates sequentially. Phase 3 will replace this with the parallel executor. Keep the function boundary clean so the swap is a one-liner.

### File: `cmd/info_test.go`

**Package**: `package cmd`

**Suite**: `InfoSuite` embedding `WorkspaceSuite` (from `add_test.go` — already defined in the package)

**`SetupTest`**: inherits from `WorkspaceSuite` — creates `.gitworkspace` in a temp dir and `os.Chdir` into it.

**Test cases**:

**`TestInfo_AllRepos`**:
- Create 2 git repos with `testutil.MakeGitRepo`
- Add both with `execCmd("add", repoPath)`
- Run `execCmd("info")`
- Assert: output contains both repo names; output contains "REPO" header

**`TestInfo_ByGroup`**:
- Create 2 repos, add both, create a group containing only repo 1
- Run `execCmd("info", "mygroup")`
- Assert: output contains repo 1 name; does not contain repo 2 name

**`TestInfo_GroupNotFound`**:
- Run `execCmd("info", "nonexistent")`
- Assert: returns error containing `"not found"`

**`TestInfo_EmptyWorkspace`**:
- No repos added
- Run `execCmd("info")`
- Assert: no error; output contains header

**`TestInfo_Alias`**:
- Add one repo
- Run `execCmd("ll")`
- Assert: no error; output contains repo name and "REPO" header

**`TestInfo_MissingConfig`**:
- Change to a dir with no `.gitworkspace`
- Run `execCmd("info")`
- Assert: returns error

---

## Wave 5 — Task F: Integration Verification

```
go test -race -count=1 ./...
```

All tests must pass. Fix any failures before marking Phase 2 complete.

Checklist before marking complete:
- [ ] `go vet ./...` — no issues
- [ ] `go test -race -count=1 ./...` — all pass
- [ ] Every new `.go` source file passes self-review checklist (no function >~20 lines, no unnecessary comments, no DRY violations, exported symbols have godoc)
- [ ] Every new `_test.go` file uses `testify/suite` and table-driven tests for multi-case scenarios
- [ ] `cmd/info_test.go` uses the `WorkspaceSuite` base already defined in `add_test.go` — no duplication

---

## Exit Criteria

- `git workspace info` and `git workspace ll` both display a formatted, aligned status table with headers
- Color-coded branch state symbols appear in terminal output (verified visually)
- `go test -race -count=1 ./...` passes with zero failures
- No functions exceed ~20 lines without extraction
- No `testify/suite` violations — no bare `func TestXxx(t *testing.T)` doing assertion work

---

## Files Created

| File | Wave | Depends on |
|---|---|---|
| `internal/repo/repo.go` | 1A | — |
| `internal/repo/repo_test.go` | 1A | — |
| `internal/display/colors.go` | 1B | — |
| `internal/display/colors_test.go` | 1B | — |
| `internal/repo/status.go` | 2C | 1A (Repo type) |
| `internal/repo/status_test.go` | 2C | 1A |
| `internal/display/table.go` | 3D | 1B + 2C |
| `internal/display/table_test.go` | 3D | 1B + 2C |
| `cmd/info.go` | 4E | 3D (all internal pkgs) |
| `cmd/info_test.go` | 4E | 3D |
