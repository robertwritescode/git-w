# git-workspace Architecture

## How `git workspace` Works

The binary is named `git-workspace`. Git's plugin system discovers any executable
named `git-<subcommand>` in `$PATH` and invokes it, passing remaining args through.
So `git workspace fetch` → `git-workspace fetch`.

---

## Config Files

Discovered by walking up from CWD. Env var `GIT_WORKSPACE_CONFIG` or `--config` flag override.

**`.gitworkspace`** — committed, shared workspace definition (repos, groups)
**`.gitworkspace.local`** — gitignored, per-developer state (active context)

Both files are TOML. The loader reads both and merges them; `.local` values take precedence.
`git workspace init` automatically adds `.gitworkspace.local` to `.gitignore`.

```toml
# .gitworkspace  (committed, shared)

[workspace]
name = "my-workspace"
auto_gitignore = true       # Add repo paths to .gitignore on add/clone/restore (default: true)

[repos.frontend]
path = "apps/frontend"      # Relative to .gitworkspace location
url  = "https://github.com/org/frontend"   # Set by clone/add; used by restore
flags = []                  # Optional custom git flags (e.g., bare repo worktrees)

[repos.backend]
path = "services/backend"
url  = "https://github.com/org/backend"

[groups.web]
repos = ["frontend", "backend"]
path  = "apps"              # Optional: used for auto-context detection

[groups.ops]
repos = ["infra"]
```

```toml
# .gitworkspace.local  (gitignored, per-developer)

[context]
active = "web"
```

---

## Directory Structure

```
git-workspace/
├── main.go
├── go.mod
├── go.sum
├── magefile.go                     # Mage build targets (excluded from normal builds)
├── .goreleaser.yaml                # GoReleaser: cross-compile, archive, Homebrew tap
├── release-please-config.json      # Release Please: release type + changelog config
├── .release-please-manifest.json   # Release Please: tracks current version
├── CHANGELOG.md                    # Auto-updated by Release Please on each release
│
├── .github/
│   └── workflows/
│       ├── ci.yml                  # vet + test + build on push/PR to main
│       ├── release-please.yml      # Opens Release PR on push to main
│       └── release.yml             # test + GoReleaser on v* tag push
│
├── cmd/
│   ├── root.go             # Root cmd, config loading, global flags
│   ├── init.go             # Create new .gitworkspace
│   ├── add.go              # Add repos
│   ├── remove.go           # Remove repos
│   ├── rename.go           # Rename a repo
│   ├── list.go             # List repo names / get path of one (alias: ls)
│   ├── info.go             # Status table for all or group repos (alias: ll)
│   ├── group.go            # Group subcommand tree
│   ├── context.go          # Context subcommand
│   ├── exec.go             # Execute arbitrary git commands across repos
│   ├── clone.go            # Clone a single remote repo and register it
│   ├── restore.go          # Materialize all repos from .gitworkspace (clone missing, pull existing)
│   └── git_cmds.go         # Predefined git commands (fetch, pull, push, status)
│
└── internal/
    ├── config/
    │   ├── types.go        # Config structs
    │   ├── loader.go       # TOML load/save
    │   └── discovery.go    # Walk-up .gitworkspace search
    │
    ├── repo/
    │   ├── repo.go         # Repo type, path resolution, git detection
    │   └── status.go       # Status detection (dirty/staged/untracked/stash/remote)
    │
    ├── executor/
    │   ├── parallel.go     # Goroutine pool + semaphore, result collection
    │   └── result.go       # ExecResult type, output formatting
    │
    ├── display/
    │   ├── table.go        # Status table renderer (ll command)
    │   └── colors.go       # ANSI color constants and helpers
    │
    └── testutil/
        └── helpers.go      # MakeGitRepo, MakeWorkspace — shared test helpers (test-only)
```

---

## Go Types

### Config (`internal/config/types.go`)

```go
// Merged from .gitworkspace + .gitworkspace.local at load time
type WorkspaceConfig struct {
    Workspace WorkspaceMeta          `toml:"workspace"`
    Context   ContextConfig          `toml:"context"`  // from .local
    Repos     map[string]RepoConfig  `toml:"repos"`
    Groups    map[string]GroupConfig `toml:"groups"`
}

type WorkspaceMeta struct {
    Name          string `toml:"name"`
    AutoGitignore *bool  `toml:"auto_gitignore"` // nil = true (default on)
}

type RepoConfig struct {
    Path  string   `toml:"path"`
    URL   string   `toml:"url,omitempty"`   // remote URL; set by clone/add; required for restore
    Flags []string `toml:"flags,omitempty"`
}

type GroupConfig struct {
    Repos []string `toml:"repos"`
    Path  string   `toml:"path"` // for auto-context detection
}

type ContextConfig struct {
    Active string `toml:"active"`
}
```

### Repo (`internal/repo/repo.go`)

```go
type Repo struct {
    Name    string
    AbsPath string    // config root dir + RepoConfig.Path
    Flags   []string
}

type RemoteState int
const (
    RemoteUnknown RemoteState = iota
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
```

### Executor (`internal/executor/parallel.go`)

