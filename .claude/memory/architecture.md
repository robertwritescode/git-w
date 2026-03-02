# git-w Architecture

## How `git w` Works

The binary is named `git-w`. Git's plugin system discovers any executable
named `git-<subcommand>` in `$PATH` and invokes it, passing remaining args through.
So `git w fetch` → `git-w fetch`.

---

## Config Files

Discovered by walking up from CWD. Env var `GIT_W_CONFIG` or `--config` flag override.

**`.gitw`** — committed, shared workspace definition (repos, groups)
**`.gitw.local`** — gitignored, per-developer state (active context)

Both files are TOML. The loader reads both and merges them; `.local` values take precedence.
`git w init` automatically adds `.gitw.local` to `.gitignore`.

```toml
# .gitw  (committed, shared)

[workspace]
name = "my-workspace"
auto_gitignore = true       # Add repo paths to .gitignore on add/clone/restore (default: true)

[repos.frontend]
path = "apps/frontend"      # Relative to .gitw location
url  = "https://github.com/org/frontend"   # Set by clone/add; used by restore
flags = []                  # Optional custom git flags

[repos.backend]
path = "services/backend"
url  = "https://github.com/org/backend"

[groups.web]
repos = ["frontend", "backend"]
path  = "apps"              # Optional: used for auto-context detection

[groups.ops]
repos = ["infra"]

# Worktree sets: repos sharing a bare clone (one git object store, multiple branch folders)
[worktrees.infra]
url       = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev  = "infra/dev"
test = "infra/test"
prod = "infra/prod"
```

Worktree sets synthesize repos named `<set>-<branch>` (e.g., `infra-dev`) and a group `<set>` at load time. Synthesized entries participate in all filter/execution logic but are NOT written back to `.gitw`.

```toml
# .gitw.local  (gitignored, per-developer)

[context]
active = "web"
```

---

## Directory Structure

