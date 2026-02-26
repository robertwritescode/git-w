# Design Decisions

All decisions are resolved.

---

### Binary Name: `git-workspace`
Git's plugin system finds executables named `git-<subcommand>` in `$PATH`.
Binary must be named `git-workspace` for `git workspace` to work. Non-negotiable.

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
Same pattern as `git` finding `.git`. Walk up from CWD until `.gitworkspace` is found.
Stops at filesystem root. Override via `GIT_WORKSPACE_CONFIG` env var or `--config` flag.

### Config Scope: Local (not global)
`.gitworkspace` lives at workspace root, not `~/.config/`. Benefits:
- Can be committed to a meta-repo; multiple independent workspaces per machine
- No hidden global state

### Two Config Files
- **`.gitworkspace`** — committed, shared (repos, groups, settings)
- **`.gitworkspace.local`** — gitignored, per-developer (active context only)

Loader merges both at startup; `.local` values take precedence.

### `init` Command Behavior
`git workspace init [name]` does:
1. Creates `.gitworkspace` with minimal scaffold — errors if already exists
2. Appends `.gitworkspace.local` to `.gitignore` (creates `.gitignore` if absent)
3. Does NOT create `.gitworkspace.local` — created on first `context` write

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
| `list` | `ls` |
| `info` | `ll` |
| `status` | `st` |
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
- `fetch` — `git fetch`
- `pull` — `git pull`
- `push` — `git push`
- `status` (alias: `st`) — `git status -sb`
- `exec` — any git command (escape hatch)

Cut: `br`, `log`, `diff` (covered by `ll` or `exec`), `shell` (too broad).

### Add / Clone / Restore Modes
Three ways to register repos:
1. `git workspace add <path> [-g group]` — register existing local repo; detect URL via `git remote get-url origin`
2. `git workspace clone <url> [<path>]` — clone remote repo and register it
3. `git workspace add -r <dir>` — recursive scan + auto-group

`-r` behavior:
- Walk directory tree; stop descending when a `.git` dir is found (no nested repos)
- Group name = relative path from scan root to repo's parent (e.g. `services/auth` → group `services`)
- Repos directly under scan root get no group

### `git w` Short Alias: Symlink at Install Time
Git requires a `git-w` executable in `$PATH` for `git w` to work.
Approach: install a `git-w` symlink pointing to `git-workspace` via the Homebrew formula
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

### CI/CD: GitHub Actions — Three Workflows
- `ci.yml` — `go vet`, `go test`, `go build` on push/PR to `main`
- `release-please.yml` — runs on push to `main`; opens/updates Release PR via Release Please
- `release.yml` — GoReleaser triggered on `v*` tag push (created by Release Please);
  requires `GITHUB_TOKEN` (auto) and `TAP_GITHUB_TOKEN` (repo secret for tap writes)

### Release Trigger: Release Please (Automated)
Release Please chosen over manual `git tag` push or workflow dispatch:
- Reads conventional commits since last tag; determines semver bump automatically
- Opens a "Release PR" that updates `CHANGELOG.md` and `.release-please-manifest.json`
- Merging the Release PR creates the tag and a draft GitHub Release with notes
- Developer controls release timing by deciding when to merge the Release PR
- Requires conventional commit discipline (`feat:`, `fix:`, `feat!:`, etc.)

### Release Gate: Test Before GoReleaser
`release.yml` runs `go test ./...` before invoking GoReleaser. If tests fail, the
release is aborted — binaries are never built or published for a broken commit.

### Release Please + GoReleaser Integration
Release Please creates a draft GitHub Release with changelog notes; GoReleaser uploads binaries to it.
- `release.replace_existing_draft: true` — GoReleaser finds and updates the draft rather than creating a new release
- `changelog.disable: true` — GoReleaser skips generating its own notes, preserving Release Please's changelog body
- Single source of changelog truth: Release Please (conventional commits → grouped notes → `CHANGELOG.md` + GitHub Release body)

### Distribution: Homebrew Custom Tap
Primary install path:
```sh
brew tap <user>/git-workspace
brew install git-workspace
```
Tap lives in a separate repo `homebrew-git-workspace`. GoReleaser auto-updates the
formula on release. Formula installs the binary AND the `git-w` symlink.

### No Freeze Command
`.gitworkspace` IS the persistent state — always current, committable.

`restore` replaces the freeze/clone workflow:
- Reads local `.gitworkspace`
- For each repo: clone if path missing (requires `url` field), pull if present
- `url` is auto-populated by `add` (via `git remote get-url origin`) and `clone`
- Enforces auto-gitignore on each repo after materializing it

### Testing Library: testify

`github.com/stretchr/testify` (`assert` + `require` packages) used throughout:
- `require.NoError`, `require.Equal` — fatal assertions in test setup
- `assert.Equal`, `assert.Contains` — non-fatal value checks in test bodies
- No other test libraries (no gomock, no ginkgo — keep it simple)

### Testing Architecture: Parse/Invoke Separation

`internal/repo/status.go` separates subprocess invocation from output parsing.
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

Applied on `add`, `clone`, `add -r`, and `restore`.

Check if already ignored:
1. Run `git check-ignore -q <path>` from workspace root
   - Exit 0 → already ignored, skip
   - Exit 1 → not ignored, append to `.gitignore`
   - Error (not a git repo) → fall back to string matching in `.gitignore`
2. Append path to workspace-root `.gitignore`; create file if absent