```go
type ExecOptions struct {
    MaxConcurrency int           // default: runtime.NumCPU()
    Timeout        time.Duration // default: 0 (no timeout)
    Async          bool          // false = single serial run with stdin passthrough
}

type ExecResult struct {
    RepoName string
    Stdout   []byte
    Stderr   []byte
    ExitCode int
    Err      error
}

// RunParallel executes args in each repo concurrently.
// Single-repo or non-async: stdin passes through (interactive).
// Multi-repo async: stdin suppressed (os.DevNull), output prefixed "[repo-name]".
func RunParallel(repos []repo.Repo, args []string, opts ExecOptions) []ExecResult
```

---

## Command Inventory

### Workspace Management

| Command | Description |
|---|---|
| `git workspace init [name]` | Create `.gitworkspace` in current directory; add `.gitworkspace.local` to `.gitignore` |
| `git workspace add <path> [-g group]` | Register an existing local repo |
| `git workspace add -r <dir>` | Recursively find and register all repos under `<dir>`; auto-create groups from directory structure |
| `git workspace clone <url> [<path>]` | Clone a remote repo and register it |
| `git workspace rm <name(s)>` | Unregister repos |
| `git workspace rename <old> <new>` | Rename a tracked repo |
| `git workspace list [name]` (alias: `ls`) | List repo names or print path of one |
| `git workspace info [group]` (alias: `ll`) | Status table for all or group repos |
| `git workspace restore` | For each repo in `.gitworkspace`: clone if path missing, pull if present; enforce gitignore |

### Group Management

`group` has alias `g` — e.g. `git workspace g add ...` works identically.

| Command | Description |
|---|---|
| `git workspace group add <repos> -n <name>` | Create group / add repos to group |
| `git workspace group rm <name>` | Delete group |
| `git workspace group rename <old> <new>` | Rename group |
| `git workspace group rmrepo <repos> -n <name>` | Remove repos from group |
| `git workspace group list` (alias: `ls`) | List group names |
| `git workspace group info [name]` (alias: `ll`) | List groups with their repos |

### Context

| Command | Description |
|---|---|
| `git workspace context <group>` | Scope all commands to group |
| `git workspace context auto` | Auto-detect group from CWD |
| `git workspace context none` | Clear active context |
| `git workspace context` | Show active context |

### Execution

All execution commands accept optional `[repo/group names...]` to filter targets.
When no filter: uses active context if set, otherwise all repos.

| Command | Async | Description |
|---|---|---|
| `git workspace fetch [repos]` | yes | `git fetch` |
| `git workspace pull [repos]` | yes | `git pull` |
| `git workspace push [repos]` | yes | `git push` |
| `git workspace status [repos]` (alias: `st`) | yes | `git status -sb` |
| `git workspace exec [repos] -- <git-args>` | yes* | Any git command |

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

Applied on `add`, `clone`, `add -r`, and `restore` when `auto_gitignore` is true (default).

**Checking if a path is already ignored:**
1. Run `git check-ignore -q <path>` from workspace root
   - Exit 0 → already ignored, skip
   - Exit 1 → not ignored, append to `.gitignore`
   - Non-zero with error (not a git repo) → fall back to reading `.gitignore` and checking for exact line or prefix match
2. Write `<path>` as a new line in the workspace-root `.gitignore`; create file if absent

**`restore` enforcement:**
After cloning/pulling each repo, apply the same check — ensures a fresh-machine restore also sets up `.gitignore` correctly, even if the `.gitworkspace` was committed before auto-gitignore existed.

---

## `git w` Short Alias

`git w <cmd>` requires a `git-w` executable in `$PATH`. Implemented as a symlink
(`git-w` → `git-workspace`) installed by the Homebrew formula. No code changes
needed — Cobra parses `os.Args[1:]` regardless of `os.Args[0]`.

See [release.md](release.md) for full build, CI/CD, and distribution details.

---

## Testing

All non-trivial logic has unit tests. See `testing.md` for full details.

**Key patterns:**
- `status.go` separates parsing from subprocess calls — parse functions take `[]byte` and are tested with fixture strings
- Filesystem tests use `t.TempDir()`; git repo tests use `testutil.MakeGitRepo` (runs `git init` + initial commit)
- `cmd/` tests use black-box `package cmd_test`, call `Execute()` with captured stdout
- `display/` tests set `color.NoColor = true` and compare against golden strings
- CI: `go test -race -count=1 ./...` in both `ci.yml` and `release.yml`

---

## Dependencies (`go.mod`)

```
github.com/spf13/cobra          v1.x   CLI framework
github.com/pelletier/go-toml/v2 v2.x   TOML parsing
github.com/fatih/color          v1.x   ANSI terminal colors
golang.org/x/sync               v0.x   errgroup for parallel execution
github.com/stretchr/testify     v1.x   assert + require for unit tests (test only)
```

---

## Key Differences from gita

| Concern | gita | git-workspace |
|---|---|---|
| Language | Python 3.6+ | Go — single compiled binary |
| Config location | `~/.config/gita/` (global) | `.gitworkspace` (local, workspace-scoped) |
| Config format | Multiple CSV + JSON files | Single TOML file |
| Config discovery | Env var or global default | Walk up from CWD (like `.git`) |
| Concurrency | asyncio + ThreadPoolExecutor | goroutines + semaphore |
| Installation | pip/pipx | `go install` or release binary |
| Invocation | `gita <cmd>` | `git workspace <cmd>` |
| Version control | Config is global, not in repo | `.gitworkspace` can be committed |
