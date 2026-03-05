# Design Decisions

All decisions are resolved.

---

### Binary Name: `git-w`
Git's plugin system finds executables named `git-<subcommand>` in `$PATH`.
Binary must be named `git-w` for `git w` to work. Non-negotiable.

### Config Format: TOML
Chosen over YAML:
- More explicit (no indentation ambiguity, no implicit type coercion)
- Maps cleanly to Go structs via struct tags (same API as `encoding/json`)

### TOML Library: `pelletier/go-toml` v2
Chosen over `BurntSushi/toml`:
- More actively maintained; Hugo migrated to it from BurntSushi
- Stricter TOML v1.0.0 spec compliance; better error messages
- Identical struct-tag API for our use case

### Config Discovery: Walk Up from CWD
Same pattern as `git` finding `.git`. Walk up from CWD until `.gitw` is found.
Stops at filesystem root. Override via `GIT_W_CONFIG` env var or `--config` flag.

### Config Scope: Local (not global)
`.gitw` lives at workspace root, not `~/.config/`. Benefits:
- Can be committed to a meta-repo; multiple independent workspaces per machine
- No hidden global state

### Two Config Files
- **`.gitw`** — committed, shared (repos, groups, settings)
- **`.gitw.local`** — gitignored, per-developer (active context only)

Loader merges both at startup; `.local` values take precedence.

### `init` Command Behavior
`git w init [name]` does:
1. Creates `.gitw` with minimal scaffold — errors if already exists
2. Appends `.gitw.local` to `.gitignore` (creates `.gitignore` if absent)
3. Does NOT create `.gitw.local` — created on first `context` write

Initial scaffold:
```toml
[workspace]
name = "my-workspace"
```

### CLI Framework: Cobra
Provides subcommand trees, flag parsing, help generation, shell completion scaffolding.

### No Viper
Fixed TOML schema; use `pelletier/go-toml` v2 directly.

### Command Naming: Full Names with Short Aliases
Commands use full descriptive names as canonical; short aliases for ergonomics.
Implemented via Cobra's `Aliases` field — both forms work identically.

| Canonical | Alias |
|---|---|
| `repo list` | `repo ls` |
| `info` | `ll` |
| `status` | `st` |
| `fetch` | `f` |
| `pull` | `pl` |
| `push` | `ps` |
| `repo` | `r` |
| `group` | `g` |
| `group list` | `group ls`, `g ls` |
| `group info` | `group ll`, `g ll` |

### No TUI Framework
Plain `text/tabwriter` table for `ll`. No bubbletea/lipgloss. Keeps binary small,
behavior predictable in CI/scripts.

### `ll` Table Rendering
- Column alignment via stdlib `text/tabwriter`
- Colors via `fatih/color`
- ANSI codes inflate tabwriter byte-width; fix with a `visualWidth()` helper that
  strips escape codes, then pads manually to target column width
- Rows built as pre-colored strings, written to tabwriter with tab separators

### Predefined Command Set: Hardcoded
Static, not configurable. Final execution commands:
- `fetch` (alias: `f`) — `git fetch`
- `pull` (alias: `pl`) — `git pull`
- `push` (alias: `ps`) — `git push`
- `status` (alias: `st`) — `git status -sb`
- `exec` — any git command (escape hatch)

Cut: `br`, `log`, `diff` (covered by `ll` or `exec`), `shell` (too broad).

### Repo Lifecycle Commands Under `repo` Subcommand
Repo lifecycle operations (`add`, `clone`, `unlink`, `rename`, `list`) are grouped
under a `repo` parent command (alias `r`) rather than as direct root subcommands.
Benefits:
- Clearer command surface — `git w repo add` reads as "repo management operation"
- Avoids collision with common git command names at root level
- `restore` is kept directly on root since it's a workspace-wide operation

`repo.Register` creates the parent command, registers lifecycle commands under it,
and adds `restore` directly to root.

