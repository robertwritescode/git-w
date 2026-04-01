# git-w v2: remote management & multi-destination sync

Detailed implementation reference for remote management in git-w v2. Covers
the `git w sync` fan-out executor, `git w remote` subcommand, workstream push
protection, the `pre-push` hook mechanism, and the state file.

Schema for `[[remote]]`, `[[sync_pair]]`, `[[repo]]` remote additions, and
`[[workstream]]` remote overrides is in `.planning/v2-schema.md`.

---

## Motivation

1. **Mirror** the full state of each origin repo to a personal git server as
   an ongoing backup.
2. **Isolate WIP** — keep feature branches on the personal server only, never
   pushing to origin until rebased and clean.
3. **Sync across machines** — push WIP from machine A to the personal server,
   fetch it on machine B.
4. **Gate org pushes** — some branches hard-block pushes to origin, others
   warn, others require an explicit flag.
5. **Protect workstreams** — while a workstream is active, pushes to any remote
   not on its whitelist must be blocked at the git level via a `pre-push` hook
   — not just through `git w sync`.

Collaborators who do not need multi-remote workflows are unaffected. The config
blocks are purely additive; a `.gitw` with no `[[remote]]` or `[[sync_pair]]`
blocks behaves exactly as before.

---

## `git w sync` — multi-remote fan-out

### v1 behavior
Fetch, pull, and push all repos against their `origin` remote.

### v2 behavior

For each repo, resolve its effective remote list
(`[workspace] default_remotes` → `[[workstream]] remotes` → `[[repo]] remotes`,
innermost wins). Execute the `[[sync_pair]]` fan-out in parallel. Collect
results and print a per-remote grouped summary.

Calls `reconcileHooks` on all repos as a side effect on every run.

**Fan-out execution sequence:**
1. For each repo, resolve effective remote list.
2. Fetch in parallel from all remotes with `direction = "fetch"` or `"both"`.
3. For each `[[sync_pair]]`, push filtered refs from `from` to `to`, subject
   to branch rule evaluation on the `to` remote.
4. Push any local branches not covered by a `[[sync_pair]]` to remotes whose
   branch rules permit them (local-only WIP branches being pushed for the
   first time).
5. Collect results. `critical = true` remote failure marks the repo failed.
   `critical = false` failure is logged; repo is partially successful.
6. Print grouped summary.

**Fetch-only from personal on machine B:**
When a personal remote has `direction = "both"`, `git w sync` fetches all refs
and makes them available as `gitw-<remote-name>/<branch>`. It never
auto-merges fetched refs into local branches. `git w status` shows branches
available from personal that are not present locally, with a suggested checkout
command.

**Mirror push conflict resolution:**
`push_mode = "mirror"` always uses `--force`. Personal is always a derivative
of local state; divergence from WIP pushes on another machine is expected and
correct to overwrite.

**Output format:**
```
Syncing workspace: platform-work (8 repos, 2 remotes)

── origin ────────────────────────────────────────────────────────
  api-service     ✓  pushed main (2 commits)
  auth-lib        ✓  up to date
  gateway         ⚠  feature/new-auth warned (rebase before org push), pushed
  generated-code  ✓  up to date

── personal ──────────────────────────────────────────────────────
  api-service     ✓  mirror pushed (14 refs)
  auth-lib        ✓  mirror pushed (9 refs)
  gateway         ✓  mirror pushed (11 refs)
  generated-code  –  excluded (repo override)

Summary: 8/8 repos synced  |  1 warning  |  0 errors
```

---

## `git w remote` subcommand

```
git w remote list [--json]
git w remote add [flags]
git w remote add \
  --name=personal \
  --kind=gitea \
  --url=https://gitea.example.com \
  --user=me \
  --token-env=GITEA_TOKEN \
  --prefix=work- \
  --private
git w remote remove <name>
git w remote status [<name>]
```

### `git w remote add` — wizard flow

1. Prompt for `name`, `kind`, `url`, `user`, `token_env`.
2. Prompt whether remote should be `private` (→ `.git/.gitw`) or public
   (→ `.gitw`).