```
git-w/
├── main.go
├── go.mod / go.sum
├── magefile.go                     # Mage build targets (excluded via //go:build mage)
├── .goreleaser.yaml                # GoReleaser: cross-compile, archive, Homebrew tap
│
├── .github/
│   └── workflows/
│       ├── ci.yml                  # lint + test + build on push
│       └── release.yml             # Release Please + GoReleaser combined (push to main)
│
└── pkg/
    ├── cmd/
    │   ├── root.go             # Root cmd, global --config flag, wires 4 domain Register funcs
    │   ├── completion.go       # Shell completion (bash/zsh/fish/powershell) — registerCompletion
    │   └── completion_test.go
    │
    ├── config/                 # shared: config types, loader, discovery
    │   ├── config.go           # WorkspaceConfig, RepoConfig, GroupConfig, ContextConfig, WorktreeConfig
    │   ├── loader.go           # TOML load/save, atomic writes, LoadCWD, LoadConfig(cmd); synthesizeWorktreeTargets
    │   ├── discovery.go        # Walk-up .gitw search, Discover()
    │   └── *_test.go
    │
    ├── workspace/              # domain: workspace commands (init, context, group)
    │   ├── register.go         # Register(root) → registerInit + registerContext + registerGroup
    │   ├── cmd_config.go       # Shared withConfig/withMutableConfig/withConfigReadOnly helpers
    │   ├── init.go             # Create new .gitw + .gitignore setup
    │   ├── context.go          # Context show/set/clear/auto
    │   ├── group.go            # Group subcommand tree (add/rm/rename/rmrepo/list/info/edit)
    │   └── *_test.go
    │
    ├── repo/                   # domain: repository lifecycle and commands
    │   ├── repo.go             # Repo type, FromConfig, FromNames, IsGitRepo
    │   ├── filter.go           # Filter, ForContext, ForGroup — repo selection cascade
    │   ├── status.go           # GetStatus, parse functions, RemoteState enum
    │   ├── register.go         # Register(root): creates "repo" subcommand; restore on root directly
    │   ├── add.go              # Add repos (single or -r recursive)
    │   ├── clone.go            # Clone remote repo and register it
    │   ├── unlink.go           # Unregister repos from workspace (command: "unlink")
    │   ├── rename.go           # Rename a repo (alias: mv)
    │   ├── restore.go          # Clone missing, pull existing repos
    │   ├── list.go             # List repo names / get path of one (alias: ls)
    │   └── *_test.go
    │
    ├── git/                    # domain: cross-repo git execution and commands
    │   ├── executor.go         # RunParallel: goroutine pool using pkg/parallel
    │   ├── result.go           # ExecResult, WriteResults, ExecErrors
    │   ├── register.go         # Register(root) → registerGit + registerExec + registerInfo
    │   ├── commands.go         # fetch, pull, push, status command definitions (directly on root)
    │   ├── runner.go           # Shared runGitCmd helper for git subcommands
    │   ├── exec.go             # Execute arbitrary git commands across repos
    │   ├── info.go             # Status table for all or group repos (alias: ll)
    │   └── *_test.go
    │
    ├── worktree/               # domain: git worktree set management commands
    │   ├── register.go         # Register(root): creates "worktree" subcommand (aliases: tree, t)
    │   ├── common.go           # Shared worktree helpers
    │   ├── clone.go            # clone: bare clone + create worktrees; register in .gitw
    │   ├── add.go              # add: add branch worktree to existing set
    │   ├── rm.go               # rm: remove individual worktree; safety checks + --force
    │   ├── drop.go             # drop: destroy entire set (all worktrees + bare repo); safety checks
    │   ├── list.go             # list (alias: ls): list sets or branches of a set
    │   ├── safety.go           # Safety check logic: dirty and local-ahead detection
    │   └── *_test.go
    │
    ├── gitutil/                # shared utility: low-level git subprocess wrappers
    │   ├── gitutil.go          # Clone, CloneContext, CloneBare, AddWorktree, RemoveWorktree, RemoveWorktreeForce, FetchBare, RemoteURL, EnsureGitignore (mutex-protected)
    │   └── gitutil_test.go
    │
    ├── output/                 # shared utility: standardized command output helpers
    │   ├── write.go
    │   └── write_test.go
    │
    ├── parallel/               # shared utility: generic concurrency primitives
    │   ├── parallel.go         # RunFanOut[T,R], MaxWorkers, FormatFailureError
    │   └── parallel_test.go
    │
    ├── display/                # shared utility: terminal output formatting
    │   ├── table.go            # RenderTable: tabwriter-based status table
    │   ├── colors.go           # ANSI color helpers, visualWidth()
    │   └── *_test.go
    │
    └── testutil/               # shared utility: test infrastructure
        ├── helpers.go          # MakeGitRepo, MakeWorkspace, MakeBareGitRepo, AddWorktreeToRepo, ChangeToDir, etc.
        ├── cmd.go              # CmdSuite type: SetRoot, ExecuteCmd for integration tests
        └── suite.go            # CmdSuite method delegates (all helpers available as suite methods)
```

Dependency graph (cycle-free):
```
config     → (none)
workspace  → config, gitutil
repo       → config, gitutil
display    → repo
git        → repo, config, display, parallel
worktree   → config, repo, gitutil, parallel
output     → (none)
parallel   → (none)
gitutil    → (none)
testutil   → (none)
```

---

## Go Types

### Config (`pkg/config/`)

