# git-w v2: command surface

## Design principles

- Every command must be something a human or agent invokes regularly. One-time
  setup operations that can be done by editing config directly do not get
  commands.
- Read commands accept `--json` for machine-readable output. Write commands
  print human-readable confirmations only.
- No `delete` or `remove` commands for repos or workspaces. Repos are excluded
  by removing them from scope. Workspaces accumulate archived workstreams and
  are never deleted.
- Scope filters (`--workspace`, `--workstream`, `--repo`) are expressed at the
  call site on read commands. No ambient scope-setting command exists.
- `--dry-run` is available on all write commands that touch git or the
  filesystem.

---

## Full command tree

```
# Meta-repo lifecycle
git w init
git w restore
git w migrate [--apply]

# Repo management
git w repo add <url> [--name <n>] [--branch <b>] [--no-clone]
git w repo add <url> --branch-map <b>:<n> [<b>:<n> ...] [--upstream-name <n>]
git w repo list [--upstream <n>] [--json]

# Remote management
git w remote add [flags]
git w remote list [--json]
git w remote status [<n>]

# Workspace management
git w workspace create <n> [--description <s>]
git w workspace list [--json]

# Workstream management
git w workstream create <workspace> <n> [--description <s>]
                        [--repo <repo>[:<branch>] ...]
                        [--env-group <upstream>[:<branch>] ...]
                        [--worktree name=<n>,repo=<r>,branch=<b>[,path=<p>][,scope=<s>] ...]
git w workstream list [--workspace <n>] [--active|--archived] [--json]
git w workstream status [--json]
git w workstream switch <n>
git w workstream worktree add <repo> [--branch <branch>]
                              [--worktree-name <n>] [--scope <s>]
git w workstream ship [--push-all] [--open-prs] [--dry-run]
git w workstream close [--no-archive]

# Sync and status
git w sync [--remote <n>] [--workspace <n>] [--workstream <n>]
           [--no-push] [--push-wip] [--dry-run]             (alias: s)
git w status [--workspace <n>] [--workstream <n>] [--repo <n>] [--json]   (alias: st)

# Cross-repo operations
git w branch checkout <branch> [repos...]    (alias: co)
git w branch create <branch> [repos...]      (alias: c)
git w commit [repos...] -m <msg> [--workstream] [--dry-run] [--no-verify]  (alias: ci)
git w exec <git-command> [flags]

# Agent interop
git w context rebuild
git w agent context [--json]

# Infrastructure
git w completion <shell>
```

**Net reduction: 39 → 27 commands/subcommands (−12)**

---

## Cut list from v1

**Top-level commands cut (5):**

| Command | v1 alias | Migration |
|---|---|---|
| `git w info` | `ll` | `git w status` |
| `git w fetch` | `f` | `git w sync --no-push` |
| `git w pull` | `pl` | `git w sync --no-push` |
| `git w push` | `ps` | `git w workstream ship` or `git w sync` |
| `git w context` (scope-setter) | `ctx` | `--workspace`/`--workstream`/`--repo` filter at call site |

**Command families cut entirely (3 families, 17 subcommands):**

`git w group` (aliases `g`) — all 6 subcommands:

| Subcommand | Migration |
|---|---|
| `group add` | `git w workspace create` |
| `group edit` | edit `.gitw` directly |
| `group info` | `git w workspace list` or `git w status --workspace` |
| `group list` | `git w workspace list` |
| `group remove` | workspaces are never deleted |
| `group remove-repo` | edit `.gitw` directly |
| `group rename` | edit `.gitw` directly |

`git w workgroup` (aliases `work`, `wg`) — all 6 subcommands:

| Subcommand | Migration |
|---|---|
| `workgroup create` | `git w workstream create` |
| `workgroup checkout` | `git w workstream switch` |
| `workgroup add` | `git w workstream worktree add` |
| `workgroup drop` | `git w workstream close` |
| `workgroup list` | `git w workstream list` |
| `workgroup path` | paths are predictable; `git w agent context --json` for machines |
| `workgroup push` | `git w workstream ship --push-all` |

`git w worktree` (aliases `tree`, `t`) — all 5 subcommands:

| Subcommand | Migration |
|---|---|
| `worktree clone` | `git w repo add <url> --branch-map ...` + `workstream create` |
| `worktree add` | `git w workstream worktree add` |
| `worktree drop` | `git w workstream close` |
| `worktree list` | `git w workstream status` or `git w status --workstream` |
| `worktree rm` | `git w workstream close`; or `git worktree remove` directly |

**Subcommands cut within kept families (4):**