3. Prompt for `repo_prefix` and `repo_suffix` (optional, can be empty).
4. If `kind` is `gitea`, `forgejo`, or `github` and `token_env` resolves in
   the environment, call the API to check whether each child repo's target
   remote repo exists. For alias repos (`track_branch` set), the existence
   check uses the upstream repo name since all aliases mirror to a single
   remote repo.
5. For any missing repos: display the list and offer to create them.
6. Write the `[[remote]]` block to the appropriate config file.
7. Upsert `gitw-<name>` as a git remote on each child repo.
8. Offer to run an initial `git push --mirror` for each repo immediately.
9. Offer to add a `[[sync_pair]]` to wire this remote into the fan-out.

### `git w remote remove <name>`

Removes the named remote from config (both files if present in either). Removes
`gitw-<name>` from all child repos via go-git.

**Never calls any API to delete the remote repository.** Deletion of remote
repos is a destructive, irreversible action that git-w will not perform.

### API provisioning — supported kinds

| Kind | API | Existence check | Repo creation |
|---|---|---|---|
| `gitea` | Gitea REST API v1 | yes | yes |
| `forgejo` | Forgejo REST API (Gitea-compatible) | yes | yes |
| `github` | GitHub REST API v3 | yes | yes |
| `generic` | none | no | no |

---

## `git w status` — remote section

A remote staleness section and available-branch hints appear below the
existing per-repo output. Computed from state file only — no network calls.

```
-- remote: personal [gitea]  last sync: 4 min ago -------------------------
  api-service  [ok]  in sync
  auth-lib     [!!]  3 local commits not yet pushed to personal
  gateway      [ok]  in sync
  generated-code  –  excluded (repo override)

  2 branches available from personal not present locally:
    feature/auth-oauth       → git w branch checkout feature/auth-oauth --from=personal
    wip/logging-refactor     → git w branch checkout wip/logging-refactor --from=personal
```

Network connectivity checks only on `git w remote status`.

---

## `git w branch checkout --from=<remote>`

Creates a local branch from a fetched ref on the named remote. Runs across all
repos in the workspace where the ref exists.

```
git w branch checkout feature/auth-oauth --from=personal
# equivalent per repo:
# git checkout -b feature/auth-oauth gitw-personal/feature/auth-oauth
```

Does not push the branch anywhere automatically.

---

## Workstream push protection

### The problem

A workstream is a folder containing worktrees from multiple repos, all checked
out to WIP branches under `workspaces/<workspace>/active/<n>/`. Any tool that
can run `git push` — VSCode, an AI agent, a git GUI — can push directly to
origin from within a worktree without going through `git w sync`. Config-level
`remotes` whitelists protect only `git w sync`; they do not protect against
direct git operations.

Protection must therefore live at the git level: a `pre-push` hook installed
on each child repo that enforces workstream remote rules regardless of what
initiates the push.

### Hook installation

git-w installs a `pre-push` hook on each child repo whose worktrees appear in
any protected workstream. The hook calls `git-w hook pre-push "$@"` which
evaluates the current config at push time.

**Install triggers** — both paths call the same internal `reconcileHooks(repo)`
function:
- `git w workstream create` — installs hooks immediately on all repos with
  worktrees.
- `git w sync` — refreshes hooks on every run as a side effect, ensuring that
  config changes are always reflected without any manual step (self-healing).

**Existing hook handling:**
If `.git/hooks/pre-push` does not exist, git-w writes it in full. If it already
exists (e.g. installed by husky or lefthook), git-w appends a clearly delimited
block and manages only that block:

```sh
# --- git-w managed block (do not edit) ---
git-w hook pre-push "$@"
GITW_EXIT=$?
if [ $GITW_EXIT -ne 0 ]; then exit $GITW_EXIT; fi
# --- end git-w managed block ---
```

On each `reconcileHooks` call, git-w finds this block by its delimiters and
replaces it in place. The rest of the hook file is never touched.

**Binary availability:** The hook requires `git-w` to be on `PATH`. If the
binary is not found, the hook exits non-zero and blocks the push — the safe
default.

