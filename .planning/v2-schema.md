# git-w v2: config schema

All schema blocks for `.gitw`, `.git/.gitw`, and `.gitw-stream` manifest files.

## Config file model

| File | Location | Committed? | Purpose |
|---|---|---|---|
| `.gitw` | workspace root | yes (normally) | shared workspace config |
| `.git/.gitw` | inside `.git/` | never | private local overrides |

**Merge semantics:** field-level merge; `.git/.gitw` wins on conflicts.
- `[[remote]]` merged by `name`
- `[[sync_pair]]` merged by `(from, to)` pair
- `[[repo]]` merged by `name`
- `[[workspace]]` merged by `name`
- `[[workstream]]` root blocks merged by `name`
- `.gitw-stream` files are self-contained and not merged at the root config level

**Privacy enforcement:** `private = true` remotes rejected at load time if found
in `.gitw`. Must live in `.git/.gitw`. Error message names the remote and
explains where it must live.

---

## `[workspace]` top-level block

```toml
[workspace]
name            = "platform-work"
default_remotes = ["origin", "personal"]
# Names of [[remote]] blocks that all repos use by default.
# An empty list means no workspace defaults — every repo declares its own.
# A repo with no remotes field and no workspace default_remotes gets no
# secondary remotes (only its own git-configured origin).
```

---

## `[[workspace]]` named workspace blocks

```toml
[[workspace]]
name        = "payments-platform"
description = "Payment processing and related services"
repos       = ["api-service", "payment-lib", "gateway"]
```

---

## `[[repo]]` block

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
remotes      = ["origin", "personal"]  # override workspace default_remotes
```

### `[[repo.branch_override]]`

Per-repo branch overrides — evaluated before the remote's own branch_rules.

```toml
[[repo.branch_override]]
remote  = "origin"
pattern = "experiment/*"
action  = "block"
reason  = "Experimental branches stay on personal only"

[[repo.branch_override]]
remote   = "personal"
pattern  = "internal-spike"
explicit = true             # marks this branch as explicitly opted in
action   = "allow"
```

---

## `[[remote]]` block

```toml
[[remote]]
# Required fields
name = "personal"           # referenced by repos, workstreams, sync_pairs
kind = "gitea"              # "gitea" | "forgejo" | "github" | "generic"
url  = "https://gitea.example.com"

# Identity
user      = "youruser"
token_env = "GITW_GITEA_TOKEN"  # env var name; never a literal token value
org       = ""                    # push to org namespace instead of user; optional

# Repo naming on this remote
repo_prefix = "org-"        # remote repo named "<prefix><local-repo-name>"
repo_suffix = ""            # appended after repo name; optional

# Push/fetch behaviour
direction  = "push"         # "push" | "fetch" | "both"
push_mode  = "mirror"       # "mirror" (--mirror) | "branch" (named branches only)
fetch_mode = "all"          # "all" | "tracked" — which refs to fetch

# SSH alternative
use_ssh  = false
ssh_host = ""               # hostname only, if different from url host

# Sync execution
critical = false            # if true, failure of this remote fails the whole repo sync

# Privacy
private = true              # if true, must live in .git/.gitw, never in .gitw
```

---

## `[[remote.branch_rule]]`

Branch rules are defined on the remote and apply wherever that remote is used,
unless overridden at the repo level. Rules are evaluated in declaration order;
first match wins. If no rule matches, the default action is `allow`.

Each rule matches a branch against up to three independent criteria. All
criteria present on a rule must match for the rule to fire. Omitted criteria
are wildcards.

**Criteria:**
- `pattern` — glob against branch name. `*` matches anything except `/`.
  `**` matches across `/`. Omit to match all branch names.
- `untracked` — if `true`, branch must have no configured upstream on this
  remote. Omit to ignore.
- `explicit` — if `true`, branch must appear in a `[[repo.branch_override]]`
  with `explicit = true` for this remote. Omit to ignore.

**Evaluation order (per branch, per remote):**
1. Repo-level `[[repo.branch_override]]` entries for this remote — declaration
   order, first match wins.
2. Remote-level `[[remote.branch_rule]]` entries — declaration order, first
   match wins.
3. No match — default action is `allow`.

```toml
[[remote.branch_rule]]
pattern   = "wip/*"
untracked = true
explicit  = false
action    = "block"               # "allow" | "block" | "warn" | "require-flag"
reason    = "WIP branches must not be pushed to org"
flag      = "--push-wip"          # for require-flag only
```

**Action semantics:**

| Action | Behaviour |
|---|---|
| `allow` | Branch is pushed normally. |
| `block` | Push to this remote is skipped. Logged as an error. Other remotes unaffected. |
| `warn` | Push proceeds but a warning is printed to output. |
| `require-flag` | Push is blocked unless the specified `flag` is passed to `git w sync`. |

---

## `[[sync_pair]]` block

```toml
[[sync_pair]]
from = "origin"     # name of a [[remote]] that is the ref source
to   = "personal"   # name of a [[remote]] that receives refs from `from`

