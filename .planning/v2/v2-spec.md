# git-w v2: consolidated design spec
## workspace management, agent interop, and multi-destination sync

**Version:** 2.0 (major)
**Supersedes:** git-w v1.x
**Breaking changes:** workgroup retirement, command surface reduction, directory
layout migration, `[[repo]] path` convention change to `repos/<n>`,
`git w context` command repurposed. A `git w migrate` command is provided for
the v1 to v2 upgrade path.

---

## Overview

This document is the authoritative design spec for git-w version 2.0. It
consolidates three previously separate design tracks and introduces new
capabilities that together necessitate a major version bump:

1. **Remote management and multi-destination sync** — `[[remote]]`,
   `[[sync_pair]]`, branch rule enforcement, pre-push hook protection, and the
   `git w remote` and `git w sync` command extensions.
2. **Workspace and workstream management** — replacing workgroups with a
   two-level organizational hierarchy, introducing workstream-scoped planning
   state, and building a first-class agent interop layer so that tools like GSD,
   Claude Code, and opencode can operate within git-w-managed environments
   without reinventing workspace management.
3. **Flexible infra repo patterns** — support for two common infrastructure
   repo conventions: branch-per-environment (one branch per env, modeled as
   repo aliases with `track_branch`) and folder-per-environment (one main
   branch with env subdirectories, modeled as multiple named worktrees from
   the same repo within a single workstream). Both patterns are first-class.

The full remote management design is specified in
`.planning/v2/v2-remote-management.md`. The vocabulary substitution throughout is
`workgroup` to `workstream`, with paths changing from `workgroups/<n>/` to
`workspaces/<workspace>/active/<n>/`. The hook mechanism, config merge
semantics, branch rule engine, and fan-out executor are unchanged.

No new binary is introduced. Everything lives in git-w.

---

## How to work in this repo

**Branch hierarchy (3 tiers):**

- `main` — v1.6.x stable; receives only the final v2 cut-over PR.
- `v2` — long-lived protected base for all v2 work; all milestone branches target this.
- `v2-m<N>-<short-slug>` — one branch per milestone (e.g. `v2-m1-core-config`); created off `v2`, merged back into `v2`.
- `<issue-number>-<kebab-description>` — one branch per GitHub issue (e.g. `42-add-workstream-model`); created off its milestone branch, merged back into that milestone branch.

**Sequencing rule:** milestones run strictly in order M1 → M11. M12 (`git w migrate`) may run in parallel once M1 has merged into `v2`.

**GSD workflow mapping:**

The v2 effort is one GSD project. Each v2 milestone is one GSD milestone. Each GitHub issue is one GSD phase. Implementation tasks within an issue are GSD plans. GSD's `branching_strategy` is set to `none` — GSD commits directly to the active branch; all branch creation and PR management is done manually.

Per-issue workflow:
1. Create the issue branch off the milestone branch. **Use `.planning/v2/v2-issue-map.md` for the exact branch name and issue number.**
2. Optionally run `/gsd-discuss-phase` to refine scope.
3. Run `/gsd-plan-phase` to generate the implementation plan.
4. Run `/gsd-execute-phase` to implement.
5. Verify: `mage testfast` and `go vet ./...` must pass.
6. Open a PR targeting the milestone branch and merge it.
7. Update the Active State section in `.planning/v2/v2-strategy.md`.

**Read before planning or branching:**

- `.planning/v2/v2-issue-map.md` — authoritative mapping of every GitHub issue number, issue title, milestone, branch name, and dependency. Use this to determine the correct branch name, issue number, and milestone scope for any unit of work.
- `.planning/v2/v2-strategy.md` — branching rules, naming conventions, active state, sequencing.
- `.planning/v2/v2-workflow.md` — GSD command invocations and step-by-step workflow.
- `.planning/v2/v2-remote-management.md` — detailed remote management design: `[[remote]]`, `[[sync_pair]]`, branch rule engine, pre-push hook protection, `git w remote` and `git w sync` multi-remote behaviour.

---

## Motivation

A developer working across a microservice-rich environment needs:

1. A **meta-repo** that organises all upstream repos in one place, keeps them
   fresh, and never becomes a place where direct work happens.
2. A way to **group units of work** — sometimes a ticket touches five services,
   sometimes just one. The grouping mechanism must not impose constraints on
   branch naming and must not feel like overkill for simple work.
3. **Flexible infra repo patterns** — two common infrastructure conventions
   must both be supported. Pattern A (branch-per-env): separate long-lived
   branches per environment, each treated as "main" for that env, modeled as
   independent repo aliases with `track_branch`. Pattern B (folder-per-env):
   a single main branch with env-specific subdirectories, where a ticket
   touching multiple envs requires multiple named worktrees from the same repo
   within one workstream, each on its own feature branch targeting different
   scopes.
4. **Planning state that persists** — when an AI coding agent like GSD plans and
   executes work, its `.planning/` state must be version-controlled in the
   meta-repo, not ephemeral, and not visible to individual service repos.
5. **Agent legibility** — AI agents must be able to discover the current
   environment, understand what worktrees are active, know what `git-w` commands
   are available, and understand what the tool handles on their behalf so they
   do not reinvent workspace management.
6. **Multi-destination sync with push protection** — When working on open source
   projects, work in progress must not accidentally reach org remotes. Personal
   backup remotes should mirror everything. The protection must be enforced at
   the git level, not just through `git w sync`.
7. **Discoverability of past work** — when a workstream closes, its planning
   documents and context must remain in the meta-repo in a known, searchable
   location.

---

## v1 to v2 migration

### `git w migrate`

Detects v1 config and directory structure and produces a migration plan.
Safe by default: dry-run is always implicit. Requires `--apply` to execute.

**Milestone note:** `git w migrate` is implemented in Milestone 12, which
depends on Milestone 1 (config schema and types) but can be developed in
parallel with Milestones 2–11. It ships as part of the v2.0 release.

**Detection triggers:**
- Presence of `[[workgroup]]` blocks in `.gitw`
- `workgroups/` directory at workspace root
- Any `[[repo]] path` field not matching `repos/<n>` convention

**Repo path migration cases:**

git-w v1 allowed repos to be organized into arbitrary subdirectories. Common
patterns include `services/`, `libs/`, root-level, or any other structure.
v2 requires all root clones to live under `repos/`. Migration handles five
cases:

| Case | Description | Migration action |
|---|---|---|
| Arbitrary path, directory exists | e.g. `services/api-service/` | `mv` to `repos/api-service/`; update `.gitw` |
| Path at root level | e.g. `api-service/` (no subdirectory) | `mv` to `repos/api-service/`; update `.gitw` |
| Path does not exist on disk | Referenced in `.gitw` but never cloned | Update `.gitw` only; `git w restore` clones later |
| Path collision | Two repos would become `repos/<same-name>` | **Abort with error**; user must rename one in config before migrating |
| Bare repo (no working tree) | Created by v1 `git w worktree clone` | **Flag in report**; require manual resolution; migration refuses to touch bare repos |

**Migration report (always printed):**

```
git-w v1 → v2 migration plan
==============================

Repos to move (3):
  services/api-service   →  repos/api-service      [directory exists]
  services/auth-lib      →  repos/auth-lib          [directory exists]
  libs/payment-lib       →  repos/payment-lib       [directory exists]
  generated-code         →  repos/generated-code    [directory exists, at root]
  missing-service        →  repos/missing-service   [not on disk — config only]

Workgroups to convert (2):
  auth-refactor  →  workspaces/legacy/active/auth-refactor/
    api-service:   workgroups/auth-refactor/api-service
                 →  workspaces/legacy/active/auth-refactor/api-service
    auth-lib:      workgroups/auth-refactor/auth-lib
                 →  workspaces/legacy/active/auth-refactor/auth-lib
  data-pipeline  →  workspaces/legacy/active/data-pipeline/
    ingestion:     workgroups/data-pipeline/ingestion
                 →  workspaces/legacy/active/data-pipeline/ingestion

Path collisions (0):
  (none)

Bare repos detected (0):
  (none)

Config changes:
  [[repo]] path fields updated to repos/<n> convention (5 repos)
  [[workgroup]] blocks removed; [[workspace]] "legacy" block added
  .gitw-stream manifests generated for 2 workstreams
  AGENTS.md stubs generated at meta-repo, workspace, and workstream level

Run 'git w migrate --apply' to execute.
Commit the results manually after review. Nothing has been changed.
```