```go
// config.go — merged from .gitw + .gitw.local at load time
type WorkspaceConfig struct {
    Workspace WorkspaceMeta             `toml:"workspace"`
    Context   ContextConfig             `toml:"context"`  // from .local
    Repos     map[string]RepoConfig     `toml:"repos"`
    Groups    map[string]GroupConfig    `toml:"groups"`
    Worktrees map[string]WorktreeConfig `toml:"worktrees"`
}

type WorktreeConfig struct {
    URL      string            `toml:"url"`
    BarePath string            `toml:"bare_path"`
    Branches map[string]string `toml:"branches"` // branch name → relative path
}

// Methods on WorkspaceConfig:
func (c WorkspaceConfig) AutoGitignoreEnabled() bool  // nil → true
func (c *WorkspaceConfig) AddRepoToGroup(group, name string)
func (c *WorkspaceConfig) RepoName(absPath string) (string, error)
func (c *WorkspaceConfig) WorktreeRepoName(setName, branch string) string  // "<set>-<branch>"
func (c *WorkspaceConfig) RemoveRepoFromManualGroups(repoName string)
func (c *WorkspaceConfig) SortedWorktreeBranchNames(setName string) []string

type WorkspaceMeta struct {
    Name          string `toml:"name"`
    AutoGitignore *bool  `toml:"auto_gitignore"` // nil = true (default on)
    SyncPush      *bool  `toml:"sync_push"`      // nil = true (default on)
}

type RepoConfig struct {
    Path  string   `toml:"path"`
    URL   string   `toml:"url,omitempty"`   // remote URL; set by clone/add; required for restore
    Flags []string `toml:"flags,omitempty"`
}

type GroupConfig struct {
    Repos []string `toml:"repos"`
    Path  string   `toml:"path,omitempty"` // for auto-context detection
}

type ContextConfig struct {
    Active string `toml:"active"`
}

// loader.go
func Load(configPath string) (*WorkspaceConfig, error)
func Save(configPath string, cfg *WorkspaceConfig) error
func SaveLocal(configPath string, ctx ContextConfig) error
func LoadCWD(override string) (*WorkspaceConfig, string, error)
func LoadConfig(cmd *cobra.Command) (*WorkspaceConfig, string, error)
func ConfigDir(configPath string) string
func ResolveRepoPath(cfgPath, repoPath string) (string, error)
func RelPath(cfgPath, absPath string) (string, error)

// discovery.go
const ConfigFileName = ".gitw"
var ErrNotFound = errors.New("no .gitw found in current directory or any parent")
func Discover(startDir string) (string, error)
```

### Repo (`pkg/repo/`)

```go
// repo.go
type Repo struct {
    Name    string
    AbsPath string    // config root dir + RepoConfig.Path
    Flags   []string
}

func FromConfig(cfg *workspace.WorkspaceConfig, cfgPath string) []Repo
func FromNames(cfg *workspace.WorkspaceConfig, cfgPath string, names []string) []Repo
func IsGitRepo(path string) bool

// filter.go — repo selection cascade
func Filter(cfg *workspace.WorkspaceConfig, cfgPath string, names []string) ([]Repo, error)
func ForContext(cfg *workspace.WorkspaceConfig, cfgPath string) ([]Repo, error)
func ForGroup(cfg *workspace.WorkspaceConfig, cfgPath string, groupName string) ([]Repo, error)

// status.go
type RemoteState int
const (
    Detached RemoteState = iota
    InSync
    LocalAhead
    RemoteAhead
    Diverged
    NoRemote
)

type RepoStatus struct {
    Branch      string
    RemoteState RemoteState
    Dirty       bool   // unstaged changes
    Staged      bool   // staged changes
    Untracked   bool
    Stashed     bool
    LastCommit  string
}

func GetStatus(r Repo) (RepoStatus, error)
```

### Parallel (`pkg/parallel/`)

```go
// parallel.go
func MaxWorkers(configured, total int) int  // bounds worker count; falls back to NumCPU
func RunFanOut[T any, R any](items []T, workers int, fn func(T) R) []R  // ordered fan-out
func FormatFailureError(failures []string, total int) error  // nil if no failures
```

### Executor (`pkg/git/`)

```go
// executor.go — uses pkg/parallel internally
type ExecOptions struct {
    MaxConcurrency int           // 0 → runtime.NumCPU()
    Timeout        time.Duration // 0 → no timeout
    Async          bool          // false = serial run with stdin passthrough
}

func RunParallel(repos []repo.Repo, args []string, opts ExecOptions) []ExecResult

// result.go
type ExecResult struct {
    RepoName string
    Stdout   []byte
    Stderr   []byte
    ExitCode int
    Err      error
}

func WriteResults(w io.Writer, results []ExecResult)  // writes prefixed output
func ExecErrors(results []ExecResult) error           // returns combined error if any failed
```