### `unlink` Replaces `rm`/`remove` for Repo Removal
`git w repo unlink <name>` removes a repo from the workspace config.
`unlink` was chosen over `rm`/`remove` because:
- Accurately describes the operation (unregisters the repo; does not delete files)
- Avoids confusion with `git rm` (which does affect files)
- Consistent with the semantic of "linking" repos into a workspace

### Add / Clone / Restore Modes
Three ways to register repos:
1. `git w repo add <path> [-g group]` — register existing local repo; detect URL via `git remote get-url origin`
2. `git w repo clone <url> [<path>]` — clone remote repo and register it
3. `git w repo add -r <dir>` — recursive scan + auto-group

`-r` behavior:
- Walk directory tree; stop descending when a `.git` dir is found (no nested repos)
- Group name = relative path from scan root to repo's parent (e.g. `services/auth` → group `services`)
- Repos directly under scan root get no group

### `git w` Short Alias: Symlink at Install Time
Git requires a `git-w` executable in `$PATH` for `git w` to work.
Approach: install a `git-w` symlink pointing to `git-w` via the Homebrew formula
(`bin.install_symlink`). No code changes needed — cobra parses `os.Args[1:]`
regardless of `os.Args[0]`. For non-Homebrew installs, README documents a manual `ln -s`.

### Build Tool: Mage over Makefile
Mage chosen over Makefile:
- Build logic written in Go — no shell quoting issues, cross-platform (Windows-compatible)
- `//go:build mage` tag excludes `magefile.go` from normal `go build`
- `sh.Output()` replaces `$(shell ...)` for dynamic values like version
- `mg.Deps()` handles target dependencies
- Single file at repo root (`magefile.go`); added to `go.mod` as a tool dependency

### Release Tooling: GoReleaser
GoReleaser chosen over hand-rolled shell scripts in GitHub Actions:
- Handles cross-compilation (darwin/linux × amd64/arm64), archives, checksums,
  GitHub Release creation, and Homebrew tap formula updates from a single YAML config
- Industry standard for Go open-source CLIs
- `version` injected at build time via ldflags; surfaced via `rootCmd.Version`

### CI/CD: GitHub Actions — Two Workflows
- `ci.yml` — lint + test + build on push
- `release.yml` — Release Please + GoReleaser combined in a single workflow on push to `main`;
  GoReleaser job is gated on `releases_created` output from the Release Please job.
  This avoids the GitHub Actions limitation where tags created by `GITHUB_TOKEN` cannot trigger
  other workflows. Requires `GITHUB_TOKEN` (auto) and `TAP_GITHUB_TOKEN` (tap writes).

### Release Trigger: Release Please (Automated)
Release Please chosen over manual `git tag` push or workflow dispatch:
- Reads conventional commits since last tag; determines semver bump automatically
- Opens a "Release PR" that updates `CHANGELOG.md` and `.release-please-manifest.json`
- Merging the Release PR creates the tag and a draft GitHub Release with notes
- Developer controls release timing by deciding when to merge the Release PR
- Requires conventional commit discipline (`feat:`, `fix:`, `feat!:`, etc.)

### Release Gate: Test Before GoReleaser
The GoReleaser job in `release.yml` runs `go test ./...` before invoking GoReleaser. If tests fail, the
release is aborted — binaries are never built or published for a broken commit.

### Release Please + GoReleaser Integration
Both run in the same `release.yml` workflow on push to `main`. Release Please runs first and
exposes `releases_created` and `tag_name` as job outputs; GoReleaser is a dependent job
gated on `releases_created == 'true'`.

- Combined workflow avoids the GitHub Actions limitation where tags pushed by `GITHUB_TOKEN` cannot trigger downstream workflows
- `release.replace_existing_draft: true` — GoReleaser finds and updates the draft Release Please created, rather than opening a second release
- `changelog.disable: true` — GoReleaser skips its own notes, preserving Release Please's changelog body
- Single source of changelog truth: Release Please (conventional commits → `CHANGELOG.md` + GitHub Release body)