**Collision error (aborts before any changes):**

```
git-w: migration aborted — path collision detected

Two repos would resolve to the same path under repos/:

  services/api-service  →  repos/api-service
  api-service           →  repos/api-service  ← collision

Rename one repo in [[repo]] name and path fields in .gitw before migrating.
No changes have been made.
```

**Bare repo error (aborts before any changes):**

```
git-w: migration aborted — bare repo detected

  libs/auth-lib/.git/  appears to be a bare git repository

Bare repos were created by 'git w worktree clone' in v1. git-w v2 does not
use bare repos. To resolve:
  1. Identify which worktrees were checked out from this bare repo.
  2. Pick one worktree as the canonical clone and copy it to repos/auth-lib/.
  3. Update [[repo]] path = "repos/auth-lib" in .gitw.
  4. Remove the bare repo directory.
  5. Re-run git w migrate.

No changes have been made.
```

**`--apply` execution sequence:**

Pre-flight checks run first. Any collision or bare repo aborts immediately
before any filesystem changes.

1. **Pre-flight:** detect path collisions and bare repos; abort on either.
2. **Create workspace scaffold:** `workspaces/legacy/active/` and
   `workspaces/legacy/archived/` directories.
3. **Move root clones:** for each repo not at `repos/<n>`:
   - If directory exists: `mv <old-path> repos/<n>/`
   - If directory does not exist: skip move; update config only
4. **Move workgroup worktrees:** for each workgroup entry:
   - `mv workgroups/<n>/<repo>/ workspaces/legacy/active/<n>/<repo>/`
5. **Repair worktrees:** for each repo that had worktrees moved, run
   `git worktree repair` from the new `repos/<n>/` location. This updates
   the worktree's internal `.git` file to reflect the new path.
6. **Update `.gitw`:**
   - Update all `[[repo]] path` fields to `repos/<n>`
   - Remove all `[[workgroup]]` blocks
   - Add `[[workspace]]` block for `legacy`
7. **Generate workstream artifacts:** for each converted workgroup:
   - Write `.gitw-stream` manifest (status=`active`, worktrees from old
     `[[workgroup.worktree]]` entries with updated paths)
   - Generate `AGENTS.md` stub
   - Create empty `.planning/`
8. **Run `git w context rebuild`**
9. **Print commit instructions.** Does not auto-commit.

`git w migrate` is a one-shot upgrade tool. After `--apply` completes
successfully, subsequent runs exit with "no v1 config detected."

### Breaking changes summary

**Config and layout:**

| Change | v1 behavior | v2 behavior |
|---|---|---|
| Workgroups | `[[workgroup]]` blocks, `workgroups/` dir | Retired; use workstreams |
| Repo paths | arbitrary `path` field | always `repos/<n>` |
| Directory layout | flat `workgroups/` at root | `workspaces/<ws>/active/<n>/` hierarchy |

**Top-level commands removed:**

| Removed command | v1 alias | v2 migration |
|---|---|---|
| `git w info` | `ll` | `git w status` |
| `git w fetch` | `f` | `git w sync --no-push` |
| `git w pull` | `pl` | `git w sync --no-push` |
| `git w push` | `ps` | `git w workstream ship` or `git w sync` |
| `git w context` (scope-setter) | `ctx` | pass `--workspace`/`--workstream`/`--repo` at call site |

**Command families removed entirely:**

| Removed family | v1 aliases | v2 migration |
|---|---|---|
| `git w group` (all 6 subcommands) | `g` | `git w workspace create/list`; edit `.gitw` for rare ops |
| `git w workgroup` (all 6 subcommands) | `work`, `wg` | `git w workstream` family |
| `git w worktree` (all 5 subcommands) | `tree`, `t` | `git w workstream worktree add`; `git w workstream close` |

**Subcommands removed within kept families:**

| Removed subcommand | Parent | v2 migration |
|---|---|---|
| `branch default` | `git w branch` | `git w exec checkout <default-branch>` |
| `repo clone` | `git w repo` | merged into `git w repo add <url>` |
| `repo rename` | `git w repo` | edit `.gitw`, rename `repos/<old>` manually |
| `repo unlink` | `git w repo` | remove `[[repo]]` block from `.gitw` |

**Flags renamed:**

| Command | v1 flag | v2 flag |
|---|---|---|
| `git w commit` | `--workgroup` / `-W` | `--workstream` / `-W` |

**`git w context` repurposed:** In v1, `git w context` set the active group
scope. In v2, `git w context rebuild` regenerates `CONTEXT.md` and all
`AGENTS.md` files. These are different commands with different purposes that
share a name only by coincidence. The v1 scope-setting behavior is replaced
by `--workspace`, `--workstream`, and `--repo` filter flags at the call site
of any read command.

---

## Command surface

### Design principles

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

### Full command tree

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

### Complete cut list from v1

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

**Net reduction: 39 → 27 commands/subcommands (−12)**

---

## Disk layout

```
meta-repo/
|
|  .gitw                        <- shared config (committed)
|  .git/.gitw                   <- private local overrides (never committed)
|  .git/git-w-state.json        <- ephemeral runtime state (never committed)
|  AGENTS.md                    <- meta-repo level agent context (committed)
|  CONTEXT.md                   <- system map: repos, workspaces, workstreams
|                                  (committed, regenerated by context rebuild)
|  .planning/                   <- global cross-cutting planning (committed)
|
+-- repos/
|   +-- service-a/
|   +-- service-b/
|   +-- infra-dev/              <- alias: infra repo tracking dev branch
|   +-- infra-test/             <- alias: infra repo tracking test branch
|   +-- infra-prod/             <- alias: infra repo tracking prod branch
|
+-- workspaces/
    +-- payments-platform/
    |   |  AGENTS.md            <- workspace-level agent context (committed)
    |   |  .planning/           <- GSD project-level state (committed)
    |   |
    |   +-- active/
    |   |   +-- TICKET-123/
    |   |       |  .gitw-stream <- workstream manifest (committed)
    |   |       |  AGENTS.md   <- workstream-level agent context (committed)
    |   |       |  .planning/  <- GSD phase-level state (committed)
    |   |       +-- service-a/ <- worktree (ephemeral; not committed)
    |   |       +-- service-b/ <- worktree (ephemeral; not committed)
    |   |
    |   +-- archived/
    |       +-- TICKET-100/
    |           |  .gitw-stream <- status = "archived"
    |           |  AGENTS.md
    |           +-- .planning/  <- preserved in full; no worktrees
    |
    +-- platform-infra/
    |   |  AGENTS.md
    |   |  .planning/
    |   +-- active/
    |   |   +-- INFRA-42/
    |   |       |  .gitw-stream
    |   |       |  AGENTS.md
    |   |       |  .planning/
    |   |       +-- infra-dev/  <- worktree from repos/infra-dev
    |   |       +-- infra-test/ <- worktree from repos/infra-test
    |   |       +-- infra-prod/ <- worktree from repos/infra-prod
    |   +-- archived/
    |
    +-- support/
        +-- active/
        |   +-- BUG-789/
        +-- archived/
```

**Key invariants:**
- Worktrees exist only under `active/`. Directories under `archived/` have no
  worktrees.
- `.planning/` at all three levels is committed to the meta-repo and invisible
  to individual service repos.
- `repos/` contains canonical clones never directly worked in. Worktrees are
  checked out from these clones into workstream directories.
- All committed artifacts (`.planning/`, `AGENTS.md`, `CONTEXT.md`,
  `.gitw-stream`) travel with the meta-repo. Cloning the meta-repo on a new
  machine and running `git w restore` reconstructs the full environment,
  including re-materializing worktrees for active workstreams.

---

## Config file model

| File | Location | Committed? | Purpose |
|---|---|---|---|
| `.gitw` | workspace root | yes (normally) | shared workspace config |
| `.git/.gitw` | inside `.git/` | never | private local overrides |

**Merge semantics:** field-level merge; `.git/.gitw` wins on conflicts.
`[[remote]]` merged by `name`. `[[sync_pair]]` merged by `(from, to)` pair.
`[[repo]]` merged by `name`. `[[workspace]]` merged by `name`. `[[workstream]]`
root blocks merged by `name`. `.gitw-stream` files are self-contained and not
merged at the root config level.

