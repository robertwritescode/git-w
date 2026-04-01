# git-w v2: milestones

> **For the authoritative list of GitHub issue numbers, issue titles, exact branch names, and per-milestone issue assignments, see `.planning/v2-issue-map.md`.** The milestone descriptions below are design scope only — use the issue map for all branching and planning decisions.

**Dependency note:** Milestones are numbered by logical dependency order, not
required execution order. Milestone 12 (`git w migrate`) depends only on
Milestone 1 being code-complete (types and loader exist) and can be developed
in parallel with Milestones 2–11. All milestones ship together as v2.0.

---

## Milestone 1 — v2 config schema + loader

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
- `[[remote]]` block: parsing, validation, field-level merge
- `[[remote.branch_rule]]` parsing and validation
- `[[sync_pair]]` with cycle detection
- `[[workstream]]` root config block (lightweight remote override)
- Two-file merge with field-level semantics
- `private = true` enforcement
- `[workspace] default_remotes` cascade resolution
- Load-time detection of v1 `[[workgroup]]` blocks: actionable error message
  directing user to run `git w migrate` (detection only; no migration logic)
- `UpdatePreservingComments` round-trip tests for all new fields
- Full unit tests: merge, cascade, cycle detection, v1 `[[workgroup]]` detection

## Milestone 2 — branch rule engine

- `BranchInfo` type
- `EvaluateRule` pure function
- Internal glob package: `*` (no `/` crossing) and `**` (crosses `/`)
- `untracked` criterion: query go-git for tracking ref on named remote
- `explicit` criterion from `[[repo.branch_override]]` with `explicit = true`
- All four action tiers: `allow`, `block`, `warn`, `require-flag`
- Table-driven tests: all criteria combinations × all action tiers

## Milestone 3 — `git w sync` multi-remote + fan-out

- Cascade resolution: workspace → workstream → repo effective remote list
- `track_branch` as pull target for alias repos
- `[[sync_pair]]` fan-out: fetch phase then push phase, parallel with errgroup
- Local-only WIP branch push to permissive remotes
- Branch rule evaluation: repo overrides → remote rules → default allow
- `--force` on all `push_mode = "mirror"` pushes
- `critical` flag semantics on failure
- `--remote`, `--workspace`, `--workstream`, `--no-push`, `--push-wip`,
  `--dry-run` flags
- Per-remote grouped output with summary line
- `reconcileHooks` side effect on every sync
- State file writes to `.git/git-w-state.json` after each successful operation

## Milestone 4 — `git w remote` subcommand

- `git w remote list` with `--json`
- `git w remote add` interactive wizard and non-interactive flags
- Gitea/Forgejo API provider
- GitHub API provider
- Generic no-op provider
- `repo_prefix` / `repo_suffix` during provisioning
- Alias repo awareness: existence check uses upstream repo name for
  `track_branch` repos
- `gitw-<name>` upsert on all child repos (including alias repos)
- Optional initial mirror push after provisioning
- Optional `[[sync_pair]]` creation after provisioning
- `git w remote remove` — config + local git remotes only; no API deletion
- `git w remote status` — connectivity + last-sync from state file

## Milestone 5 — `git w status` + `git w branch checkout --from`

- Unified status merging v1 `info` and `status`
- Env-group display: aliases grouped under upstream name with `(env)` annotation
  in workstream section
- `--workspace`, `--workstream`, `--repo` filter flags
- `--repo` with upstream name: matches all aliases, grouped output
- Remote staleness section from state file
- Available-branch hints with suggested `git w branch checkout --from` commands
- `git w branch checkout <branch> --from <remote>`
- `--json` output

## Milestone 6 — workstream push protection

- `reconcileHooks(repo RepoConfig)` internal function
  - Determines protected state by scanning `.gitw-stream` manifests across all
    active workstreams
  - Writes or updates git-w managed block in `.git/hooks/pre-push`
  - Appends to existing hooks; never overwrites non-git-w content
  - Removes block and cleans up empty hook files when protection lifted
  - Idempotent: safe to call on every sync
- `git w workstream create` calls `reconcileHooks` on all repos with worktrees
- `git w sync` calls `reconcileHooks` on all repos as side effect
- `git-w hook pre-push` subcommand (internal, called by installed hook):
  - Walks up to workspace root
  - Resolves current worktree path
  - Matches against `[[worktree]] path` entries in `.gitw-stream` manifests
    across all active workstreams
  - Evaluates remote name against `[[workstream]] remotes` whitelist in
    `.git/.gitw`
  - Blocks with formatted error message if not whitelisted
  - No-op (exits 0) if worktree not in any workstream
- `git w workstream ship` lifts protection:
  - Adds `origin` to `[[workstream]] remotes` in `.git/.gitw`
  - Calls `reconcileHooks` on all repos with worktrees in this workstream
- Unit tests for `reconcileHooks`: install, append, update, remove, cleanup
- Unit tests for hook evaluation: worktree match, remote whitelist, no-match
  allow, binary-not-on-PATH block
- Integration test: direct `git push` from a protected worktree is blocked

## Milestone 7 — workspace and workstream lifecycle

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

## Milestone 8 — infra repo patterns (Pattern A aliases + Pattern B multi-worktree)

**Pattern A — branch-per-env (repo aliases):**
- `track_branch` field: `--branch` and `--branch-map` flags on `git w repo add`
- `upstream` field: auto-set by `--branch-map`; `--upstream-name` override
- `git w repo list --upstream <n>` filter
- `git w status --repo <upstream>` grouped display for alias sets
- `ResolveEnvGroup` in `pkg/workspace`
- `--env-group` in `git w workstream create`
- Sync: `track_branch` as pull target per alias
- Mirror push: aliases share a single personal remote repo named after the upstream
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
- `--open-prs` in ship opens one PR per worktree entry

**Shared:**
- Unit tests: `name`/`path` uniqueness enforcement; Pattern A and B manifest
  round-trips; `--worktree` flag parsing; `--env-group` expansion; `ResolveEnvGroup`

## Milestone 9 — agent context layer

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

## Milestone 10 — ship pipeline

- `git w workstream ship`: dirty check, optional interactive squash pass
  (`--squash`), pre-ship backup branch creation on personal remote,
  `pre_ship_branches` recording in `.gitw-stream`, push protection lift,
  `--push-all`, `--open-prs` (one PR per worktree branch), URL recording,
  `shipped_at`, status update, auto-commit, `--dry-run`

## Milestone 11 — close and archival

- `git w workstream close`: shipped check, worktree removal, hook cleanup,
  branch pruning prompts, `mv` to archived, status update, `AGENTS.md` update,
  `.planning/` preservation, auto-commit, context rebuild
- `--no-archive` with explicit confirmation

## Milestone 12 — `git w migrate` (parallelizable after M1)

**Dependency:** Milestone 1 must be code-complete (config types and loader
exist). Can be developed concurrently with Milestones 2–11.

See `.planning/v2-migration.md` for full spec of the `pkg/migrate` package,
detection logic, pre-flight conditions, `--apply` execution sequence, and
unit tests.

---

## Resolved design decisions

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
| Commands cut from v1 (top-level) | `info`, `fetch`, `pull`, `push`, `context` (scope-setter) |
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
- Pattern B cross-PR linking when `--open-prs` opens multiple PRs for the
  same underlying repo
- Pattern B scope enforcement: optional pre-commit hook that warns (not blocks)
  when files outside `scope` are staged
- `git w workstream ship` coordinated PR sequencing for Pattern B