### Distribution: Homebrew Custom Tap
Primary install path:
```sh
brew tap robertwritescode/tap
brew install git-w
```
Tap lives in a separate repo `github.com/robertwritescode/homebrew-tap`. GoReleaser
auto-updates the formula on release.

### No Freeze Command
`.gitw` IS the persistent state — always current, committable.

`restore` replaces the freeze/clone workflow:
- Reads local `.gitw`
- For each repo: clone if path missing (requires `url` field), pull if present
- `url` is auto-populated by `repo add` (via `git remote get-url origin`) and `repo clone`
- Enforces auto-gitignore on each repo after materializing it

### Testing Library: testify
`github.com/stretchr/testify` (`assert` + `require` packages) used throughout:
- `require.NoError`, `require.Equal` — fatal assertions in test setup
- `assert.Equal`, `assert.Contains` — non-fatal value checks in test bodies
- No other test libraries (no gomock, no ginkgo — keep it simple)

### Testing Architecture: Parse/Invoke Separation
`pkg/repo/status.go` separates subprocess invocation from output parsing.
Parse functions (`parsePorcelainV1`, `parseBranchLine`, `parseStashCount`) take `[]byte`
and are tested exhaustively with fixture strings — no real git subprocess needed for
coverage of all status states.

Thin subprocess wrappers are covered by a single integration smoke test using a real
git repo created via `testutil.MakeGitRepo` (runs `git init` + initial commit in `t.TempDir()`).

### Testing: No Mocking Framework
No gomock, no mockery. Instead:
- Filesystem behaviour: `t.TempDir()` with real files
- Git behaviour: `testutil.MakeGitRepo` (real `git init` subprocess)
- Subprocess flexibility: where injection is needed, pass a function type rather than an interface

### Auto-Gitignore Child Repos
On by default; opt-out via `auto_gitignore = false` in `[workspace]`.
`WorkspaceMeta.AutoGitignore` is `*bool`; nil = true.

Applied on `repo add`, `repo clone`, `repo add -r`, and `restore`.

Check if already ignored:
1. Run `git check-ignore -q <path>` from workspace root
   - Exit 0 → already ignored, skip
   - Exit 1 → not ignored, append to `.gitignore`
   - Error (not a git repo) → fall back to string matching in `.gitignore`
2. Append path to workspace-root `.gitignore`; create file if absent

### Domain Package Layout: `pkg/` with Register Pattern
Commands live in three domain packages under `pkg/`: `workspace`, `repo`, and `git`.
Each package exports a single `Register(root *cobra.Command)` in `register.go` that
calls private `register<Name>` functions — one per command file.

Benefits over the old flat `cmd/<verb>` sub-package layout:
- No variable name collisions between commands in the same domain
- Domain-oriented ownership — clear where new features belong
- `pkg/cmd/root.go` wires three domains with three `Register` calls — zero per-command imports
- New commands: add a file, add one line to `register.go`

### `pkg/parallel/` — Generic Concurrency Primitives
Parallel execution utilities extracted into their own package rather than living in `pkg/git/`:
- `RunFanOut[T, R]` — ordered generic fan-out over goroutines with semaphore
- `MaxWorkers` — bounds worker count (falls back to NumCPU, caps at total)
- `FormatFailureError` — formats a summary error from a list of failure strings

Rationale: `pkg/git/` depends on `pkg/repo`, so if parallel primitives stayed in `pkg/git/`,
any future package that needs concurrency but not git operations would have to import git.
`pkg/parallel/` has no internal dependencies — clean leaf package.

No `golang.org/x/sync` dependency — uses native channels and `sync.WaitGroup`.