**Privacy enforcement:** `private = true` remotes rejected at load time if found
in `.gitw`. Must live in `.git/.gitw`.

---

## Schema

### `[metarepo]` top-level block

```toml
[metarepo]
name                = "platform-work"
default_remotes     = ["origin", "personal"]
agentic_frameworks  = ["gsd"]
# The spec-driven agentic frameworks active in this meta-repo.
# A slice — multiple frameworks may be present (e.g. ["gsd", "speckit"]).
# Valid values at v2 launch: "gsd"
# Omitting defaults to ["gsd"]. Unknown values are a load-time error.
# See .planning/v2/v2-agent-interop.md for the SpecFramework interface design.
```

### `[[workspace]]` named workspace block

```toml
[[workspace]]
name        = "payments-platform"
description = "Payment processing and related services"
repos       = ["api-service", "payment-lib", "gateway"]
```

### `[[repo]]` block with v2 additions

```toml
[[repo]]
name         = "infra-dev"
path         = "repos/infra-dev"       # always repos/<n> in v2
clone_url    = "https://github.com/work-org/infra"
track_branch = "dev"    # branch treated as "main" for this alias.
                         # Used for clone --branch and sync pull target.
                         # Omit to use remote HEAD (standard repos).
upstream     = "infra"  # logical grouping name for aliases sharing a clone_url.
                         # Display metadata only; no git behavior change.
                         # git w status --repo infra matches all aliases.
                         # git w workstream create --env-group infra expands
                         # to all repos with this upstream value.
remotes      = ["origin", "personal"]  # override [metarepo] default_remotes
```

### `.gitw-stream` manifest

Lives at `workspaces/<workspace>/active/<n>/.gitw-stream`. Committed to the
meta-repo. Machine-readable source of truth for a workstream.

The `[[worktree]]` array is the core of the manifest. Two constraints apply:
- `name` must be unique within the workstream (it is the primary key)
- `path` must be unique within the workstream (it is the on-disk key)

When a repo appears only once in a workstream, `name` defaults to the repo
name and may be omitted. When the same repo appears more than once (Pattern B
consolidated infra), `name` is required on all entries for that repo and must
be set explicitly.

```toml
name        = "INFRA-42"
description = "Add RDS config to all environments"
workspace   = "platform-infra"
status      = "active"        # "active" | "shipped" | "archived"
created     = "2026-03-15"

# --- Pattern A: branch-per-env (repo aliases, one worktree per alias) ---
# Expansion of --env-group infra; always stored as explicit entries.
# name defaults to repo name (each alias is a distinct repo).

[[worktree]]
repo   = "infra-dev"
branch = "feat/INFRA-42-new-rds"
path   = "infra-dev"
# name omitted: defaults to "infra-dev"

[[worktree]]
repo   = "infra-test"
branch = "feat/INFRA-42-new-rds"
path   = "infra-test"

[[worktree]]
repo   = "infra-prod"
branch = "feat/INFRA-42-new-rds"
path   = "infra-prod"

# --- Pattern B: folder-per-env (one repo, multiple named worktrees) ---
# consolidated-infra has one main branch; envs are subdirectories.
# Two worktrees from the same repo require explicit name, path, and scope.

# [[worktree]]
# name   = "dev"                        # required; unique within workstream
# repo   = "consolidated-infra"
# branch = "feat/TICKET-123-dev"        # merges into main; touches environments/dev/
# path   = "consolidated-infra-dev"     # unique on-disk path within workstream dir
# scope  = "environments/dev"           # advisory: expected modification subdirectory
#                                        # surfaced in AGENTS.md and --json; no git
#                                        # enforcement. agents must respect it.
#
# [[worktree]]
# name   = "prod"
# repo   = "consolidated-infra"
# branch = "feat/TICKET-123-prod"       # merges into main; touches environments/prod/
# path   = "consolidated-infra-prod"
# scope  = "environments/prod"

[ship]
pr_urls             = []
pre_ship_branches   = {}   # worktree-name -> "branch-name-pre-ship-<timestamp>" on personal
shipped_at          = ""

[context]
summary       = ""
key_decisions = []
```

**`name` field:** unique key within the workstream. Used in status output
(`consolidated-infra / dev`), hook path resolution, and `AGENTS.md` worktree
table. For single-occurrence repos, defaults to the repo name. For
multi-occurrence repos (Pattern B), required and must be set explicitly on all
entries for that repo. Validated unique at creation time.

**`path` field:** relative path from the workstream directory to the worktree
on disk. Must be unique within the workstream. When `name` is explicitly set
and `path` is omitted, `path` defaults to `name`. For Pattern A, typically
the alias name. For Pattern B, typically `<repo>-<name>` to avoid collision.

**`scope` field:** optional advisory metadata. The subdirectory within the
repo that this worktree is expected to modify. Not enforced by any git
operation. Surfaced in the workstream `AGENTS.md` worktree table and in
`git w agent context --json`. Tells an agent "this worktree should only touch
`environments/prod/`." Agents must respect scope boundaries; git-w does not
enforce them mechanically.

### `[[workstream]]` block in root `.gitw` (lightweight remote override only)

The root config does not contain full workstream definitions. It may contain
per-workstream remote overrides in the private config:

```toml
# In .git/.gitw only
[[workstream]]
name    = "INFRA-42"
remotes = ["personal"]    # replaces workspace defaults for this workstream's worktrees
```

### `[[remote]]` block

```toml
[[remote]]
name        = "personal"
kind        = "gitea"             # "gitea" | "forgejo" | "github" | "generic"
url         = "https://gitea.example.com"
user        = "youruser"
token_env   = "GITWBU_GITEA_TOKEN"
org         = ""
repo_prefix = "work-"
repo_suffix = ""
direction   = "push"              # "push" | "fetch" | "both"
push_mode   = "mirror"            # "mirror" | "branch"
fetch_mode  = "all"               # "all" | "tracked"
use_ssh     = false
ssh_host    = ""
critical    = false
private     = true
```

### `[[remote.branch_rule]]`

```toml
[[remote.branch_rule]]
pattern   = "wip/*"
untracked = true
explicit  = false
action    = "block"               # "allow" | "block" | "warn" | "require-flag"
reason    = "WIP branches must not be pushed to org"
flag      = "--push-wip"          # for require-flag only
```

Rule evaluation order: repo `[[repo.branch_override]]` first (declaration
order), then remote `[[remote.branch_rule]]` (declaration order), then default
`allow`.

### `[[sync_pair]]` block

```toml
[[sync_pair]]
from = "origin"
to   = "personal"
refs = ["**"]       # ** crosses /; * does not
```

Cycle detection at load time. Fan-out: fetch all `from` remotes in parallel,
then push to `to` remotes filtered by `refs` and subject to branch rule
evaluation.

---

## Infrastructure repo patterns

Two common conventions for organizing infrastructure repos are both supported
as first-class patterns in git-w.

### Pattern A: branch-per-environment (`infra`)

One upstream repo, one branch per deployed environment. Each branch is treated
as "main" for that environment. Promotion between environments is a PR/merge
on the upstream repo — not something git-w orchestrates.

git-w models this as **repo aliases**: multiple `[[repo]]` entries sharing the
same `clone_url` but each tracking a different `track_branch`. They are
independent repos from git-w's perspective — separate directories under
`repos/`, separate worktrees in workstreams, separate entries in status output.

```
github.com/work-org/infra
  branch: dev   →  repos/infra-dev   (track_branch = "dev",  upstream = "infra")
  branch: test  →  repos/infra-test  (track_branch = "test", upstream = "infra")
  branch: prod  →  repos/infra-prod  (track_branch = "prod", upstream = "infra")
```

A workstream touching all environments uses `--env-group infra` to create one
worktree per alias in a single command.

### Pattern B: folder-per-environment (`consolidated-infra`)

One upstream repo, one main branch. Environment-specific configuration lives
in subdirectories:

```
github.com/work-org/consolidated-infra  (branch: main)
  environments/
    dev/
    test/
    prod/
```

A ticket touching dev and prod simultaneously requires two feature branches —
`feat/TICKET-123-dev` (modifies `environments/dev/`, merges to main) and
`feat/TICKET-123-prod` (modifies `environments/prod/`, merges to main at a
different time). These are two worktrees of the same repo in the same
workstream.

