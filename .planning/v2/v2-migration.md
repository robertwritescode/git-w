# git-w v2: v1 → v2 migration

## `git w migrate`

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

---

## Breaking changes summary

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
`AGENTS.md` files. The v1 scope-setting behavior is replaced by
`--workspace`, `--workstream`, and `--repo` filter flags at the call site
of any read command.

---

## `pkg/migrate` package

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