# Optional: restrict which refs flow through this pair.
# Omit to pass all refs fetched from `from`.
refs = ["**"]       # default: all refs. ** crosses /; * does not.
# refs = ["main", "release/**"]   # only these refs flow origin → personal
```

Multiple `[[sync_pair]]` blocks are allowed. Cycle detection at load time.
Fan-out: fetch all `from` remotes in parallel, then push to `to` remotes
filtered by `refs` and subject to branch rule evaluation.

---

## `[[workstream]]` block in root `.gitw` (lightweight remote override only)

The root config does not contain full workstream definitions. It may contain
per-workstream remote overrides in the private config:

```toml
# In .git/.gitw only
[[workstream]]
name    = "INFRA-42"
remotes = ["personal"]    # replaces workspace defaults for this workstream's worktrees
```

`remotes` on a `[[workstream]]` completely replaces workspace `default_remotes`
for all worktrees in that workstream. Repo-level `remotes` overrides still take
precedence over this.

**Cascade resolution (innermost wins):**
`[workspace] default_remotes` → `[[workstream]] remotes` → `[[repo]] remotes`

---

## `.gitw-stream` manifest

Lives at `workspaces/<workspace>/active/<n>/.gitw-stream`. Committed to the
meta-repo. Machine-readable source of truth for a workstream.

The `[[worktree]]` array is the core of the manifest. Two constraints apply:
- `name` must be unique within the workstream (it is the primary key)
- `path` must be unique within the workstream (it is the on-disk key)

When a repo appears only once in a workstream, `name` defaults to the repo
name and may be omitted. When the same repo appears more than once (Pattern B
consolidated infra), `name` is required on all entries for that repo.

```toml
name        = "INFRA-42"
description = "Add RDS config to all environments"
workspace   = "platform-infra"
status      = "active"        # "active" | "shipped" | "archived"
created     = "2026-03-15"

# --- Pattern A: branch-per-env (repo aliases, one worktree per alias) ---
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
# [[worktree]]
# name   = "dev"                        # required; unique within workstream
# repo   = "consolidated-infra"
# branch = "feat/TICKET-123-dev"
# path   = "consolidated-infra-dev"     # unique on-disk path within workstream dir
# scope  = "environments/dev"           # advisory; no git enforcement

[ship]
pr_urls             = []
pre_ship_branches   = {}   # worktree-name -> "branch-name-pre-ship-<timestamp>" on personal
shipped_at          = ""

[context]
summary       = ""
key_decisions = []
```

**`name` field:** unique key within the workstream. For single-occurrence repos,
defaults to the repo name. For multi-occurrence repos (Pattern B), required and
must be set explicitly. Validated unique at creation time.

**`path` field:** relative path from the workstream directory to the worktree
on disk. Must be unique within the workstream. When `name` is explicitly set
and `path` is omitted, `path` defaults to `name`.

**`scope` field:** optional advisory metadata. The subdirectory within the
repo that this worktree is expected to modify. Not enforced by any git
operation. Surfaced in the workstream `AGENTS.md` worktree table and in
`git w agent context --json`. Agents must respect scope boundaries; git-w
does not enforce them mechanically.

---

## Full annotated config example

### `.gitw` (committed, shared)

```toml
[workspace]
name            = "platform-work"
default_remotes = ["origin"]

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

# Infra repo materialized as three env aliases (Pattern A)
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
# single main branch; environments separated by folder (Pattern B)

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

[workspace]
default_remotes = ["origin", "personal"]

# WIP protection: personal only during active work
[[workstream]]
name    = "TICKET-123"
remotes = ["personal"]

[[workstream]]
name    = "INFRA-42"
remotes = ["personal"]
```

---

## Implementation notes

**Config loading:**
- `MergeRemote(base, override Remote) Remote` in `pkg/config`: for each
  exported field, override value wins if non-zero, else base value used.
- Same pattern: `MergeRepo`, `MergeWorkstream`, `MergeSyncPair`.
- `private = true` enforcement: named error if found in `.gitw` (not
  `.git/.gitw`). Error message names the remote and the correct file.
- Cycle detection in `[[sync_pair]]` graph at config load time.
- Table-driven tests: every merge case including zero-value fields, partial
  overrides, and array field behaviour.
- `UpdatePreservingComments`: all new config fields round-trip without losing
  comments or field order.
- Load-time detection of v1 `[[workgroup]]` blocks: actionable error message
  directing user to run `git w migrate` (detection only; no migration logic in
  config loader).