| Subcommand | Parent | Migration |
|---|---|---|
| `branch default` | `git w branch` | `git w exec checkout <default-branch>` |
| `repo clone` | `git w repo` | merged into `git w repo add <url>` |
| `repo rename` | `git w repo` | edit `.gitw`, rename `repos/<old>` manually |
| `repo unlink` | `git w repo` | remove `[[repo]]` block from `.gitw` |

---

## Command specifications

### `git w init`

Creates `.gitw` in the current directory. Prompts for a workspace name. Creates
`repos/` and `workspaces/` directories. Generates initial meta-repo `AGENTS.md`
and empty `CONTEXT.md`. Optionally runs `git init` if not already a git repo.

### `git w restore`

Materialize all repos: clone missing into `repos/`, pull existing. Re-creates
any worktrees listed in `.gitw-stream` files for active workstreams that are
missing on disk via `git worktree repair`.

### `git w migrate [--apply]`

v1 to v2 upgrade. Always prints migration report. Requires `--apply` to
execute. Does not auto-commit results. See `.planning/v2/v2-migration.md` for
full spec.

### `git w repo add <url> [flags]`

Clones into `repos/<n>`, writes `[[repo]]` block to `.gitw`, upserts configured
remotes as git remotes on the cloned repo.

```
--name <n>             alias name (default: basename of URL)
--branch <b>           track_branch value; omit for standard repos
--no-clone             register in .gitw without cloning
--branch-map <b>:<n>   create multiple aliases (repeatable)
--upstream-name <n>    upstream grouping name for --branch-map (default: URL basename)
```

No `git w repo remove`. To stop tracking a repo: remove its `[[repo]]` block
from `.gitw` and delete `repos/<n>` manually.

### `git w repo list [--upstream <n>] [--json]`

Lists all tracked repos. With `--upstream <n>`: shows only aliases of that
upstream group. Output includes clone status, `track_branch`, `upstream` group,
and which active workstreams each repo appears in.

### `git w remote add [flags]`

Interactive wizard or non-interactive via flags.

```
git w remote add \
  --name=personal --kind=gitea --url=https://gitea.example.com \
  --user=me --token-env=GITEA_TOKEN --prefix=work- --private
```

Checks for existing repos on the remote (Gitea, GitHub providers). Offers to
create missing repos. Offers to add an initial mirror push. Offers to add a
`[[sync_pair]]` block.

### `git w remote list [--json]`

Lists configured remotes with kind, direction, and last-sync timestamp.

### `git w remote status [<n>]`

Connectivity check and last-sync timestamps. Makes network calls.

### `git w workspace create <n> [--description <s>]`

Creates `workspaces/<n>/active/` and `workspaces/<n>/archived/`. Writes
`[[workspace]]` block to `.gitw`. Generates workspace-level `AGENTS.md`.
Creates empty `.planning/`. Runs `git w context rebuild`.

### `git w workspace list [--json]`

Lists workspaces with description, active workstream count, archived workstream
count.

### `git w workstream create <workspace> <n> [flags]`

Creates `workspaces/<workspace>/active/<n>/`. Writes `.gitw-stream`. Generates
workstream-level `AGENTS.md`. Creates empty `.planning/`.

**Flags:**
```
--description <s>
--repo <repo>[:<branch>]
    Simple worktree spec. name defaults to repo name, path defaults to name.
    Error if the same repo appears more than once without --worktree.
--env-group <upstream>[:<branch>]
    Expand all aliases with upstream = "<upstream>" to individual worktrees.
    Branch defaults to workstream name if not specified.
--worktree name=<n>,repo=<r>,branch=<b>[,path=<p>][,scope=<s>]
    Explicit worktree spec. Required when the same repo appears more than once
    in the workstream (Pattern B). path defaults to name. scope is advisory.
```

`--repo`, `--env-group`, and `--worktree` are all composable in the same
command. If neither is provided, creates the directory structure only.

For each resolved worktree entry:
- Validates `name` is unique within the workstream.
- Validates `path` is unique within the workstream.
- Runs `git worktree add <path> -b <branch>` from `repos/<repo>`.
- Adds `[[worktree]]` entry to `.gitw-stream` with all fields populated.
- Calls `reconcileHooks` on the affected repo.

Auto-commits `.gitw-stream`, `AGENTS.md`, and `.planning/` to the meta-repo.
Runs `git w context rebuild`.

### `git w workstream list [--workspace <n>] [--active|--archived] [--json]`

Lists workstreams. Default: all active. `--archived` shows closed workstreams.
Output: name, workspace, description, repo count, status, created date.

### `git w workstream status [--json]`