git-w models this as **multiple named worktrees** per repo within a workstream.
The `name`, `path`, and `scope` fields on `[[worktree]]` entries distinguish
them.

### `git w repo add` for Pattern A aliases

**Single alias:**
```
git w repo add https://github.com/work-org/infra --name infra-dev --branch dev
git w repo add https://github.com/work-org/infra --name infra-test --branch test
git w repo add https://github.com/work-org/infra --name infra-prod --branch prod
```

**Multi-alias shorthand:**
```
git w repo add https://github.com/work-org/infra \
  --branch-map dev:infra-dev test:infra-test prod:infra-prod
```

`--branch-map` takes one or more `<branch>:<alias-name>` pairs. Creates all
`[[repo]]` blocks and clones all directories in one invocation. The `upstream`
field is set automatically from the URL basename (`infra` from `.../infra`),
overridable with `--upstream-name <n>`.

Pattern B repos (`consolidated-infra`) are added with a standard `git w repo
add` — no special flags. The multi-worktree behavior is configured at workstream
creation time, not at repo registration time.

### Sync behavior for Pattern A aliases

`git w sync` on an alias repo fetches all refs from the remote, pulls
`track_branch` into the local checkout, and mirror-pushes all refs to a single
personal backup repo named after the upstream (e.g., `infra`). All aliases that
share the same upstream mirror-push to that one repo, so all branches
(`work-infra-dev`, `work-infra-test`, `work-infra-prod`) are preserved together
in a single 1-to-1 mirror of the upstream — not as separate backup repos per alias.

### `--env-group` in workstream create (Pattern A)

```
# All infra envs — branch defaults to workstream name
git w workstream create platform-infra INFRA-42 \
  --description "Add RDS config to all environments" \
  --env-group infra

# All infra envs with explicit branch name
git w workstream create platform-infra INFRA-43 \
  --env-group infra:feat/INFRA-43-secret-rotation

# One env only (use explicit --repo)
git w workstream create platform-infra INFRA-55 \
  --repo infra-dev:feat/INFRA-55-experiment

# Mixed: all infra envs + another repo
git w workstream create platform-infra INFRA-60 \
  --env-group infra:feat/INFRA-60-deploy \
  --repo k8s-config:feat/INFRA-60-deploy
```

`--env-group <upstream>` resolves all `[[repo]]` blocks with
`upstream = "<upstream>"` and creates one `[[worktree]]` entry per alias in the
`.gitw-stream`. Branch defaults to the workstream name if not specified.
Expansion is always stored as explicit `[[worktree]]` entries — no runtime
resolution needed. `--repo` and `--env-group` are composable.

### `--worktree` flag in workstream create (Pattern B)

For consolidated repos requiring multiple worktrees, use the `--worktree` flag
with key=value pairs:

```
git w workstream create platform-infra TICKET-123 \
  --description "Update dev and prod consolidated infra" \
  --worktree name=dev,repo=consolidated-infra,branch=feat/TICKET-123-dev,scope=environments/dev \
  --worktree name=prod,repo=consolidated-infra,branch=feat/TICKET-123-prod,scope=environments/prod
```

`--worktree` keys:
```
name    required  unique label within the workstream
repo    required  repo name from [[repo]] config
branch  required  feature branch to create
path    optional  on-disk directory name; defaults to name
scope   optional  advisory subdirectory hint for agents
```

**`path` defaults to `name`** when omitted. For Pattern B, `name=dev` produces
`path=dev` and the worktree lands at `workspaces/<ws>/active/<n>/dev/`.
Override with `path=consolidated-infra-dev` if a more descriptive directory
name is preferred.

**Error when `--repo` is used twice for the same repo without disambiguation:**
```
Error: repo "consolidated-infra" appears more than once.
Use --worktree to specify name and scope for each:
  --worktree name=dev,repo=consolidated-infra,branch=feat/TICKET-123-dev,scope=environments/dev
  --worktree name=prod,repo=consolidated-infra,branch=feat/TICKET-123-prod,scope=environments/prod
```

`--repo` and `--worktree` are composable. A workstream can mix simple repos
(`--repo api-service:feat/TICKET-123`) with named worktrees
(`--worktree name=dev,repo=consolidated-infra,...`).

### `git w workstream worktree add` for Pattern B

To add a named worktree to an existing workstream post-creation:

```
git w workstream worktree add consolidated-infra \
  --branch feat/TICKET-123-test \
  --worktree-name test \
  --scope environments/test
```

`--worktree-name` and `--scope` are the additional flags on `worktree add` for
the multi-worktree case. `--worktree-name` is required when the repo already
has a worktree in the workstream.

### `git w workstream status` output for Pattern B

The display format `<repo> / <name>` is used when a repo has multiple worktrees
in the workstream. Single-occurrence repos display as just `<repo>`.

```
workstream: TICKET-123 (platform-infra)

  consolidated-infra / dev    feat/TICKET-123-dev   [scope: environments/dev]
    M  environments/dev/rds.tf
    M  environments/dev/variables.tf

  consolidated-infra / prod   feat/TICKET-123-prod  [scope: environments/prod]
    [clean]
```

### `git w agent context --json` for Pattern B

```json
{
  "agentic_frameworks": ["gsd"],
  "workstream": {
    "worktrees": [
      {
        "name": "dev",
        "repo": "consolidated-infra",
        "branch": "feat/TICKET-123-dev",
        "path": "dev",
        "scope": "environments/dev",
        "abs_path": "/Users/robert/platform/workspaces/platform-infra/active/TICKET-123/dev"
      },
      {
        "name": "prod",
        "repo": "consolidated-infra",
        "branch": "feat/TICKET-123-prod",
        "path": "prod",
        "scope": "environments/prod",
        "abs_path": "/Users/robert/platform/workspaces/platform-infra/active/TICKET-123/prod"
      }
    ]
  }
}
```

### Workstream `AGENTS.md` for Pattern B

The worktree table gains `Name` and `Scope` columns. A warning note is
generated when a repo appears more than once:

```markdown
## Worktrees

| Name | Repo               | Branch                  | Path | Scope             |
|------|--------------------|-------------------------|------|-------------------|
| dev  | consolidated-infra | feat/TICKET-123-dev     | dev  | environments/dev  |
| prod | consolidated-infra | feat/TICKET-123-prod    | prod | environments/prod |

**Important:** `consolidated-infra` has two worktrees in this workstream — one
per environment branch. The `dev` worktree must only modify files under
`environments/dev/`. The `prod` worktree must only modify files under
`environments/prod/`. Do not cross-modify between worktrees. Each branch will
merge into `main` independently at different times.
```

### `git w workstream ship` for Pattern B

`--open-prs` opens one PR per `[[worktree]]` entry — so a Pattern B workstream
with two worktrees of `consolidated-infra` opens two PRs against the same repo.
Each PR description notes the scope: "Part of TICKET-123 — covers
`environments/dev/` only." Cross-linking between the two PRs is deferred
(post-v2).

### `git w status --repo consolidated-infra`

Shows the root clone at `repos/consolidated-infra` plus all named worktrees
of that repo across all active workstreams, with their `name` and `scope`:

```
-- repo: consolidated-infra --
  root clone   main   repos/consolidated-infra   [clean]

  Active worktrees:
    TICKET-123 / dev   feat/TICKET-123-dev   environments/dev   [M 2 files]
    TICKET-123 / prod  feat/TICKET-123-prod  environments/prod  [clean]
```

### `git w agent context --json` env-group data (Pattern A)

```json
{
  "agentic_frameworks": ["gsd"],
  "env_groups": [
    {
      "name": "infra",
      "upstream_url": "https://github.com/work-org/infra",
      "aliases": [
        { "name": "infra-dev",  "track_branch": "dev" },
        { "name": "infra-test", "track_branch": "test" },
        { "name": "infra-prod", "track_branch": "prod" }
      ],
      "create_hint": "git w workstream create <workspace> <n> --env-group infra"
    }
  ]
}
```

The `create_hint` field gives agents the correct incantation without requiring
them to enumerate aliases manually. Pattern B repos have no env-group entry —
their multi-worktree structure is expressed through the `worktrees` array with
`scope` fields, which is sufficient for agent orientation.

---

## Workstream push protection

