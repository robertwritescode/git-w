# Phase 1 Implementation Plan — Scaffold + Config + Basic Commands

**Goal**: Binary builds, config file can be created and edited, repos can be listed.

**Exit criteria**: `git workspace init`, `add`, `rm`, `rename`, `ls` all work. All Phase 1 `_test.go` files pass `go test -race ./...`.

---

## Dependency Graph

```
Foundation (A) ──────────────────────────────────┐
                                                  │
Config Package (B) ───────────────────────────┐  │
                                              │  │
Testutil + Config Tests (C) ──────────────┐  │  │
                                          │  │  │
Commands (D) ──────────────────────────┐  │  │  │
                                       │  │  │  │
Command Tests (E) ──────────────────┐  │  │  │  │
                                    │  │  │  │  │
                                    └──┴──┴──┴──┴──▶  Integration (F)
                                                         go mod tidy
                                                         go build ./...
                                                         go test -race ./...
```

A–E are independent (write to different file paths) and run in parallel.
F runs after all A–E complete.

---

## Agent A — Foundation

**Files to create:**

### `go.mod`
Run: `go mod init github.com/robertwritescode/git-workspace` in repo root (Go 1.26).
This creates go.mod; `go mod tidy` later adds go.sum.

### Directory skeleton
```
mkdir -p cmd internal/config internal/repo internal/executor internal/display internal/testutil .github/workflows
```

### `main.go`
```go
package main

import "github.com/robertwritescode/git-workspace/cmd"

func main() {
    cmd.Execute()
}
```

### `magefile.go`
Build tool with targets:
- `Build`: `go build -ldflags="-X main.version=$(git describe --tags --always)" -o bin/git-workspace .`
- `Install`: `go install .`
- `Test`: `go test -race -count=1 ./...`
- `Cover`: `go test -race -count=1 -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`
- `Vet`: `go vet ./...`
- `All` (default): Vet → Test → Build

Tags: `//go:build mage`

### `.github/workflows/ci.yml`
Trigger: push/PR to main.
Steps: checkout → setup-go (1.26) → `go vet ./...` → `go test -race -count=1 ./...` → `go build ./...`

### `.github/workflows/release-please.yml`
Trigger: push to main.
Uses: `googleapis/release-please-action@v4`
Config: `release-type: go`, reads `release-please-config.json`

### `.github/workflows/release.yml`
Trigger: push tags matching `v*`.
Steps: checkout → setup-go → `go test ./...` → GoReleaser

### `.goreleaser.yaml`
- `builds`: linux/darwin × amd64/arm64, binary name `git-workspace`, ldflags version injection
- `archives`: tar.gz for unix
- `release`: `replace_existing_draft: true`
- `changelog`: `disable: true`
- `brews`: tap `robertwritescode/homebrew-git-workspace`, installs symlink `git-w → git-workspace`

### `release-please-config.json`
```json
{
  "$schema": "https://raw.githubusercontent.com/googleapis/release-please/main/schemas/config.json",
  "release-type": "go",
  "packages": {".": {}}
}
```

### `.release-please-manifest.json`
```json
{".": "0.1.0"}
```

---

## Agent B — Config Package

### `internal/config/types.go`
```go
package config

// WorkspaceConfig is the merged result of .gitworkspace + .gitworkspace.local.
type WorkspaceConfig struct {
    Workspace WorkspaceMeta          `toml:"workspace"`
    Context   ContextConfig          `toml:"context"` // from .local
    Repos     map[string]RepoConfig  `toml:"repos"`
    Groups    map[string]GroupConfig `toml:"groups"`
}

type WorkspaceMeta struct {
    Name          string `toml:"name"`
    AutoGitignore *bool  `toml:"auto_gitignore"` // nil means true (default on)
}

type RepoConfig struct {
    Path  string   `toml:"path"`
    URL   string   `toml:"url,omitempty"`
    Flags []string `toml:"flags,omitempty"`
}

type GroupConfig struct {
    Repos []string `toml:"repos"`
    Path  string   `toml:"path,omitempty"` // for auto-context detection
}

type ContextConfig struct {
    Active string `toml:"active"`
}
```