**Worktree coverage:** `git worktree add` creates worktrees whose hooks resolve
to the main repo's `.git/hooks/`. Installing the hook on the main repo
automatically covers all of its worktrees — no per-worktree installation needed.

### Hook execution logic

At push time, `git-w hook pre-push` receives the remote name and URL as
arguments from git. It then:

1. Walks up from `$GIT_DIR` to find the workspace root (presence of `.gitw`
   or `.git/.gitw`).
2. Reads the merged config.
3. Resolves the current worktree's absolute path from `$GIT_DIR` and `pwd`.
4. Searches all `.gitw-stream` manifests across all active workstreams for a
   `[[worktree]] path` entry whose resolved absolute path matches.
5. If a match is found, retrieves the workstream's `remotes` whitelist (from
   `[[workstream]] remotes` in `.git/.gitw`).
6. Checks whether the push target remote name appears in the whitelist.
   - Remote is identified by name (e.g. `origin`), not URL.
   - If the remote name is not in the whitelist: **block**.
   - If the remote name is in the whitelist: **allow**.
7. If no workstream match is found for the current worktree: **allow** (hook
   is a no-op for unprotected worktrees).

**Block output:**
```
git-w: push blocked
  Worktree:    workspaces/platform-infra/active/INFRA-42/infra-prod
  Workstream:  INFRA-42
  Remote:      origin  (not in workstream's allowed remotes)
  Allowed:     personal

  This workstream is protected while WIP is active.
  To sync to personal:    git w sync
  To ship to origin:      git w workstream ship
```

### Lifting protection

**Option A — `git w workstream ship`** (recommended):
1. Adds `origin` to the workstream's `remotes` list in `.git/.gitw`.
2. Calls `reconcileHooks` on all repos with worktrees in this workstream.
3. Optionally runs `git w sync` immediately (see `--push-all` flag).

**Option B — manual config edit + sync:**
The user adds `origin` to the `[[workstream]] remotes` list in `.git/.gitw`
directly, then runs `git w sync`. The sync's `reconcileHooks` side effect picks
up the change automatically. No explicit command required beyond the sync.

Both paths produce identical results.

### Hook idempotency and cleanup

`reconcileHooks` is safe to call multiple times. It produces the same result
regardless of current hook state. If a workstream is closed or all its
worktrees are removed from config, `reconcileHooks` removes the git-w managed
block from the hook file. If that leaves the hook file empty or containing only
a shebang, the file is deleted.

---

## State file

**Path:** `.git/git-w-state.json` — machine-local, never committed, never
pushed, never synced by cloud storage tools.

Tracks per-repo, per-remote:
- Last successful push timestamp (RFC3339)
- Remote HEAD SHA at time of last push (for staleness computation)
- Last fetch timestamp
- Hook installation state per repo

Created automatically on first run. Used by `git w status` and
`git w remote status` with no network calls required.

---

## Implementation notes

Follow all existing git-w conventions exactly:
- `pkg/` layout
- `github.com/pelletier/go-toml/v2` via `pkg/toml` wrapper with
  `UpdatePreservingComments`
- Cobra with pflag (no Viper)
- Mage for builds
- `pkg/output.Writef` for all terminal output
- `pkg/cmdutil.ResolveBoolFlag` for mutually exclusive flags
- Atomic config writes
- `Register(root *cobra.Command)` per-command registration
- `go 1.26.0`

**Branch rule engine:**
- `BranchInfo`: `Name string`, `HasUpstreamOn func(remoteName string) bool`,
  `ExplicitOn func(remoteName string) bool`.
- `EvaluateRule(branch BranchInfo, rules []BranchRule, remoteName string) (Action, *BranchRule)` — pure function, no side effects. Returns matched action and
  the matching rule (nil = default allow).
- Glob: `*` does not cross `/`; `**` crosses `/`. Internal package, not
  `filepath.Match` (platform-specific behaviour).
- Table-driven tests: 8 criteria combinations × 4 action tiers × representative
  branch names.