### Display (`pkg/display/`)

```go
// table.go
type TableEntry struct {
    Name        string
    Branch      string
    RemoteState repo.RemoteState  // uses canonical repo.RemoteState enum
    Dirty       bool
    Staged      bool
    Untracked   bool
    Stashed     bool
    LastCommit  string
}

func RenderTable(w io.Writer, entries []TableEntry)
```

### Gitutil (`pkg/gitutil/`)

```go
func Clone(url, destPath string) error
func CloneContext(ctx context.Context, url, destPath string) error
func CloneBare(ctx context.Context, url, dest string) error
func AddWorktree(ctx context.Context, barePath, treePath, branch string) error
func RemoveWorktree(barePath, treePath string) error
func RemoveWorktreeForce(barePath, treePath string) error
func FetchBare(barePath string) error
func RemoteURL(repoPath string) string
func EnsureGitignore(dir, entry string) error  // mutex-protected for concurrent use
```

---

## Command Inventory

### Workspace Management (directly on root)

| Command | Description |
|---|---|
| `git w init [name]` | Create `.gitw` in current directory; add `.gitw.local` to `.gitignore` |
| `git w restore` | For each repo in `.gitw`: clone if path missing, pull if present; enforce gitignore |

### Repo Lifecycle (`git w repo` / `git w r`)

| Command | Description |
|---|---|
| `git w repo add <path> [-g group]` | Register an existing local repo |
| `git w repo add -r <dir>` | Recursively find and register all repos under `<dir>`; auto-create groups from directory structure |
| `git w repo clone <url> [<path>]` | Clone a remote repo and register it |
| `git w repo unlink <name(s)>` | Unregister repos from workspace (does not delete directories) |
| `git w repo rename <old> <new>` | Rename a tracked repo (alias: `mv`) |
| `git w repo list [name]` | List repo names or print path of one (alias: `ls`) |

### Group Management

`group` has alias `g` — e.g. `git w g add ...` works identically.

| Command | Description |
|---|---|
| `git w group add <repos> -n <name>` | Create group / add repos to group |
| `git w group rm <name>` | Delete group |
| `git w group rename <old> <new>` | Rename group |
| `git w group rmrepo <repos> -n <name>` | Remove repos from group |
| `git w group list` (alias: `ls`) | List group names |
| `git w group info [name]` (alias: `ll`) | List groups with their repos |

### Worktree Set Management (`git w worktree` / `git w tree` / `git w t`)

| Command | Description |
|---|---|
| `git w worktree clone <url> <base-path> <branch> [branch...]` | Bare clone + create a worktree per branch; register set in `.gitw` |
| `git w worktree add <set-name> <branch> [path]` | Add a new branch worktree to an existing set |
| `git w worktree rm <name>` | Remove individual worktree (e.g., `infra-dev`); safety checks; `--force` override |
| `git w worktree drop <set-name>` | Destroy all worktrees + bare repo for a set; safety checks; `--force` override |
| `git w worktree list [set-name]` (alias: `ls`) | List all sets, or branches of a specific set |

Safety checks for `rm` and `drop`: refuses if working tree is dirty or has unpushed commits unless `--force`.
`rm` also refuses if it would remove the last worktree in a set (use `drop` instead).

**`restore`** handles worktrees: bare-clones if missing, runs `git worktree add` per branch if path missing, else `git pull`. Concurrent with regular repo restore.

**`fetch` deduplication**: fetching a worktree set fetches once via `git -C <bare-path> fetch`. All-repo fetch deduplicates worktrees sharing a bare path.

### Context

| Command | Description |
|---|---|
| `git w context <group>` | Scope all commands to group |
| `git w context auto` | Auto-detect group from CWD |
| `git w context none` | Clear active context |
| `git w context` | Show active context |

### Execution (directly on root)

All execution commands accept optional `[repo/group names...]` to filter targets.
When no filter: uses active context if set, otherwise all repos.