### `internal/config/loader.go`
Key behaviors:
- `Load(configPath string) (*WorkspaceConfig, error)` — reads `.gitworkspace`, then merges `.gitworkspace.local` (if present; non-existence is not an error)
- `Save(configPath string, cfg *WorkspaceConfig) error` — atomic write: marshal → write to `<path>.tmp` → `os.Rename`
- `SaveLocal(configPath string, ctx ContextConfig) error` — same atomic pattern, only writes `[context]` section
- `ConfigDir(configPath string) string` — returns `filepath.Dir(configPath)`
- Initialize `cfg.Repos` and `cfg.Groups` to non-nil maps before unmarshaling
- TOML library: `github.com/pelletier/go-toml/v2`

### `internal/config/discovery.go`
Key behaviors:
- `const ConfigFileName = ".gitworkspace"`
- `var ErrNotFound = errors.New("no .gitworkspace found")`
- `Discover(startDir string) (string, error)` — checks `GIT_WORKSPACE_CONFIG` env var first; then walks up from startDir, checking for `.gitworkspace` at each level; stops at filesystem root (when `filepath.Dir(dir) == dir`); returns `ErrNotFound` if not found

---

## Agent C — Testutil + Config Tests

### `internal/testutil/helpers.go`
```go
package testutil

// MakeGitRepo creates a temp git repo with an initial commit.
// Returns the absolute path. Caller should use t.TempDir() if they want cleanup.
func MakeGitRepo(t *testing.T, dir string) string

// MakeWorkspace creates a .gitworkspace file in dir with the given TOML content.
// Returns the path to the config file.
func MakeWorkspace(t *testing.T, dir, content string) string
```
MakeGitRepo runs: `git init`, `git config user.email`, `git config user.name`, touches `README.md`, `git add .`, `git commit -m "init"`
All git commands via `exec.Command`; use `t.Helper()` and `require.NoError`.

### `internal/config/loader_test.go`
Test cases:
- `TestLoadRoundTrip` — write TOML, load, verify all fields
- `TestLoadMissingFile` — returns error wrapping `os.ErrNotExist`
- `TestLoadMalformedTOML` — returns parse error
- `TestLoadLocalOverride` — `.local` file's `[context]` takes precedence over empty context
- `TestSaveAtomic` — save, read back, verify; no `.tmp` file left behind
- `TestLoadInitializesNilMaps` — loading a minimal config produces non-nil Repos and Groups maps

### `internal/config/discovery_test.go`
Test cases:
- `TestDiscoverAtRoot` — `.gitworkspace` in startDir itself
- `TestDiscoverOneLevel` — `.gitworkspace` one level up
- `TestDiscoverTwoLevels` — `.gitworkspace` two levels up
- `TestDiscoverNotFound` — no `.gitworkspace` anywhere → returns `ErrNotFound`
- `TestDiscoverEnvVar` — `GIT_WORKSPACE_CONFIG` set → returned directly (file need not exist)

---

## Agent D — Commands

### `cmd/root.go`
- Package `cmd`
- `var rootCmd = &cobra.Command{Use: "git-workspace", ...}`
- `func Execute()` — calls `rootCmd.Execute()`; on error prints to stderr and `os.Exit(1)`
- `PersistentPreRunE` is NOT used on root — subcommands call `loadConfig()` themselves
- Global `--config` flag → `cfgFile` package var
- `func loadConfig() (*config.WorkspaceConfig, string, error)` — uses `cfgFile` or `config.Discover(cwd)`

### `cmd/init.go`
- `Use: "init [name]"`, `Args: cobra.MaximumNArgs(1)`
- If name not provided, use `filepath.Base(cwd)`
- Error if `.gitworkspace` already exists
- Write minimal TOML: `[workspace]\nname = "<name>"\n`
- Call `ensureGitignore(cwd, ".gitworkspace.local")`
- Print success to `cmd.OutOrStdout()`
- Export `ensureGitignore(dir, entry string) error` (package-level, used by add.go too):
  - Read existing `.gitignore`; skip if `entry` already appears as a line
  - Append entry; create file if absent; ensure newline separator

### `cmd/add.go`
- `Use: "add <path>"`, `Args: cobra.ExactArgs(1)`, flag `-g/--group`
- Resolve path to absolute
- Check `.git` subdir exists (isGitRepo helper)
- Repo name = `filepath.Base(absPath)`
- Detect URL: `git -C <path> remote get-url origin` (empty string if fails)
- Compute relPath = `filepath.Rel(configDir, absPath)`
- Error if name already in `cfg.Repos`
- If `auto_gitignore` is true (nil = true): call `ensureGitignore(configDir, relPath)`
- If `--group` set: append name to `cfg.Groups[group].Repos` (create group if absent)
- `config.Save(cfgPath, cfg)`