**Parallel fan-out:**
- `errgroup.WithContext` with worker pool.
- Default concurrency: `min(len(syncPairs)*len(repos), runtime.NumCPU()*2)`.
- Fetch phase completes fully before push phase begins.
- Cycle detection at load time; not repeated at runtime.

**Remote naming on local repos:**
- Always `gitw-<remote-name>` (e.g. `gitw-personal`).
- Upserted (add or set-url) on every sync.
- Never modifies or renames `origin`.

**Mirror push:**
- `push_mode = "mirror"` always passes `--force`. Personal divergence from
  cross-machine WIP is expected; force is correct.

**API providers:**
- Interface: `Provider` with `RepoExists(ctx, owner, name string) (bool, error)`
  and `CreateRepo(ctx, owner, name string, opts CreateRepoOpts) error`.
- `internal/provider/gitea.go` — Gitea REST API v1 (also handles forgejo).
- `internal/provider/github.go` — GitHub REST API v3.
- `internal/provider/generic.go` — no-op; `RepoExists` returns
  `(false, ErrNoAPI)`.

**`git w remote add` wizard:**
- Uses `pkg/output` interactive prompts consistent with `git w init`.
- Non-interactive (all flags provided) must be fully testable without a TTY.
- Remote upsert via go-git only; no shell exec.

**`--dry-run`:**
- Output structurally identical to real run; every operation annotated
  `[dry-run]`.
- No git operations, no API calls, no state file writes.

**`reconcileHooks(repo RepoConfig)`:**
- Accepts a repo config and the full merged workspace config.
- Determines whether any workstream references this repo's worktrees by
  scanning `.gitw-stream` manifest files across all active workstreams.
- Hook file path: `<repo-path>/.git/hooks/pre-push`.
- If no workstreams reference this repo: removes git-w managed block if
  present; deletes hook file if it becomes empty or shebang-only.
- If workstreams reference this repo: writes or updates the managed block.
- Managed block delimiters (exact strings, used for find/replace):
  `# --- git-w managed block (do not edit) ---`
  `# --- end git-w managed block ---`
- Hook file is written with mode `0755`.
- If hook file does not exist: write full file with shebang + managed block.
- If hook file exists and contains managed block: replace block in place.
- If hook file exists and does not contain managed block: append block.
- Never modifies any content outside the managed block delimiters.

**`git-w hook pre-push` subcommand:**
- Receives remote name as `$1` and remote URL as `$2` (standard git pre-push
  args passed through `"$@"` in the hook script).
- Uses `GIT_DIR` env var (set by git) to locate the repo root.
- Workspace root discovery: walk up from repo root looking for `.gitw`.
- Worktree path resolution: `filepath.Rel(workspaceRoot, repoWorktreePath)`.
- Workstream match: scan `.gitw-stream` manifests in all active workstream
  directories; compare resolved relative paths against each `[[worktree]] path`
  entry.
- Remote whitelist check: remote name (arg 1) must appear in the matched
  workstream's `[[workstream]] remotes` list in `.git/.gitw`. URL is available
  for diagnostic output only, not for matching.
- Exit 0 = allow. Exit 1 = block. Structured error output to stderr.

---

## Full annotated config example

### `.gitw` (committed, shared with coworkers)

```toml
[workspace]
name = "platform-work"
default_remotes = ["origin"]    # coworkers only see origin

[[remote]]
name      = "origin"
kind      = "github"
direction = "both"
push_mode = "branch"
critical  = true

[[remote.branch_rule]]
pattern = "wip/*"
action  = "block"
reason  = "WIP branches must not be pushed to org"

[[remote.branch_rule]]
pattern = "feature/**"
action  = "warn"
reason  = "Feature branches should be rebased before pushing to org"

[[remote.branch_rule]]
pattern = "**"
action  = "allow"

[[repo]]
name      = "api-service"
path      = "repos/api-service"
clone_url = "https://github.com/work-org/api-service"

[[repo]]
name      = "auth-lib"
path      = "repos/auth-lib"
clone_url = "https://github.com/work-org/auth-lib"

[[repo]]
name      = "gateway"
path      = "repos/gateway"
clone_url = "https://github.com/work-org/gateway"

[[repo]]
name      = "generated-code"
path      = "repos/generated-code"
clone_url = "https://github.com/work-org/generated-code"
remotes   = ["origin"]     # opt out of personal mirror
```