| Command | Alias | Async | Description |
|---|---|---|---|
| `git w fetch [repos]` | `f` | yes | `git fetch` |
| `git w pull [repos]` | `pl` | yes | `git pull` |
| `git w push [repos]` | `ps` | yes | `git push` |
| `git w status [repos]` | `st` | yes | `git status -sb` |
| `git w exec [repos] -- <git-args>` | — | yes* | Any git command |
| `git w info [group]` | `ll` | — | Status table for all or group repos |

*`exec` with a single repo target: always synchronous (stdin passthrough).

---

## Status Display (`info` / `ll`)

```
REPO          BRANCH          STATUS  COMMIT
frontend      main ✓          *+      feat: add login page
backend       feature/auth ↑  +       fix: token validation
infra         main ↓          ?       chore: bump versions
```

**Branch color coding:**
- `✓` green — in sync with remote
- `↑` purple — local ahead (push ready)
- `↓` yellow — remote ahead (pull ready)
- `⇕` red — diverged
- `∅` white — no remote

**Status symbols:**
- `*` dirty (unstaged changes)
- `+` staged changes
- `?` untracked files
- `$` stashed content

---

## Auto-Gitignore Logic

Applied on `repo add`, `repo clone`, `repo add -r`, and `restore` when `auto_gitignore` is true (default).

**Checking if a path is already ignored:**
1. Run `git check-ignore -q <path>` from workspace root
   - Exit 0 → already ignored, skip
   - Exit 1 → not ignored, append to `.gitignore`
   - Non-zero with error (not a git repo) → fall back to reading `.gitignore` and checking for exact line or prefix match
2. Write `<path>` as a new line in the workspace-root `.gitignore`; create file if absent

**`restore` enforcement:**
After cloning/pulling each repo, apply the same check — ensures a fresh-machine restore also sets up `.gitignore` correctly.

---

## `git w` Short Alias

`git w <cmd>` requires a `git-w` executable in `$PATH`. Implemented as a symlink
(`git-w` → `git-w`) installed by the Homebrew formula. No code changes
needed — Cobra parses `os.Args[1:]` regardless of `os.Args[0]`.

See [release.md](release.md) for full build, CI/CD, and distribution details.

---

## Testing

All non-trivial logic has unit tests. See `testing.md` for full details.

**Key patterns:**
- `status.go` separates parsing from subprocess calls — parse functions take `[]byte` and are tested with fixture strings
- Filesystem tests use `t.TempDir()`; git repo tests use `testutil.MakeGitRepo` (runs `git init` + initial commit)
- `pkg/` tests use black-box `package <domain>_test`, call via `s.ExecuteCmd()` with captured stdout
- `display/` tests set `color.NoColor = true` and compare against golden strings
- CI: `go test -race -count=1 ./...` in both `ci.yml` and `release.yml`

---

## Dependencies (`go.mod`)

```
github.com/spf13/cobra          v1.x   CLI framework
github.com/pelletier/go-toml/v2 v2.x   TOML parsing
github.com/fatih/color          v1.x   ANSI terminal colors
github.com/stretchr/testify     v1.x   assert + require for unit tests (test only)
```

No `golang.org/x/sync` — parallel execution uses native goroutines with channels and `sync.WaitGroup` in `pkg/parallel`.

---

## Key Differences from gita

| Concern | gita | git-w |
|---|---|---|
| Language | Python 3.6+ | Go — single compiled binary |
| Config location | `~/.config/gita/` (global) | `.gitw` (local, workspace-scoped) |
| Config format | Multiple CSV + JSON files | Single TOML file |
| Config discovery | Env var or global default | Walk up from CWD (like `.git`) |
| Concurrency | asyncio + ThreadPoolExecutor | goroutines + semaphore (`pkg/parallel`) |
| Installation | pip/pipx | `go install` or release binary |
| Invocation | `gita <cmd>` | `git w <cmd>` |
| Version control | Config is global, not in repo | `.gitw` can be committed |