Identical mechanism to the workgroup push protection described in
`.planning/v2/v2-remote-management.md`, with `workgroup` replaced by `workstream`
throughout and worktree discovery using `.gitw-stream` manifest files instead of
`[[workgroup.worktree]]` entries in root config.

`reconcileHooks(repo RepoConfig)` installs and manages a `pre-push` hook on
each repo whose worktrees appear in any protected workstream. The hook calls
`git-w hook pre-push "$@"`.

**Install triggers:**
- `git w workstream create` — installs hooks on all repos with worktrees.
- `git w sync` — calls `reconcileHooks` on all repos as a self-healing side
  effect.

**Hook execution logic:** walks up from `$GIT_DIR` to workspace root, resolves
current worktree path, searches `.gitw-stream` `[[worktree]]` entries across all
active workstreams for a path match, checks remote name against the matched
workstream's `remotes` whitelist.

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

**Lifting protection:** `git w workstream ship` adds `origin` to the
workstream's `remotes` in `.git/.gitw` and calls `reconcileHooks`. Alternatively:
edit `[[workstream]] remotes` in `.git/.gitw` manually, then run `git w sync`
(the `reconcileHooks` side effect picks up the change automatically).

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
execute. Does not auto-commit results.

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

When the same repo appears more than once (Pattern B), the workstream
`AGENTS.md` gains a cross-modification warning for each such repo, explicitly
listing which worktree must stay within which `scope`.

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
      `<branch>-pre-ship-<timestamp>` (e.g.,
      `feat/INFRA-42-secret-rotation-pre-ship-20260401`). This branch is pushed
      to `personal` only and is never synced back to `origin`. Its purpose is to
      preserve the full messy commit history before the squash.
   c. Records the backup branch name in `.gitw-stream` under a new
      `[ship] pre_ship_branches` array keyed by worktree name.
   d. Prompts the user for a single squash commit message for that worktree
      branch (pre-filled with the first commit subject in the range as a
      starting point).
   e. Performs a soft-reset to the divergence point and commits with the
      provided message, producing one clean commit on the local branch.
   f. Repeats steps a–e for each remaining worktree with unpushed commits.
   The squash pass is skipped for any worktree whose branch is already clean
   (no commits ahead of `origin/<branch>`). `--squash` forces the flow even on
   clean branches (useful when the branch has no upstream tracking ref yet).
3. Lifts push protection: adds `origin` to `[[workstream]] remotes` in
   `.git/.gitw`, calls `reconcileHooks` on all repos.
4. `--push-all`: pushes all worktree branches to `origin` via scoped sync.
5. `--open-prs`: opens one PR per worktree branch on the configured GitHub
   remote. PR URLs written to `.gitw-stream [ship] pr_urls`. Records
   `shipped_at`.
6. Updates workstream status to `"shipped"` in `.gitw-stream`.
7. Commits updated `.gitw-stream`.

For env-group workstreams (Pattern A), one PR is opened per alias branch.
For consolidated-infra workstreams (Pattern B), one PR is opened per named
worktree entry even when multiple entries share the same underlying repo.
Each PR description notes the scope if set. `--open-prs` requires a
`kind = "github"` remote configured for the affected repos.

**Pre-ship backup branches** on the personal remote are intentionally never
included in `[[sync_pair]]` push rules and are never forwarded to `origin`.
They exist solely as a safety net and can be pruned manually after the PR is
merged.

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
remote (e.g., a personal mirror) without a separate `git w sync` step. When
`--from` is omitted the branch is created from the local HEAD or current
tracking state.

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

## Agent interop

### Philosophy

git-w is opinionated about conventions, not implementations. It does not know
any spec framework's internals. Spec frameworks do not know git-w's internals.
They compose because they agree on directory structure and because git-w's
generated `AGENTS.md` files declare capabilities explicitly enough that agents
do not reinvent what git-w already handles.

The agent interop layer is designed to support **multiple spec frameworks**
(GSD, speckit, openspec, or any future tool). The active framework is declared
in `[metarepo] agentic_frameworks` in `.gitw`. At v2 launch, only `"gsd"` is
supported. Framework-specific behavior is isolated behind the `SpecFramework`
Go interface in `pkg/agents` — adding support for a new framework requires only
implementing that interface. See `.planning/v2/v2-agent-interop.md` for the full
interface design and registry contract.

Explicit prohibitions with alternatives ("do not do X, instead call Y") are
more reliable than vague guidance.

### Three-level `AGENTS.md` strategy

**Meta-repo `AGENTS.md`** — "what is this environment?"

Critical section always present verbatim (git-w-owned, framework-invariant):

```markdown
## What git-w manages — do not do these manually

- **Worktree lifecycle**: do not run `git worktree add/remove` directly.
  Use `git w workstream create` and `git w workstream close`.
- **Push protection**: pre-push hooks enforce allowed remotes during WIP.
  Do not attempt to bypass hooks or force-push to origin from a workstream.
- **Sync**: do not push directly to remotes. Use `git w sync` or
  `git w workstream ship`.
- **Workspace creation**: do not use GSD's `/gsd:new-workspace`,
  `/gsd:new-project` workspace scaffolding, or any equivalent command from
  another tool. git-w creates workstreams; GSD initializes `.planning/` inside
  them via `/gsd:new-project` pointed at the workstream directory.
- **PR creation**: do not open PRs directly.
  Use `git w workstream ship --open-prs`.
```

Framework-specific prohibition items (e.g. workspace-creation command names)
are generated by the active `SpecFramework` implementation and appended to the
above. The git-w-owned prohibitions are always present regardless of framework.

Also contains: all available git-w commands with descriptions, workspace and
active workstream list, env-group summary.

**Workspace `AGENTS.md`** — "what is this product area?"

Contains workspace description, repo membership, env-group summary for any
upstream repos, active workstreams with goals and repo/branch breakdown,
archived workstreams (names and paths), conventions stub.

**Workstream `AGENTS.md`** — "what is my current task?"

Contains workstream name, workspace, goal, status; worktree table (repo, `name`
if multi-worktree, `track_branch` if alias, feature branch, `scope` if set,
relative path); `.planning/` path and framework initialization instructions
(from the active `SpecFramework`); how to work across multiple worktrees in one
session; explicit "when done, call `git w workstream ship`; then call
`git w workstream close`"; explicit "do not open PRs directly."

When a repo appears more than once in the workstream (Pattern B), the table
includes a `Scope` column and a cross-modification warning block is appended:

```markdown
**Important:** `consolidated-infra` has multiple worktrees in this workstream.
Each worktree must only modify files within its declared scope:
- `dev` (feat/TICKET-123-dev): modify only `environments/dev/`
- `prod` (feat/TICKET-123-prod): modify only `environments/prod/`
Do not cross-modify between worktrees. Each branch merges into `main`
independently.
```

### GSD interop specifics (v2 launch framework)

When GSD is invoked inside a workstream directory it sees:

```
workspaces/platform-infra/active/INFRA-42/
  AGENTS.md       <- GSD reads this; knows git-w manages worktrees and push
  .planning/      <- GSD reads/writes here
  .gitw-stream    <- GSD can read for context; does not write
  infra-dev/      <- GSD executor works here (normal repo directory)
  infra-test/     <- GSD executor works here
  infra-prod/     <- GSD executor works here
```

GSD's `/gsd:new-project` skips workspace scaffolding because `.planning/`
already exists and `AGENTS.md` explicitly prohibits GSD from creating
workspaces, worktrees, or opening PRs.

GSD manages the planning lifecycle inside the workstream. git-w manages the git
lifecycle. They compose through directory convention and the AGENTS.md contract,
not API coupling.

This behavior is entirely contained in `GSDFramework` (the `SpecFramework`
implementation for GSD). A future framework adds its own implementation without
touching any other code.

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

## State file

**Path:** `.git/git-w-state.json` — machine-local, never committed.

Tracks: per-repo per-remote last push/fetch timestamps and remote HEAD SHA;
active workstream pointer; hook installation state per repo.

---

## Implementation notes

All v1 Go conventions apply unchanged: `pkg/` layout, `go-toml/v2` via
`pkg/toml` wrapper with `UpdatePreservingComments`, Cobra with pflag (no Viper),
Mage for builds, `pkg/output.Writef`, `pkg/cmdutil.ResolveBoolFlag`, atomic
config writes, `Register(root *cobra.Command)` per command, `go 1.26.0`.

