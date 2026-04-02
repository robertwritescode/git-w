# git-w v2: infrastructure repo patterns

Two common conventions for organizing infrastructure repos are both supported
as first-class patterns in git-w.

---

## Pattern A: branch-per-environment

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

### Sync behavior for Pattern A aliases

`git w sync` on an alias repo fetches all refs from the remote, pulls
`track_branch` into the local checkout, and mirror-pushes all refs to a single
personal backup repo named after the upstream (e.g., `infra`). All aliases that
share the same upstream mirror-push to that one repo, preserving all branches
(`work-infra-dev`, `work-infra-test`, `work-infra-prod`) together in a single
1-to-1 mirror.

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

### `git w agent context --json` env-group data (Pattern A)

```json
{
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
them to enumerate aliases manually.

---

## Pattern B: folder-per-environment

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

Pattern B repos (`consolidated-infra`) are added with a standard `git w repo
add` — no special flags. The multi-worktree behavior is configured at workstream
creation time, not at repo registration time.

### `--worktree` flag in workstream create (Pattern B)

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

**Error when `--repo` is used twice for the same repo without disambiguation:**
```
Error: repo "consolidated-infra" appears more than once.
Use --worktree to specify name and scope for each:
  --worktree name=dev,repo=consolidated-infra,branch=feat/TICKET-123-dev,scope=environments/dev
  --worktree name=prod,repo=consolidated-infra,branch=feat/TICKET-123-prod,scope=environments/prod
```

`--repo` and `--worktree` are composable.

### `git w workstream worktree add` for Pattern B

To add a named worktree to an existing workstream post-creation:

```
git w workstream worktree add consolidated-infra \
  --branch feat/TICKET-123-test \
  --worktree-name test \
  --scope environments/test
```

`--worktree-name` is required when the repo already has a worktree in the
workstream.

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

---

## Workstream creation examples

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