Current workstream context. Resolved by CWD first, then state file pointer.
Output: name, workspace, description, each worktree (repo, `name` if multiple
worktrees share a repo, `track_branch` if alias, branch, `scope` if set,
`git status -sb` summary), remote staleness, `.planning/` presence.

When a repo appears more than once in the workstream, uses the `<repo> / <name>`
display format. When a repo appears only once, displays as `<repo>` alone.

### `git w workstream switch <n>`

Sets active workstream pointer in `.git/git-w-state.json`. Used when operating
from the meta-repo root where CWD-based resolution is not possible.

### `git w workstream worktree add <repo> [--branch <branch>] [--worktree-name <n>] [--scope <s>]`

Adds a worktree to the current workstream post-creation. Creates the worktree
directory, adds `[[worktree]]` entry to `.gitw-stream`, calls `reconcileHooks`,
commits updated `.gitw-stream`.

`--worktree-name <n>`: required when the repo already has a worktree in this
workstream (Pattern B). Sets the `name` field. Also used as `path` if `--path`
is not provided.

`--scope <s>`: optional advisory subdirectory hint. Sets the `scope` field.
When provided alongside `--worktree-name`, a cross-modification warning is
added to the workstream `AGENTS.md`.

### `git w workstream ship [--push-all] [--open-prs] [--squash] [--dry-run]`

1. Validates all worktrees; warns on uncommitted changes.
2. **Optional squash pass** (`--squash` flag or prompted interactively when
   unpushed-to-origin commits are detected): for each worktree that has commits
   not yet present on `origin/<branch>`, git-w runs an interactive squash
   flow before lifting push protection:
   a. Detects the divergence point between the local branch and `origin/<branch>`
      (or the branch base if origin has no copy yet).
   b. Creates a pre-ship backup branch on the personal remote:
      `<branch>-pre-ship-<timestamp>`. This branch is pushed to `personal` only
      and is never synced back to `origin`. Preserves full messy history.
   c. Records the backup branch name in `.gitw-stream` under
      `[ship] pre_ship_branches` keyed by worktree name.
   d. Prompts for a single squash commit message (pre-filled with the first
      commit subject in the range as a starting point).
   e. Performs a soft-reset to the divergence point and commits with the
      provided message.
   f. Repeats for each remaining worktree with unpushed commits.
3. Lifts push protection: adds `origin` to `[[workstream]] remotes` in
   `.git/.gitw`, calls `reconcileHooks` on all repos.
4. `--push-all`: pushes all worktree branches to `origin` via scoped sync.
5. `--open-prs`: opens one PR per worktree branch on the configured GitHub
   remote. PR URLs written to `.gitw-stream [ship] pr_urls`. Records `shipped_at`.
6. Updates workstream status to `"shipped"` in `.gitw-stream`.
7. Commits updated `.gitw-stream`.

Pre-ship backup branches on the personal remote are intentionally never
included in `[[sync_pair]]` push rules and are never forwarded to `origin`.

### `git w workstream close [--no-archive]`

1. Verifies workstream is shipped; prompts for confirmation if still active.
2. Removes all worktrees via `git worktree remove`.
3. Removes git-w managed block from pre-push hooks via `reconcileHooks`.
4. Optionally prunes local branches; prompts per-branch.
5. Moves `workspaces/<workspace>/active/<n>/` to `archived/<n>/`.
6. Updates `.gitw-stream`: status `"archived"`, records `closed_at`.
7. Updates workstream `AGENTS.md` to reflect archived state.
8. Commits all changes. Runs `git w context rebuild`.

`.planning/` is always preserved. Never deleted by `git w workstream close`.

`--no-archive`: deletes the directory without archiving. Requires explicit
confirmation. Use only for workstreams created in error.

### `git w sync [flags]`

Fan-out sync per `[[sync_pair]]` routing. Calls `reconcileHooks` on all repos
as a side effect. `track_branch` used as pull target for alias repos.

```
--remote <n>       sync only this remote (repeatable)
--workspace <n>    scope to repos in this workspace's active workstreams
--workstream <n>   scope to repos in this workstream
--no-push          fetch only
--push-wip         override require-flag rules flagged --push-wip
--dry-run
```

Fan-out: resolve effective remote list per repo (workspace -> workstream ->
repo, innermost wins). Fetch all `from` remotes in parallel. Push to `to`
remotes per `[[sync_pair]]`. Push local-only WIP branches to permissive remotes.
Print per-remote grouped summary.

### `git w status [--workspace <n>] [--workstream <n>] [--repo <n>] [--json]`

Unified status. Merges v1 `info` and `status` commands.