### `cmd/remove.go`
- `Use: "rm <name(s)>"`, `Aliases: []string{"remove"}`, `Args: cobra.MinimumNArgs(1)`
- For each arg: error if not in `cfg.Repos`; `delete(cfg.Repos, name)`
- Remove name from all group `.Repos` slices
- `config.Save` once after all deletions

### `cmd/rename.go`
- `Use: "rename <old> <new>"`, `Args: cobra.ExactArgs(2)`
- Error if old not found or new already exists
- Copy `cfg.Repos[old]` to `cfg.Repos[new]`, delete old
- Update all group `.Repos` slices: replace old with new
- `config.Save`

### `cmd/list.go`
- `Use: "list [name]"`, `Aliases: []string{"ls"}`, `Args: cobra.MaximumNArgs(1)`
- With name arg: print `filepath.Join(configDir, cfg.Repos[name].Path)` — error if not found
- Without arg: print sorted repo names, one per line

---

## Agent E — Command Tests

All test files use package `cmd_test` (black-box), import `testing` + `testify`.

Pattern for each test:
1. Create temp dir with `t.TempDir()`
2. Use `testutil.MakeGitRepo` or `testutil.MakeWorkspace` as needed
3. Execute cobra command with captured output (use `bytes.Buffer`, set `cmd.SetOut`)
4. Assert output and side effects (file contents)

### `cmd/init_test.go`
- `TestInitCreatesConfig` — runs in temp dir, verifies `.gitworkspace` created with correct TOML
- `TestInitErrorIfExists` — second init returns error
- `TestInitAddsLocalToGitignore` — `.gitignore` contains `.gitworkspace.local`
- `TestInitCreatesGitignoreIfAbsent` — no prior `.gitignore` → creates it
- `TestInitCustomName` — `init myworkspace` sets `name = "myworkspace"` in TOML

### `cmd/add_test.go`
- `TestAddRegistersRepo` — adds a real git repo; config saved with correct path
- `TestAddDetectsRemoteURL` — if origin remote exists, URL is captured
- `TestAddErrorNotGitRepo` — non-git dir returns error
- `TestAddErrorAlreadyRegistered` — adding same repo twice returns error
- `TestAddWithGroup` — `-g mygroup` creates group entry
- `TestAddAutoGitignore` — relPath appears in `.gitignore`

### `cmd/remove_test.go`
- `TestRemoveRepo` — removes from config; config saved
- `TestRemoveMultiple` — removes multiple repos in one call
- `TestRemoveErrorNotFound` — unknown name returns error
- `TestRemoveUpdatesGroups` — removed repo no longer in group membership

### `cmd/rename_test.go`
- `TestRenameRepo` — old key gone, new key present, path unchanged
- `TestRenameUpdatesGroups` — group references updated
- `TestRenameErrorOldNotFound`
- `TestRenameErrorNewExists`

### `cmd/list_test.go`
- `TestListAll` — outputs sorted repo names
- `TestListSinglePath` — `ls <name>` prints absolute path
- `TestListErrorNotFound` — unknown name returns error
- `TestListEmpty` — workspace with no repos outputs nothing

---

## Agent F — Integration (runs after A–E complete)

1. `go mod tidy` — resolves and downloads all imports, writes `go.sum`
2. `go build ./...` — verifies compilation; fix any errors found
3. `go test -race -count=1 ./...` — all tests must pass

---

## Key Constraints

- Module: `github.com/robertwritescode/git-workspace`
- Go version: `1.26`
- TOML lib: `github.com/pelletier/go-toml/v2` (import path: `github.com/pelletier/go-toml/v2`)
- CLI: `github.com/spf13/cobra`
- Colors: `github.com/fatih/color` (unused in Phase 1 but declared in go.mod)
- Concurrency: `golang.org/x/sync` (unused in Phase 1)
- Test assertions: `github.com/stretchr/testify/assert` + `/require`
- No global state beyond cobra command tree + `cfgFile` var
- Atomic config writes: write to `<file>.tmp`, then `os.Rename`
- `cmd/` tests: package `cmd_test` (black-box); call exported `Execute()` or invoke commands directly via `cmd.RunE`
- No TUI, no viper, no go-git library
- All git calls: `os/exec` subprocesses