### New internal packages

**`pkg/workspace`**
- `ResolveWorkstream(cwd, stateFile string) (*Workstream, error)`
- `LoadManifest(path string) (*GitwStream, error)`
- `WriteManifest(path string, m *GitwStream) error`
- `ResolveEnvGroup(cfg Config, upstream string) ([]RepoConfig, error)`

**`pkg/agents`**
- `GenerateMetaRepoAgentsMD(cfg Config, state State) string`
- `GenerateWorkspaceAgentsMD(cfg Config, ws Workspace, state State) string`
- `GenerateWorkstreamAgentsMD(cfg Config, stream Workstream) string`
- `GenerateContextMD(cfg Config, state State) string`
- All functions are pure (no file I/O).

**`pkg/worktrees`**
- `Add(repoPath, worktreePath, branch string) error`
- `Remove(worktreePath string, force bool) error`
- `List(repoPath string) ([]WorktreeInfo, error)`
- `Repair(worktreePath string) error`
- `ReconcileHooks(repoPaths []string, cfg Config) error`

### `--json` output contract

All read commands emit a single JSON object to stdout. Errors to stderr.
Exit codes: 0 = success, 1 = user error, 2 = git/system error.

### Branch rule engine

`BranchInfo`: `Name string`, `HasUpstreamOn func(remoteName string) bool`,
`ExplicitOn func(remoteName string) bool`.
`EvaluateRule(branch BranchInfo, rules []BranchRule, remoteName string) (Action, *BranchRule)` — pure function.
Internal glob: `*` no `/` crossing; `**` crosses `/`.
Table-driven tests: all criteria combinations x all action tiers.

### `reconcileHooks`

Full specification in `.planning/v2/v2-remote-management.md`. Managed block
delimiters (exact strings):
```
# --- git-w managed block (do not edit) ---
# --- end git-w managed block ---
```
Idempotent. Removes block when no workstream references a repo's worktrees.
Cleans up empty hook files.

---

## Milestones

> **For the authoritative list of GitHub issue numbers, issue titles, exact branch names, and per-milestone issue assignments, see `.planning/v2/v2-issue-map.md`.** The milestone descriptions below are design scope only — use the issue map for all branching and planning decisions.

**Dependency note:** Milestones are numbered by logical dependency order, not
required execution order. Milestone 12 (`git w migrate`) depends only on
Milestone 1 being code-complete (types and loader exist) and can be developed
in parallel with Milestones 2–11. All milestones ship together as v2.0.

### Milestone 1 — v2 config schema + loader

- `[[workspace]]` block parsing, validation, field-level merge
- `[[repo]]` v2 additions: `track_branch`, `upstream` fields
- `[[repo]] path` convention: must be `repos/<n>`; load-time warning for v1
  paths with suggestion to run `git w migrate`
- `.gitw-stream` manifest parsing, validation, atomic write:
  - `name` field on `[[worktree]]`: unique-within-workstream key; defaults to
    repo name for single-occurrence repos; required when same repo appears more
    than once (Pattern B)
  - `path` field: unique on-disk path within workstream directory; defaults to
    `name` when not explicitly set
  - `scope` field: optional advisory string; surfaced in AGENTS.md and JSON;
    no git enforcement
  - Load-time validation: `name` unique within workstream, `path` unique within
    workstream; same repo appearing without distinct `name` values is a
    load-time error with actionable message suggesting `--worktree` flag
- `[[remote]]` block (from v1 remote spec)
- `[[remote.branch_rule]]` (from v1 remote spec)
- `[[sync_pair]]` with cycle detection (from v1 remote spec)
- `[[workstream]]` root config block (lightweight remote override)
- Two-file merge with field-level semantics
- `private = true` enforcement
- `[metarepo] default_remotes` cascade resolution
- Load-time detection of v1 `[[workgroup]]` blocks: actionable error
  message directing user to run `git w migrate` (detection only; no migration
  logic in this milestone)
- `UpdatePreservingComments` round-trip tests for all new fields
- Full unit tests: merge, cascade, cycle detection, v1 `[[workgroup]]`
  detection

### Milestone 2 — branch rule engine

- `BranchInfo` type
- `EvaluateRule` pure function
- Internal glob package
- `untracked` criterion via go-git
- `explicit` criterion from `[[repo.branch_override]]`
- All four action tiers
- Table-driven tests

### Milestone 3 — `git w sync` multi-remote + fan-out

- Cascade resolution: workspace -> workstream -> repo effective remote list
- `track_branch` as pull target for alias repos
- `[[sync_pair]]` fan-out with errgroup parallel execution
- Branch rule evaluation per remote
- `critical` flag semantics
- `reconcileHooks` side effect on every sync
- `--remote`, `--workspace`, `--workstream`, `--no-push`, `--push-wip`,
  `--dry-run` flags
- Per-remote grouped output with summary line
- State file writes

### Milestone 4 — `git w remote` subcommand

- `git w remote list` with `--json`
- `git w remote add` interactive wizard and non-interactive flags
- Gitea/Forgejo API provider
- GitHub API provider
- Generic no-op provider
- `gitw-<n>` upsert on child repos including alias repos
- Optional initial mirror push and `[[sync_pair]]` creation
- `git w remote status`

### Milestone 5 — `git w status` + `git w branch checkout --from`

- Unified status merging v1 `info` and `status`
- Env-group display: aliases grouped under upstream name with `(env)` annotation
  in workstream section
- `--workspace`, `--workstream`, `--repo` filter flags
- `--repo` with upstream name: matches all aliases, grouped output
- Remote staleness section from state file
- Available-branch hints
- `git w branch checkout <branch> --from <remote>`
- `--json` output

### Milestone 6 — workstream push protection

- `reconcileHooks` internal function (v1 spec adapted for workstream paths)
- `git w workstream create` calls `reconcileHooks` on all repos with worktrees
- `git w sync` calls `reconcileHooks` as self-healing side effect
- `git-w hook pre-push` subcommand: workstream path resolution, remote whitelist
  check
- `git w workstream ship` lifts protection and calls `reconcileHooks`
- Unit tests: install, append, update, remove, cleanup
- Integration test: direct `git push` from protected worktree is blocked

### Milestone 7 — workspace and workstream lifecycle

- `pkg/workspace` and `pkg/worktrees` packages
- `git w workspace create` with directory scaffolding and `AGENTS.md` stub
- `git w workspace list` with `--json`
- `git w workstream create` with:
  - `.gitw-stream` write, `AGENTS.md` generation, `.planning/` creation
  - `--repo <n>:<branch>` worktree creation (name defaults to repo name)
  - `--worktree name=<n>,repo=<r>,branch=<b>[,path=<p>][,scope=<s>]` explicit
    worktree spec for Pattern B (same repo, multiple named worktrees)
  - `--env-group <upstream>[:<branch>]` expansion via `ResolveEnvGroup`
  - `--repo`, `--worktree`, and `--env-group` all composable in one command
  - Validation: `name` unique within workstream; `path` unique within workstream;
    duplicate repo without unique `name` is an error with `--worktree` hint
  - Cross-modification warning in `AGENTS.md` when same repo appears multiple times
  - `reconcileHooks` per affected repo
  - Auto-commit of `.gitw-stream`, `AGENTS.md`, `.planning/`
- `git w workstream list`, `status` (with `<repo> / <n>` display for multi-worktree
  repos), `switch`
- `git w workstream worktree add` with `--worktree-name` and `--scope` flags
  for Pattern B additions
- `git w restore` re-materializes missing worktrees via `git worktree repair`
- `active/` and `archived/` directory convention enforced

### Milestone 8 — infra repo patterns (Pattern A aliases + Pattern B multi-worktree)

**Pattern A — branch-per-env (repo aliases):**
- `track_branch` field: `--branch` and `--branch-map` flags on `git w repo add`
- `upstream` field: auto-set by `--branch-map`; `--upstream-name` override
- `git w repo list --upstream <n>` filter
- `git w status --repo <upstream>` grouped display for alias sets
- `ResolveEnvGroup` in `pkg/workspace`
- `--env-group` in `git w workstream create`
- Sync: `track_branch` as pull target per alias
- Mirror push: aliases get independent personal remote repos
- `env_groups` array in `git w agent context --json` with `create_hint`