### `.git/.gitw` (private, never committed, machine-local)

```toml
# Field-level merge: adds token_env to origin without redefining the block.
[[remote]]
name      = "origin"
token_env = "GITHUB_TOKEN"

# Personal Gitea — full mirror of all org repos + WIP branches
[[remote]]
name        = "personal"
kind        = "gitea"
url         = "https://gitea.example.com"
user        = "gitea_user"
token_env   = "GITEA_TOKEN"
repo_prefix = "org-"
direction   = "both"
push_mode   = "mirror"
fetch_mode  = "all"
critical    = false
private     = true

[[remote.branch_rule]]
pattern = "**"
action  = "allow"

# Explicit sync routing: all refs flow origin → personal
[[sync_pair]]
from = "origin"
to   = "personal"
refs = ["**"]

# Expand workspace defaults to include personal for all repos
[workspace]
default_remotes = ["origin", "personal"]

# auth-refactor workstream: personal only during WIP phase
[[workstream]]
name    = "auth-refactor"
remotes = ["personal"]

# Contractor remote for sharing auth work externally
[[remote]]
name        = "contractor-gitea"
kind        = "gitea"
url         = "https://gitea.contractor.dev"
user        = "gitea-collab"
token_env   = "CONTRACTOR_GITEA_TOKEN"
repo_prefix = "collab-"
direction   = "push"
push_mode   = "branch"
private     = true

[[remote.branch_rule]]
pattern = "auth-refactor/**"
action  = "allow"

[[remote.branch_rule]]
pattern = "**"
action  = "block"
reason  = "Only auth-refactor/** branches go to contractor remote"

[[sync_pair]]
from = "origin"
to   = "contractor-gitea"
refs = ["auth-refactor/**"]
```

---

## Resolved design decisions

| Question | Decision |
|---|---|
| Ref routing during fan-out | Explicit `[[sync_pair]]` blocks; from → to with optional ref filter |
| Sync execution order | Parallel; fetch phase completes before push phase |
| Remote failure handling | Log and continue per remote; `critical = true` marks repo failed |
| State file location | `.git/git-w-state.json` — always machine-local, never synced |
| Workstream remote restriction | Always active; replaces workspace defaults for member worktrees |
| Workstream protection mechanism | `pre-push` hook calling `git-w hook pre-push`; managed by `reconcileHooks` |
| Hook install trigger | `workstream create` + `sync` side effect (self-healing) |
| Existing hook handling | Append git-w managed block with delimiters; never overwrite |
| Hook binary missing | Block push (exit non-zero) — safe default |
| Push blocked action | Block and explain workstream, blocked remote; suggest `git w sync` or `git w workstream ship` |
| Lifting protection | `git w workstream ship` or manual config edit + sync; both paths identical |
| Fetch from personal on machine B | Fetch-only; never auto-merge; status shows available branches |
| Mirror push on diverged personal | Always `--force`; personal is derivative of local state |
| `git w remote remove` API deletion | Never; git-w will not delete remote repos via API under any circumstances |
| `**` glob semantics | Crosses `/`; `*` does not |
| Workstream source of truth | `.gitw-stream` manifest files in workstream directories |
| Hook worktree discovery | Scan `.gitw-stream` manifests; not root config entries |
| Alias repo mirroring | All aliases sharing one upstream mirror-push to a single remote repo named after the upstream |

---

## Deferred (post-v2.0)

- **Token storage alternatives**: `token_env` requires env var in shell. Future
  milestone: keychain or 1Password CLI as alternative source.
- **`[[sync_pair]]` ref filtering beyond globs**: filter by commit age, exclude
  tags, etc.
- **Forgejo API divergence**: currently treated as Gitea-compatible. Separate
  provider can be added without schema changes if needed.
- **`git w workstream ship` auto-sync**: whether ship auto-runs sync or just
  updates config is configurable in a future revision.