`--repo <n>` where `<n>` is a repo name or upstream group name:
- Repo name: root clone + all worktrees of that repo across workstreams.
- Upstream name: all aliases grouped, with their worktrees.

Remote staleness from state file (no network calls). Available-branch hints
from personal remote fetched refs.

### `git w branch checkout <branch> [repos...] (alias: co)`

Checks out a branch across repos, creating it locally if it doesn't exist.
Scopes to all repos if no `repos` list is provided. Operates on repos in the
current workstream context if one is active.

Key flags: `--from <remote>`, `--push/--no-push`, `--pull`, `--allow-upstream/--no-upstream`.

`--from <remote>`: fetches the ref from the named remote first, then creates
the local branch from it. Useful for materializing a branch that exists on a
remote without a separate `git w sync` step.

`branch default` (v1) is cut. Use `git w exec checkout <default-branch>`.

### `git w branch create <branch> [repos...] (alias: c)`

Creates a branch across repos. Scopes to all repos if no `repos` list is
provided.

Key flags: `-c/--checkout` (check out after creating), `--push/--no-push`,
`--allow-upstream/--no-upstream`.

### `git w commit [repos...] -m <msg> (alias: ci)`

Atomically commits staged changes across repos. Requires `-m`. Scopes to
specified repos or all repos with staged changes if none specified.

Key flags:
```
-m <msg>          commit message (required)
--workstream      scope to repos in the current workstream (renamed from --workgroup)
--dry-run         show what would be committed without executing
--no-verify       skip pre-commit hooks
```

The `--workgroup` flag is renamed to `--workstream` in v2. The short form
`-W` is preserved.

### `git w exec <git-command> [flags]`

Executes an arbitrary git command across all repos (or scoped repos). The
primary escape hatch for operations not covered by other commands.

Example: `git w exec checkout main` replaces the cut `git w branch default`.

### `git w context rebuild`

Regenerates and commits:
- `CONTEXT.md` at meta-repo root: all repos (with upstream groupings), all
  workspaces, all active workstreams, all archived workstreams.
- `AGENTS.md` at meta-repo root, each workspace root, and each workstream
  directory.

Idempotent. Called automatically by `workstream create`, `workstream close`,
and `workspace create`.

### `git w agent context [--json]`

Emits full context for the current scope. Scope: CWD-based first (workstream
-> workspace -> meta-repo root), then state file pointer.

`--json` output includes `workstream`, `env_groups`, `capabilities`, and
`commands` blocks. The `capabilities` block is the machine-readable equivalent
of the AGENTS.md prohibition section. The `env_groups` block includes a
`create_hint` per group so agents know the correct workstream creation command
without enumerating aliases manually.

---

## `git w status` output format

```
workspace: platform-work  |  4 active workstreams  |  14 repos (1 env group)

-- repos ------------------------------------------------------------------
  api-service   main   repos/api-service             [clean]
  payment-lib   main   repos/payment-lib             [clean]
  consolidated-infra  main  repos/consolidated-infra  [clean]

  infra [env group: dev/test/prod]
    infra-dev   dev    repos/infra-dev     [clean]
    infra-test  test   repos/infra-test    [clean]
    infra-prod  prod   repos/infra-prod    [clean]

-- workstreams ------------------------------------------------------------
  payments-platform / TICKET-456
    service-a  feat/TICKET-456-retry              M  2 files changed
    service-b  feat/TICKET-456-consumer-compat    [clean]

  platform-infra / INFRA-42    [Pattern A: env-group]
    infra-dev   (dev)  feat/INFRA-42-new-rds  [clean]
    infra-test  (test) feat/INFRA-42-new-rds  [clean]
    infra-prod  (prod) feat/INFRA-42-new-rds  M  1 file changed

  platform-infra / TICKET-123  [Pattern B: multi-worktree]
    consolidated-infra / dev   feat/TICKET-123-dev   [scope: environments/dev]   M  2 files
    consolidated-infra / prod  feat/TICKET-123-prod  [scope: environments/prod]  [clean]

  support / BUG-789
    api-service  fix/BUG-789-null-check  [clean]

-- remote: personal [gitea]  last sync: 4 min ago ------------------------
  api-service  [ok]  in sync
  infra-dev    [ok]  in sync
  infra-test   [!!]  3 local commits not yet pushed
  infra-prod   [ok]  in sync

Summary: 14 repos  |  4 active workstreams  |  1 remote warning
```

---

## `--json` output contract

All read commands emit a single JSON object to stdout. Errors to stderr.
Exit codes: 0 = success, 1 = user error, 2 = git/system error.