**Pattern B — folder-per-env (named worktrees from same repo):**
- `name`, `path`, `scope` fields parsed and validated in `.gitw-stream`
- `--worktree` flag parsing in `git w workstream create` (key=value pairs)
- `--worktree-name` and `--scope` flags on `git w workstream worktree add`
- `path` defaults to `name` when omitted
- Duplicate-repo validation at creation time with actionable error message
- `<repo> / <n>` display format in `git w workstream status` for multi-worktree repos
- `scope` surfaced in `git w agent context --json` per worktree entry
- Cross-modification warning block generated in workstream `AGENTS.md`
- `git w status --repo <n>` shows named worktrees with `name` and `scope`
- `--open-prs` in ship opens one PR per worktree entry (one per branch, even
  when multiple entries share the same underlying repo)

**Shared:**
- Unit tests: `name`/`path` uniqueness enforcement; Pattern A and B manifest
  round-trips; `--worktree` flag parsing; `--env-group` expansion; `ResolveEnvGroup`

### Milestone 9 — agent context layer

- `pkg/agents` package with pure generator functions
- `git w context rebuild`: `CONTEXT.md`, three-level `AGENTS.md` generation,
  env-group summary, auto-commit
- `git w agent context` with CWD-based scope resolution and `--json`
- `capabilities`, `commands`, `env_groups` blocks in JSON output
- `create_hint` field per env group
- `scope` field per worktree entry in `worktrees` JSON array
- `name` field per worktree entry (present always; equals repo name for
  single-occurrence repos)
- Cross-modification warning block generated in workstream `AGENTS.md` when
  any repo appears more than once; warning lists each `name`, `branch`, and
  `scope` explicitly
- Tests: generator functions produce required prohibition strings, command
  references, env-group create_hints, and scope-boundary warnings for Pattern B

### Milestone 10 — ship pipeline

- `git w workstream ship`: dirty check, optional interactive squash pass
  (`--squash`), pre-ship backup branch creation on personal remote,
  `pre_ship_branches` recording in `.gitw-stream`, push protection lift,
  `--push-all`, `--open-prs` (one PR per worktree branch), URL recording,
  `shipped_at`, status update, auto-commit, `--dry-run`

### Milestone 11 — close and archival

- `git w workstream close`: shipped check, worktree removal, hook cleanup,
  branch pruning prompts, `mv` to archived, status update, `AGENTS.md` update,
  `.planning/` preservation, auto-commit, context rebuild
- `--no-archive` with explicit confirmation

### Milestone 12 — `git w migrate` (parallelizable after M1)

**Dependency:** Milestone 1 must be code-complete (config types and loader
exist). Can be developed concurrently with Milestones 2–11.

**`pkg/migrate` package:**
- `DetectV1(cfg Config, workspaceRoot string) (*MigrationPlan, error)`
  — scans for `[[workgroup]]` blocks, non-`repos/<n>` repo paths, bare repos,
  path collisions; produces a complete plan without touching the filesystem
- `ReportPlan(plan *MigrationPlan) string` — formats the human-readable report
- `ApplyPlan(plan *MigrationPlan, cfg Config, workspaceRoot string) error`
  — executes migration in the correct sequence with pre-flight abort on errors

**Detection and pre-flight (always runs, no filesystem changes):**
- Scan `[[repo]]` blocks for paths not matching `repos/<n>`:
  - Classify each as: directory exists, directory missing, or bare repo
  - Detect name collisions: two repos that would map to same `repos/<n>` target
- Scan `[[workgroup]]` blocks and associated `[[workgroup.worktree]]` entries
- Scan `workgroups/` directory for orphaned worktree directories not in config
- Build complete `MigrationPlan` struct (all moves, config updates, worktree
  repairs needed)

**Pre-flight abort conditions (abort before any filesystem changes):**
- Path collision: two repos would produce the same `repos/<n>` target
  — error names both repos and instructs user to rename one before migrating
- Bare repo detected: `[[repo]] path` points to a bare git repo
  — error names the bare repo, explains the v1 `worktree clone` origin,
  provides step-by-step manual resolution instructions

**`--apply` execution sequence:**
1. Run pre-flight; abort on collision or bare repo
2. Create `workspaces/legacy/active/` and `workspaces/legacy/archived/`
3. Move root clones to `repos/<n>/` (skip if directory does not exist on disk)
4. Move workgroup worktree directories to new workstream paths
5. Run `git worktree repair` from each affected `repos/<n>/` (fixes internal
   `.git` file pointer after directory move)
6. Update `.gitw`: all `[[repo]] path` fields, remove `[[workgroup]]` blocks,
   add `[[workspace]]` block for `legacy`
7. Generate `.gitw-stream`, `AGENTS.md`, empty `.planning/` for each converted
   workstream
8. Run `git w context rebuild`
9. Print commit instructions (does not auto-commit)

**`git w migrate` command:**
- Default (no flags): run detection, print report, exit cleanly
- `--apply`: run detection, print report, execute if no pre-flight errors
- `--dry-run`: explicit alias for the default behavior; same as no flags
- After successful `--apply`: subsequent runs exit cleanly with
  "no v1 config detected — workspace is already v2"

**Unit tests:**
- `DetectV1`: workgroups present, no workgroups, mixed paths, collision,
  bare repo, missing directories
- `ReportPlan`: report format for each case including collision and bare repo
  error messages
- `ApplyPlan`: successful full migration; abort on collision; abort on bare repo
- `git worktree repair` invocation: verify called for each moved worktree
- Config round-trip: `.gitw` after migration is valid v2 config (passes M1
  loader with no errors or warnings)
- Idempotency: running detect after `--apply` reports "no v1 config detected"



| Question | Decision |
|---|---|
| Version | 2.0 major; v1 workgroup config triggers actionable load-time error |
| Migration | `git w migrate` in M12 (parallel after M1); moves repos from arbitrary v1 paths to `repos/<n>`; converts workgroups to workstreams under `legacy` workspace; aborts on path collisions and bare repos; `git worktree repair` after directory moves |
| v1 repo path handling | Arbitrary `path` field in v1; `git w migrate --apply` moves directories to `repos/<n>` and updates config; bare repos abort migration with manual resolution instructions |
| Migration parallelism | M12 depends only on M1 types being available; can be developed concurrently with M2–11 |
| Infra Pattern A (branch-per-env) | `track_branch` + `upstream` fields on `[[repo]]`; repo aliases; `--env-group` expansion |
| Infra Pattern B (folder-per-env) | Multiple named `[[worktree]]` entries from same repo; `name`, `path`, `scope` fields |
| `[[worktree]] name` field | Unique key within workstream; defaults to repo name for single-occurrence; required for multi-occurrence |
| `[[worktree]] path` field | Unique on-disk path; defaults to `name` when not set |
| `[[worktree]] scope` field | Advisory subdirectory hint; no git enforcement; surfaced in AGENTS.md and JSON |
| Pattern B creation flag | `--worktree name=<n>,repo=<r>,branch=<b>[,path][,scope]`; error on duplicate repo without disambiguation |
| Pattern B AGENTS.md | Cross-modification warning block generated when same repo appears multiple times |
| Pattern B ship | One PR per `[[worktree]]` entry; scope noted in PR description |
| Env-group creation | `--env-group <upstream>[:<branch>]` on `workstream create` |
| Env-group expansion | Always stored as explicit `[[worktree]]` entries in `.gitw-stream` |
| `--repo` + `--env-group` + `--worktree` | All composable in same `workstream create` command |
| Branch defaults for env-group | Defaults to workstream name; overridable with `--env-group upstream:branch` |
| Upstream match in `--repo` filter | Matches all aliases; grouped output |
| Workgroup retirement | Replaced by workstream; v1 config triggers error with migrate instructions |
| Workspace representation | `[[workspace]]` blocks in `.gitw` + directories under `workspaces/` |
| Workstream config | Self-contained `.gitw-stream`; not in root `.gitw` |
| Active/archived split | `active/` and `archived/` under each workspace |
| Worktree ephemerality | Ephemeral; `.planning/`, `AGENTS.md`, `.gitw-stream` committed |
| `.planning/` on close | Always preserved; archived with workstream directory |
| Scope selection | Filter flags at call site; no ambient scope command |
| Agent interop | Generated `AGENTS.md` + `git w agent context --json`; no GSD API coupling |
| GSD workspace creation | Prohibited by `AGENTS.md`; git-w creates workstreams |
| Ship command | `git w workstream ship`; agents must not push or open PRs directly |
| Ref routing | Explicit `[[sync_pair]]` blocks |
| State file | `.git/git-w-state.json`; machine-local |
| Hook mechanism | `pre-push` hook + `reconcileHooks`; self-healing via sync side effect |
| Commands cut from v1 (top-level) | `info`, `fetch`, `pull`, `push`, `context` (scope-setter); all aliases cut with them |
| Command families cut from v1 | `group` (6 subcommands), `workgroup` (6 subcommands), `worktree` (5 subcommands) |
| Subcommands cut within kept families | `branch default`, `repo clone` (merged into `repo add`), `repo rename`, `repo unlink` |
| Net command reduction | 39 → 27 commands/subcommands (−12) |
| `commit --workgroup` flag | Renamed to `--workstream`; short form `-W` preserved |
| `branch default` cut | Use `git w exec checkout <default-branch>` |
| `repo clone` cut | Merged into `git w repo add <url>`; `--no-clone` for edge case |
| Delete commands | Not added for repos, workspaces, or remotes; edit config directly |