### `workspace.LoadConfig(cmd)` — Centralized Config Loading
Config loading is the single canonical entry point for all commands.
- In `pkg/workspace` commands: call `LoadConfig(cmd)` directly (same package)
- In `pkg/repo` and `pkg/git` commands: call `workspace.LoadConfig(cmd)`
- Never inline the flag-reading logic (`cmd.Root().PersistentFlags().GetString("config")`)

`LoadConfig` reads the `--config` flag then delegates to `LoadCWD(override)`.

### `EnsureGitignore` Mutex Protection
`restore` runs gitignore updates concurrently across repos. The read→check→append
sequence in `EnsureGitignore` has a TOCTOU race window. Fixed with a package-level
`sync.Mutex` in `pkg/gitutil/gitutil.go` that serializes all calls process-wide.

### Linting: golangci-lint over `go vet`
`mage lint` runs `golangci-lint fmt --diff` + `golangci-lint run` instead of bare `go vet`.
Provides formatting enforcement plus a broader set of static analysis checks.
CI uses `golangci-lint-action@v7` with version `v2.10.1`.

### `runGitCmd` Shared Helper in `pkg/git/runner.go`
The predefined git commands (fetch, pull, push, status) all share identical logic:
load config → filter repos → run parallel → write results → collect errors.
Extracted to `runGitCmd(cmd, args, gitArgs...)` in `runner.go` to avoid duplicating
this pipeline across four command definitions.

### `ResolveBoolFlag` in `pkg/cmdutil`
The pattern of `--flag` / `--no-flag` pairs with a config-default fallback was duplicated
between `pkg/branch/create.go` and `pkg/workgroup`. Extracted to `cmdutil.ResolveBoolFlag(cmd, onFlag, offFlag, dflt)`.
Both packages import `pkg/cmdutil` — a leaf package with no internal dependencies.

### `repo.SafetyViolations` as Canonical Safety Check
Drop-safety logic (uncommitted changes + unpushed commits) was originally inline in `pkg/worktree/safety.go`.
Extracted to `pkg/repo/safety.go` as `repo.SafetyViolations(ctx, r)` so `pkg/workgroup/drop.go` can reuse it.
`pkg/worktree/safety.go` is now a one-line wrapper delegating to `repo.SafetyViolations`.

### Workgroup Storage: `.gitw.local` + `.workgroup/` directory
Two-part storage for workgroups:
- **Membership metadata** (`WorkgroupConfig`: which repos, branch name, created timestamp) stored in `.gitw.local` under `[workgroup.<name>]`. Never committed — purely local.
- **Worktree directories** at `<configDir>/.workgroup/<name>/<repo>/`. Auto-gitignored via `EnsureGitignore` on create/checkout.

`SaveLocalWorkgroup` and `RemoveLocalWorkgroup` in `pkg/config/loader.go` do read-modify-write on
the `localDiskConfig` struct (which holds both context and workgroups) to avoid clobbering other `.local` fields.

### Workgroup `create` vs `checkout` Design
Two entry points with different strictness:
- `create` — strict by default: fails if workgroup already exists. Pass `--checkout/-c` for idempotent behavior.
  Intended for "start fresh" workflows where collision is an error.
- `checkout` — always idempotent: attaches to existing local branch, fetches+attaches remote branch, or creates new.
  Intended as the "resume" operation; uses stored repo list if workgroup exists.
- `add` — requires existing workgroup; enrolls only repos not already tracked.

This design means `checkout` is safe to script (always succeeds if possible) while `create` protects against
accidental overwrites in interactive use.

### `SilenceUsage: true` on Root Command
Added to `pkg/cmd/root.go`. Cobra's default behavior prints the full usage text on any command error,
which is noisy and unhelpful for runtime errors (as opposed to usage errors). Silencing it means
errors print just the error message.

### Workgroup Worktree Path Convention
Worktrees are stored at `<configDir>/.workgroup/<workgroupName>/<repoName>/`.
`configDir` is the directory containing `.gitw` (not the CWD). This makes worktree paths stable
regardless of where in the workspace the user runs commands from.