---

## Deferred (post-v2.0)

- Token storage via keychain or 1Password CLI (currently `token_env` only)
- `git w workstream ship --open-prs` for non-GitHub remotes
- Cross-workstream dependency tracking
- `git w context rebuild` heuristic repo descriptions from README parsing
- Forgejo API divergence from Gitea (currently treated as compatible)
- `[[sync_pair]]` ref filtering beyond globs (by age, exclude tags)
- Per-worktree devcontainer support
- Infra Pattern A promotion tracking: awareness of dev→test→prod chain
- Pattern B cross-PR linking: when `--open-prs` opens multiple PRs for the
  same underlying repo (e.g., TICKET-123-dev and TICKET-123-prod), automatically
  cross-reference them in PR descriptions
- Pattern B scope enforcement: optional pre-commit hook that warns (not blocks)
  when files outside `scope` are staged in a worktree where scope is set
- `git w workstream ship` coordinated PR sequencing for Pattern B (e.g., ensure
  dev PR merges before prod PR is opened)

---

## Full annotated config example

### `.gitw` (committed, shared)

```toml
[metarepo]
name               = "platform-work"
default_remotes    = ["origin"]
agentic_frameworks = ["gsd"]

[[workspace]]
name        = "payments-platform"
description = "Payment processing and related services"
repos       = ["api-service", "payment-lib", "gateway"]

[[workspace]]
name        = "platform-infra"
description = "Infrastructure for the platform — all environments"
repos       = ["infra-dev", "infra-test", "infra-prod", "k8s-config"]

[[workspace]]
name        = "support"
description = "Bug fixes and production support"
repos       = ["api-service", "auth-lib", "gateway", "service-c"]

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
name      = "payment-lib"
path      = "repos/payment-lib"
clone_url = "https://github.com/work-org/payment-lib"

[[repo]]
name      = "gateway"
path      = "repos/gateway"
clone_url = "https://github.com/work-org/gateway"

[[repo]]
name      = "k8s-config"
path      = "repos/k8s-config"
clone_url = "https://github.com/work-org/k8s-config"

# Infra repo materialized as three env aliases
[[repo]]
name         = "infra-dev"
path         = "repos/infra-dev"
clone_url    = "https://github.com/work-org/infra"
track_branch = "dev"
upstream     = "infra"

[[repo]]
name         = "infra-test"
path         = "repos/infra-test"
clone_url    = "https://github.com/work-org/infra"
track_branch = "test"
upstream     = "infra"

[[repo]]
name         = "infra-prod"
path         = "repos/infra-prod"
clone_url    = "https://github.com/work-org/infra"
track_branch = "prod"
upstream     = "infra"

[[repo]]
name      = "consolidated-infra"
path      = "repos/consolidated-infra"
clone_url = "https://github.com/work-org/consolidated-infra"
# single main branch; environments separated by folder: environments/dev/, test/, prod/
# no track_branch or upstream — standard single-branch repo

[[repo]]
name      = "generated-code"
path      = "repos/generated-code"
clone_url = "https://github.com/work-org/generated-code"
remotes   = ["origin"]      # opt out of personal mirror
```

### `.git/.gitw` (private, never committed)

```toml
[[remote]]
name      = "origin"
token_env = "GITHUB_TOKEN"

[[remote]]
name        = "personal"
kind        = "gitea"
url         = "https://gitea.robertwritescode.com"
user        = "robert"
token_env   = "GITEA_TOKEN"
repo_prefix = "work-"
direction   = "both"
push_mode   = "mirror"
fetch_mode  = "all"
critical    = false
private     = true

[[remote.branch_rule]]
pattern = "**"
action  = "allow"

[[sync_pair]]
from = "origin"
to   = "personal"
refs = ["**"]

[metarepo]
default_remotes = ["origin", "personal"]

# WIP protection: personal only during active work
[[workstream]]
name    = "TICKET-123"
remotes = ["personal"]

[[workstream]]
name    = "INFRA-42"
remotes = ["personal"]
```

### Example workstream creation commands

```sh
# Standard multi-service workstream
git w workstream create payments-platform TICKET-456 \
  --description "Add retry logic to payment processor and update consumer" \
  --repo api-service:feat/TICKET-456-retry \
  --repo payment-lib:feat/TICKET-456-consumer-compat

# Pattern A: all infra envs in one command (branch defaults to workstream name)
git w workstream create platform-infra INFRA-42 \
  --description "Add RDS config to all environments" \
  --env-group infra

# Pattern A: all infra envs with explicit branch name
git w workstream create platform-infra INFRA-43 \
  --description "Rotate secrets across environments" \
  --env-group infra:feat/INFRA-43-secret-rotation

# Pattern A: one env only (use explicit --repo)
git w workstream create platform-infra INFRA-55 \
  --description "Dev-only experiment" \
  --repo infra-dev:feat/INFRA-55-experiment

# Pattern A: mixed — all infra envs + k8s config
git w workstream create platform-infra INFRA-60 \
  --description "New deployment strategy: infra and k8s" \
  --env-group infra:feat/INFRA-60-deploy \
  --repo k8s-config:feat/INFRA-60-deploy

# Pattern B: consolidated-infra, touch dev and prod in same workstream
git w workstream create platform-infra TICKET-123 \
  --description "Update RDS config for dev and prod environments" \
  --worktree name=dev,repo=consolidated-infra,branch=feat/TICKET-123-dev,scope=environments/dev \
  --worktree name=prod,repo=consolidated-infra,branch=feat/TICKET-123-prod,scope=environments/prod

# Pattern B: consolidated-infra, touch all three envs
git w workstream create platform-infra TICKET-200 \
  --description "Rotate TLS certs across all environments" \
  --worktree name=dev,repo=consolidated-infra,branch=feat/TICKET-200-certs,scope=environments/dev \
  --worktree name=test,repo=consolidated-infra,branch=feat/TICKET-200-certs,scope=environments/test \
  --worktree name=prod,repo=consolidated-infra,branch=feat/TICKET-200-certs,scope=environments/prod

# Pattern B: consolidated-infra dev only (single worktree; --repo is fine)
git w workstream create platform-infra TICKET-210 \
  --description "Experiment with new caching layer in dev" \
  --repo consolidated-infra:feat/TICKET-210-cache-dev

# Mixed: Pattern B consolidated-infra + a standard service repo
git w workstream create platform-infra TICKET-300 \
  --description "API config change + infra update for dev and prod" \
  --repo api-service:feat/TICKET-300-config \
  --worktree name=dev,repo=consolidated-infra,branch=feat/TICKET-300-infra-dev,scope=environments/dev \
  --worktree name=prod,repo=consolidated-infra,branch=feat/TICKET-300-infra-prod,scope=environments/prod

# Simple single-repo bug fix (support workspace)
git w workstream create support BUG-789 \
  --description "Fix null check in auth middleware" \
  --repo api-service:fix/BUG-789-null-check

# Adding a Pattern B worktree to an existing workstream post-creation
git w workstream worktree add consolidated-infra \
  --branch feat/TICKET-123-test \
  --worktree-name test \
  --scope environments/test
```