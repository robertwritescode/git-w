# Roadmap: git-w v2

## Overview

git-w v2 replaces the workgroup model with workspace/workstream hierarchy, adds multi-remote sync with push protection, supports infra repo patterns (branch-per-env and folder-per-env), and introduces an agent interop layer. The roadmap follows the dependency chain: config types first (everything imports them), then branch rule engine, sync fan-out, remotes, status, hooks, workstream lifecycle, infra patterns, agent context, ship, close, and migration (parallel after M1). 63 phases across 12 milestones.

## Milestones

- 📋 **M1: Config Schema + Loader** - Phases 1-11
- 📋 **M2: Branch Rule Engine** - Phases 12-14
- 📋 **M3: Sync Multi-Remote Fan-Out** - Phases 15-20
- 📋 **M4: Remote Subcommand** - Phases 21-25
- 📋 **M5: Status + Checkout --from** - Phases 26-30
- 📋 **M6: Push Protection** - Phases 31-34
- 📋 **M7: Workspace + Workstream Lifecycle** - Phases 35-41
- 📋 **M8: Infra Patterns (A + B)** - Phases 42-47
- 📋 **M9: Agent Context Layer** - Phases 48-50
- 📋 **M10: Ship Pipeline** - Phases 51-55
- 📋 **M11: Close and Archival** - Phases 56-58
- 📋 **M12: Migration** - Phases 59-63

## Phases

- [x] **Phase 1: Add `[[workspace]]` block** - Parse workspace blocks with name, description, repos list, and agentic_frameworks validation
- [ ] **Phase 2: Add `track_branch` and `upstream` fields** - Env alias fields on `[[repo]]` for branch-per-env pattern
- [ ] **Phase 3: Enforce `repos/<n>` path convention** - Load-time warning for v1 paths with migration suggestion
- [ ] **Phase 4: Add `[[remote]]` and `[[remote.branch_rule]]`** - Remote config and branch rule block parsing
- [ ] **Phase 5: Add `[[sync_pair]]` parsing** - Sync pair config with cycle detection at load time
- [ ] **Phase 6: Add `[[workstream]]` root block** - Lightweight remote override block in root config
- [ ] **Phase 7: Two-file config merge** - Field-level merge of `.gitw` and `.git/.gitw` with private-wins semantics
- [ ] **Phase 8: Parse `.gitw-stream` manifest** - Workstream manifest with `[[worktree]]` entries (name, path, scope)
- [ ] **Phase 9: Default remotes cascade** - `[metarepo] default_remotes` resolution: metarepo -> workstream -> repo
- [ ] **Phase 10: Detect v1 `[[workgroup]]` blocks** - Actionable error at load time directing to `git w migrate`
- [ ] **Phase 11: `UpdatePreservingComments` round-trip** - Round-trip tests for all v2 config fields
- [ ] **Phase 12: `BranchInfo` type and glob package** - Branch info type and internal glob matching with `*` and `**` patterns
- [ ] **Phase 13: `EvaluateRule` pure function** - Four action tiers (allow, block, warn, require-flag) with criteria evaluation
- [ ] **Phase 14: Rule criteria combination tests** - Table-driven tests for all criteria x action tier combinations
- [ ] **Phase 15: `sync_pair` fan-out executor** - Parallel fetch then push with errgroup bounded concurrency
- [ ] **Phase 16: Resolve effective remote list** - Per-repo remote resolution using cascade (metarepo -> workstream -> repo)
- [ ] **Phase 17: `track_branch` as pull target** - Alias repos pull from their tracked branch during sync
- [ ] **Phase 18: Sync flags** - `--remote`, `--workspace`, `--workstream`, `--no-push`, `--push-wip`, `--dry-run`
- [ ] **Phase 19: Sync output and state file** - Per-remote grouped output and `.git/git-w-state.json` writes
- [ ] **Phase 20: Branch rule eval in sync push** - Wire branch rule evaluation into sync push phase
- [ ] **Phase 21: `git w remote list`** - Display configured remotes with `--json` support
- [ ] **Phase 22: API providers** - Gitea/Forgejo and GitHub API providers for repo existence check and creation
- [ ] **Phase 23: `git w remote add`** - Interactive wizard and non-interactive flag-based remote addition
- [ ] **Phase 24: `git w remote status`** - Connectivity check and last-sync timestamps from state file
- [ ] **Phase 25: `git w remote remove`** - Remove config and local git remotes without API deletion
- [ ] **Phase 26: Unified status command** - Merge `info` and `status` into single display with repos, workstreams, remotes
- [ ] **Phase 27: Status filter flags** - `--workspace`, `--workstream`, `--repo` filter flags
- [ ] **Phase 28: Env-group display in status** - Aliases grouped under upstream name with available-branch hints
- [ ] **Phase 29: Status `--json` output** - JSON output format for unified status
- [ ] **Phase 30: `branch checkout --from`** - Fetch and create local branch from named remote
- [ ] **Phase 31: `reconcileHooks` in sync** - Wire `reconcileHooks` as side effect of `git w sync`
- [ ] **Phase 32: `reconcileHooks` function** - Install/update/remove git-w managed block in pre-push hook
- [ ] **Phase 33: `hook pre-push` subcommand** - Evaluate worktree path against workstream manifests, block unauthorized remotes
- [ ] **Phase 34: Push protection integration test** - Direct `git push` from protected worktree is blocked
- [ ] **Phase 35: `pkg/workspace` and `pkg/worktrees`** - Package scaffolding for workspace and worktree management
- [ ] **Phase 36: `workspace create` and `list`** - Directory scaffolding, `[[workspace]]` write, AGENTS.md stub, `--json` list
- [ ] **Phase 37: `workstream create` with `--repo`** - `.gitw-stream` write, AGENTS.md generation, `.planning/` creation, worktree setup
- [ ] **Phase 38: `--worktree` flag (Pattern B)** - Same repo multiple named worktrees with key=value pair syntax
- [ ] **Phase 39: `workstream list`, `status`, `switch`** - Workstream navigation commands with `--json` support
- [ ] **Phase 40: `workstream worktree add`** - Post-creation worktree addition with `--worktree-name` and `--scope` flags
- [ ] **Phase 41: `git w restore` worktrees** - Re-materialize missing worktrees for active workstreams via `git worktree repair`
- [ ] **Phase 42: `--branch` and `--branch-map` on repo add** - Create env aliases with branch tracking during repo add
- [ ] **Phase 43: `ResolveEnvGroup` and `--env-group`** - Expand upstream name to all alias repos; workstream create integration
- [ ] **Phase 44: `--upstream` filter** - `git w repo list --upstream <n>` and `git w status --repo <upstream>` grouped display
- [ ] **Phase 45: Pattern B name/path validation** - Unique name/path enforcement with actionable error on duplicate repo
- [ ] **Phase 46: Pattern B scope display** - Scope in status output and cross-modification warning in AGENTS.md
- [ ] **Phase 47: Mirror push naming for aliases** - Use upstream repo name (not alias name) on personal remote
- [ ] **Phase 48: `pkg/agents` package** - `SpecFramework` interface, `GSDFramework` impl, registry functions
- [ ] **Phase 49: `git w context rebuild`** - Regenerate `CONTEXT.md` and three-level `AGENTS.md` with framework content
- [ ] **Phase 50: `git w agent context --json`** - CWD-based scope resolution with workstream, env_groups, capabilities blocks
- [ ] **Phase 51: Ship dirty check** - Validate worktrees and warn on uncommitted changes
- [ ] **Phase 52: Ship `--push-all`** - Lift push protection and push all worktree branches to origin
- [ ] **Phase 53: Ship `--open-prs`** - Open one PR per worktree branch on GitHub, record URLs
- [ ] **Phase 54: Ship `--dry-run`** - Show what would happen without executing
- [ ] **Phase 55: Ship `--squash` and backup** - Backup branches on personal remote, squash per worktree
- [ ] **Phase 56: Close worktree removal** - Remove worktrees and clean up pre-push hooks
- [ ] **Phase 57: Close archive move** - Move workstream to `archived/` and update manifest status
- [ ] **Phase 58: Close `--no-archive`** - Delete directory with explicit confirmation instead of archiving
- [ ] **Phase 59: `MigrationPlan` and `DetectV1`** - V1 config detection and migration plan type
- [ ] **Phase 60: `ReportPlan` formatting** - Formatted migration report output
- [ ] **Phase 61: `ApplyPlan` with pre-flight abort** - Execute migration with collision and bare repo detection
- [ ] **Phase 62: `git w migrate` command** - CLI command with `--apply` flag (report-only by default)
- [ ] **Phase 63: Migration end-to-end tests** - Config round-trip, path moves, workgroup-to-workstream conversion

## Phase Details

### 📋 M1: Config Schema + Loader

Milestone branch: `v2-m1-config-schema` | Depends on: none

---

### Phase 1: Add `[[workspace]]` block
**Issue**: #36 | **Branch**: `36-add-workspace-block`
**Goal**: Users can define workspace blocks in `.gitw` and the tool validates agentic_frameworks against the framework registry
**Depends on**: Nothing (first phase)
**Requirements**: CFG-01, CFG-11
**Success Criteria** (what must be TRUE):
  1. `[[workspace]]` blocks with name, description, and repos list parse correctly from `.gitw`
  2. `agentic_frameworks` field validates against known framework registry; unknown values produce named error listing valid identifiers
  3. Missing `agentic_frameworks` defaults to `["gsd"]`
  4. Multi-value `agentic_frameworks` slices parse and validate correctly
**Canonical refs**: `.planning/v2/v2-schema.md`, `.planning/v2/v2-milestones.md` (M1), `.planning/v2/v2-agent-interop.md` (agentic_frameworks)
**Plans**: 3 plans
- [x] 01-01-PLAN.md — Rename WorkspaceMeta → MetarepoConfig and add WorkspaceBlock struct in pkg/config
- [x] 01-02-PLAN.md — Bootstrap pkg/agents with SpecFramework interface and GSDFramework implementation
- [x] 01-03-PLAN.md — Wire agentic_frameworks validation + defaulting into loader with full test coverage

---

### Phase 2: Add `track_branch` and `upstream` fields
**Issue**: #37 | **Branch**: `37-track-branch-upstream`
**Goal**: Users can annotate repos with env alias fields for branch-per-env infrastructure patterns
**Depends on**: Phase 1
**Requirements**: CFG-02
**Success Criteria** (what must be TRUE):
  1. `[[repo]]` blocks accept `track_branch` and `upstream` string fields
  2. `track_branch` repos are recognized as env aliases during config load
  3. `upstream` field links alias repos to their upstream repo name
**Canonical refs**: `.planning/v2/v2-schema.md` (repo block), `.planning/v2/v2-infra-patterns.md` (Pattern A)
**Plans**: 2 plans
- [ ] 02-01-PLAN.md — `[[repo]]` array-of-tables migration: Name + CloneURL fields, disk/in-memory split, test fixture cascade
- [ ] 02-02-PLAN.md — `track_branch` + `upstream` fields on RepoConfig, IsAlias() method, validateAliasFields (D-01/D-02)

---

### Phase 3: Enforce `repos/<n>` path convention
**Issue**: #38 | **Branch**: `38-repos-path-convention`
**Goal**: Tool warns at load time when repos use v1-style paths and suggests `git w migrate`
**Depends on**: Phase 2
**Requirements**: CFG-03
**Success Criteria** (what must be TRUE):
  1. Repos with paths not matching `repos/<n>` produce a load-time warning
  2. Warning message includes actionable suggestion to run `git w migrate`
  3. Non-conforming paths do not prevent config loading (warning, not error)
**Canonical refs**: `.planning/v2/v2-schema.md` (repo path), `.planning/v2/v2-migration.md`
**Plans**: 1 plan

Plans:
- [ ] 03-01-PLAN.md — Add Warnings field, path-convention check, and LoadConfig stderr output

---

### Phase 4: Add `[[remote]]` and `[[remote.branch_rule]]`
**Issue**: #39 | **Branch**: `39-remote-branch-rule-parsing`
**Goal**: Users can define remote configurations with branch-level push rules in `.gitw`
**Depends on**: Phase 3
**Requirements**: CFG-04
**Success Criteria** (what must be TRUE):
  1. `[[remote]]` blocks parse all specified fields (name, url, type, token_env, repo_prefix, repo_suffix, push_mode, critical)
  2. `[[remote.branch_rule]]` sub-blocks parse pattern, action, criteria fields
  3. Branch rules preserve declaration order (array-of-tables, not map)
  4. Invalid remote or rule configurations produce actionable validation errors
**Canonical refs**: `.planning/v2/v2-schema.md` (remote block), `.planning/v2/v2-remote-management.md`
**Plans**: 2 plans

Plans:
- [x] 04-01-PLAN.md — Type definitions: RemoteConfig, BranchRuleConfig, BranchAction, MergeRemote, diskConfig wiring
- [x] 04-02-PLAN.md — Validation: validateRemotes wired into buildAndValidate (name, uniqueness, kind, action, private)

---

### Phase 5: Add `[[sync_pair]]` parsing
**Issue**: #40 | **Branch**: `40-sync-pair-cycle-detection`
**Goal**: Users can define explicit sync routing between remotes with cycle detection preventing infinite loops
**Depends on**: Phase 4
**Requirements**: CFG-05
**Success Criteria** (what must be TRUE):
  1. `[[sync_pair]]` blocks parse source, destination, and ref_patterns fields
  2. Cycle detection at load time identifies circular sync routes
  3. Cycles produce actionable error message naming the cycle path
**Canonical refs**: `.planning/v2/v2-schema.md` (sync_pair block), `.planning/v2/v2-remote-management.md` (sync fan-out)
**Plans**: 2 plans
- [ ] 05-01-PLAN.md — SyncPairConfig struct, SyncPairs field on WorkspaceConfig, MergeSyncPair function + tests
- [ ] 05-02-PLAN.md — diskConfig wiring, validateSyncPairFields, detectSyncCycles, buildAndValidate integration + tests

---

### Phase 6: Add `[[workstream]]` root config block
**Issue**: #41 | **Branch**: `41-workstream-root-block`
**Goal**: Users can define lightweight workstream overrides in root config for remote targeting
**Depends on**: Phase 5
**Requirements**: CFG-06
**Success Criteria** (what must be TRUE):
  1. `[[workstream]]` blocks in `.gitw` parse name and remotes fields
  2. Workstream remote overrides participate in cascade resolution
  3. Multiple workstream blocks can coexist in the same config file
**Canonical refs**: `.planning/v2/v2-schema.md` (workstream block), `.planning/v2/v2-remote-management.md` (cascade)
**Plans**: 2 plans

Plans:
- [x] 06-01-PLAN.md — Define WorkstreamConfig contract, merge helper, and config-level tests
- [x] 06-02-PLAN.md — Wire loader parse/validation/normalization for [[workstream]] with strict-key and reference checks

---

### Phase 7: Two-file config merge
**Issue**: #42 | **Branch**: `42-two-file-config-merge`
**Goal**: Tool merges `.gitw` (shared) and `.git/.gitw` (private) with field-level semantics where private file wins on conflicts
**Depends on**: Phase 6
**Requirements**: CFG-07
**Success Criteria** (what must be TRUE):
  1. `.gitw` and `.git/.gitw` merge at field level (not file-level override)
  2. Private file fields win on conflict with shared file
  3. `private = true` fields in shared config are rejected with error
  4. Merge handles all v2 block types (workspace, repo, remote, sync_pair, workstream)
**Canonical refs**: `.planning/v2/v2-schema.md` (two-file merge, private enforcement)
**Plans**: 2 plans
Plans:
- [x] 07-01-PLAN.md — Add MergeRepo, MergeWorkspace, and mergeMetarepo helpers to config.go
- [x] 07-02-PLAN.md — Implement mergePrivateConfig and wire into Load()

---

### Phase 8: Parse `.gitw-stream` manifest
**Issue**: #43 | **Branch**: `43-gitw-stream-manifest`
**Goal**: Tool parses workstream manifests with worktree entries supporting name, path, and scope fields
**Depends on**: Phase 7
**Requirements**: CFG-08
**Success Criteria** (what must be TRUE):
  1. `.gitw-stream` files parse with all specified fields (name, status, worktrees)
  2. `[[worktree]]` entries support `name`, `path`, `scope` fields
  3. `name` defaults to repo name for single-occurrence repos; required for multi-occurrence
  4. `path` defaults to `name` when not explicitly set
  5. Load-time validation enforces `name` and `path` uniqueness within workstream
**Canonical refs**: `.planning/v2/v2-schema.md` (gitw-stream), `.planning/v2/v2-infra-patterns.md` (Pattern B worktree fields)
**Plans**: 2 plans

Plans:
- [x] 08-01-PLAN.md — Define WorkstreamManifest, WorktreeEntry, ShipState, StreamContext, WorkstreamStatus types in pkg/config/config.go
- [x] 08-02-PLAN.md — Implement LoadStream, applyStreamDefaults, validateStream in stream.go with full TDD test coverage

---

### Phase 9: Default remotes cascade
**Issue**: #44 | **Branch**: `44-default-remotes-cascade`
**Goal**: Tool resolves effective remotes per repo through three-level cascade where innermost wins
**Depends on**: Phase 8
**Requirements**: CFG-09
**Success Criteria** (what must be TRUE):
  1. Cascade resolves metarepo -> workstream -> repo (innermost wins)
  2. Repo-level remote overrides fully replace (not merge with) workstream-level
  3. Missing override at any level falls through to next outer level
**Canonical refs**: `.planning/v2/v2-schema.md` (cascade resolution), `.planning/v2/v2-remote-management.md` (effective remote list)
**Plans**: 2 plans
Plans:
- [ ] 09-01-PLAN.md — Fix MergeWorkstream/MergeRepo nil guards (prerequisite for cascade)
- [ ] 09-02-PLAN.md — Add ResolveRepoRemotes and ResolveWorkstreamRemotes cascade methods

---

### Phase 10: Detect v1 `[[workgroup]]` blocks
**Issue**: #45 | **Branch**: `45-detect-v1-workgroup`
**Goal**: v1 config files with workgroup blocks are caught at load time with clear migration instructions
**Depends on**: Phase 9
**Requirements**: CFG-10
**Success Criteria** (what must be TRUE):
  1. `[[workgroup]]` blocks in `.gitw` trigger an actionable error at load time
  2. Error message directs user to run `git w migrate`
  3. Detection only; no migration logic executed
**Canonical refs**: `.planning/v2/v2-schema.md`, `.planning/v2/v2-migration.md` (v1 detection)
**Plans**: 1 plan
Plans:
- [ ] 10-01-PLAN.md — Detect v1 `[[workgroup]]` blocks at load time, return hard error with migration instruction

---

### Phase 11: `UpdatePreservingComments` round-trip
**Issue**: #46 | **Branch**: `46-round-trip-tests`
**Goal**: Config round-trips preserve comments and field order for all v2 schema additions
**Depends on**: Phase 10
**Requirements**: CFG-12
**Success Criteria** (what must be TRUE):
  1. `UpdatePreservingComments` round-trips all v2 fields without losing comments
  2. Field ordering is preserved after write-read cycle
  3. All new block types (workspace, remote, sync_pair, workstream) are covered
**Canonical refs**: `.planning/v2/v2-schema.md`, `.planning/codebase/CONCERNS.md` (TOML comment preservation)
**Plans**: TBD

---

### 📋 M2: Branch Rule Engine

Milestone branch: `v2-m2-branch-rules` | Depends on: M1

---

### Phase 12: `BranchInfo` type and glob package
**Issue**: #47 | **Branch**: `47-branchinfo-glob-package`
**Goal**: Branch rule engine has its foundational type and glob matching supporting `*` and `**` patterns
**Depends on**: Phase 11
**Requirements**: BRULE-01
**Success Criteria** (what must be TRUE):
  1. `BranchInfo` type holds branch name, remote name, and tracking metadata
  2. Internal glob package supports `*` (no `/` crossing) and `**` (crosses `/`) patterns
  3. Glob matching is pure (no I/O) and table-driven testable
**Canonical refs**: `.planning/v2/v2-milestones.md` (M2), `.planning/v2/v2-remote-management.md` (branch rules)
**Plans**: TBD

---

### Phase 13: `EvaluateRule` pure function
**Issue**: #48 | **Branch**: `48-evaluate-rule`
**Goal**: Branch rules can be evaluated against branch info producing one of four action tiers
**Depends on**: Phase 12
**Requirements**: BRULE-02
**Success Criteria** (what must be TRUE):
  1. `EvaluateRule` accepts a rule and `BranchInfo`, returns an action tier
  2. All four action tiers work: allow, block, warn, require-flag
  3. Criteria evaluation handles `untracked` and `explicit` conditions
  4. Function is pure (no I/O, no side effects)
**Canonical refs**: `.planning/v2/v2-remote-management.md` (branch rule evaluation), `.planning/v2/v2-milestones.md` (M2)
**Plans**: TBD

---

### Phase 14: Rule criteria combination tests
**Issue**: #49 | **Branch**: `49-rule-criteria-tests`
**Goal**: All criteria combinations x action tiers are verified correct through comprehensive tests
**Depends on**: Phase 13
**Requirements**: BRULE-03
**Success Criteria** (what must be TRUE):
  1. Table-driven tests cover every criteria combination (pattern, untracked, explicit)
  2. Every action tier (allow, block, warn, require-flag) is tested with each criteria combo
  3. Edge cases (empty patterns, no criteria, conflicting criteria) are covered
**Canonical refs**: `.planning/v2/v2-milestones.md` (M2), `.planning/v2/v2-remote-management.md`
**Plans**: TBD

---

### 📋 M3: Sync Multi-Remote Fan-Out

Milestone branch: `v2-m3-sync-fanout` | Depends on: M2

---

### Phase 15: `sync_pair` fan-out executor
**Issue**: #50 | **Branch**: `50-sync-pair-fanout`
**Goal**: Sync pairs execute with parallel fetch then push using errgroup bounded concurrency
**Depends on**: Phase 14
**Requirements**: SYNC-01
**Success Criteria** (what must be TRUE):
  1. `[[sync_pair]]` routes execute fetch phase then push phase
  2. Fan-out uses errgroup with `SetLimit` for bounded parallelism
  3. `critical` flag semantics: failure of critical sync pair aborts remaining operations
  4. `push_mode = "mirror"` uses `--force` on push
**Canonical refs**: `.planning/v2/v2-remote-management.md` (sync fan-out executor), `.planning/v2/v2-milestones.md` (M3)
**Plans**: TBD

---

### Phase 16: Resolve effective remote list
**Issue**: #51 | **Branch**: `51-resolve-remote-list`
**Goal**: Each repo resolves its effective remote list through cascade for sync operations
**Depends on**: Phase 15
**Requirements**: SYNC-02
**Success Criteria** (what must be TRUE):
  1. Effective remote list per repo resolves via cascade (metarepo -> workstream -> repo)
  2. Cascade result determines which remotes a repo syncs to
  3. Resolution is consistent with the default_remotes cascade from Phase 9
**Canonical refs**: `.planning/v2/v2-remote-management.md` (cascade resolution), `.planning/v2/v2-schema.md`
**Plans**: TBD

---

### Phase 17: `track_branch` as pull target
**Issue**: #52 | **Branch**: `52-track-branch-pull`
**Goal**: Alias repos pull from their tracked branch instead of the default branch during sync
**Depends on**: Phase 16
**Requirements**: SYNC-03
**Success Criteria** (what must be TRUE):
  1. Repos with `track_branch` use it as the pull target during sync
  2. Non-alias repos continue pulling from their default branch
  3. Alias pull target works correctly with the fan-out executor
**Canonical refs**: `.planning/v2/v2-remote-management.md` (sync), `.planning/v2/v2-infra-patterns.md` (Pattern A sync)
**Plans**: TBD

---

### Phase 18: Sync flags
**Issue**: #54 | **Branch**: `54-sync-flags`
**Goal**: Users can filter and control sync behavior with scope and mode flags
**Depends on**: Phase 17
**Requirements**: SYNC-04
**Success Criteria** (what must be TRUE):
  1. `--remote` filters sync to specific remote(s)
  2. `--workspace` and `--workstream` scope sync to matching repos
  3. `--no-push` skips push phase (fetch only)
  4. `--push-wip` allows WIP branches to push to permissive remotes
  5. `--dry-run` shows what would happen without executing
**Canonical refs**: `.planning/v2/v2-commands.md` (sync command), `.planning/v2/v2-remote-management.md`
**Plans**: TBD

---

### Phase 19: Sync output and state file
**Issue**: #55 | **Branch**: `55-sync-output-state`
**Goal**: Sync produces per-remote grouped output and records timestamps in state file
**Depends on**: Phase 18
**Requirements**: SYNC-05
**Success Criteria** (what must be TRUE):
  1. Output groups results by remote with summary lines
  2. `.git/git-w-state.json` records last-sync timestamps per remote per repo
  3. State file is machine-local (not committed)
  4. State file writes are atomic (no partial writes on failure)
**Canonical refs**: `.planning/v2/v2-remote-management.md` (state file), `.planning/v2/v2-commands.md` (sync output)
**Plans**: TBD

---

### Phase 20: Branch rule eval in sync push
**Issue**: #98 | **Branch**: `98-sync-branch-rule-eval`
**Goal**: Sync push phase evaluates branch rules before pushing, respecting override precedence
**Depends on**: Phase 19
**Requirements**: SYNC-06
**Success Criteria** (what must be TRUE):
  1. Branch rules are evaluated during push phase (before each push)
  2. Evaluation order: repo overrides -> remote rules -> default allow
  3. `block` action prevents push with error message
  4. `warn` action prints warning but proceeds
  5. `require-flag` action blocks unless override flag is present
**Canonical refs**: `.planning/v2/v2-remote-management.md` (branch rule evaluation in sync), `.planning/v2/v2-milestones.md` (M3)
**Plans**: TBD

---

### 📋 M4: Remote Subcommand

Milestone branch: `v2-m4-remote-subcommand` | Depends on: M3

---

### Phase 21: `git w remote list`
**Issue**: #56 | **Branch**: `56-remote-list`
**Goal**: Users can view all configured remotes with human-readable and JSON output
**Depends on**: Phase 20
**Requirements**: RMT-01
**Success Criteria** (what must be TRUE):
  1. `git w remote list` displays all `[[remote]]` blocks from config
  2. Output shows name, URL, type, and push_mode per remote
  3. `--json` flag produces structured JSON output
**Canonical refs**: `.planning/v2/v2-commands.md` (remote list), `.planning/v2/v2-remote-management.md`
**Plans**: TBD

---

### Phase 22: API providers
**Issue**: #59 | **Branch**: `59-api-providers`
**Goal**: Gitea/Forgejo and GitHub providers can check repo existence and create repos via API
**Depends on**: Phase 21
**Requirements**: RMT-02
**Success Criteria** (what must be TRUE):
  1. `Provider` interface abstracts repo existence check and creation
  2. Gitea/Forgejo provider uses Gitea SDK for API operations
  3. GitHub provider uses go-github for API operations
  4. Generic no-op provider exists for remotes without API support
  5. `ProviderError` type wraps API errors without exposing raw HTTP status codes
**Canonical refs**: `.planning/v2/v2-remote-management.md` (providers), `.planning/v2/v2-milestones.md` (M4)
**Plans**: TBD

---

### Phase 23: `git w remote add`
**Issue**: #60 | **Branch**: `60-remote-add`
**Goal**: Users can add remotes through an interactive wizard or non-interactive flags
**Depends on**: Phase 22
**Requirements**: RMT-03
**Success Criteria** (what must be TRUE):
  1. Interactive wizard collects name, URL, type, token_env, repo_prefix/suffix
  2. Non-interactive mode accepts all fields via flags
  3. Provisioning creates repos on remote using appropriate provider
  4. `gitw-<name>` remote is upserted on all child repos (including aliases)
  5. Optional initial mirror push and `[[sync_pair]]` creation after provisioning
**Canonical refs**: `.planning/v2/v2-commands.md` (remote add), `.planning/v2/v2-remote-management.md` (provisioning)
**Plans**: TBD

---

### Phase 24: `git w remote status`
**Issue**: #61 | **Branch**: `61-remote-status`
**Goal**: Users can check remote connectivity and see last-sync timestamps
**Depends on**: Phase 23
**Requirements**: RMT-04
**Success Criteria** (what must be TRUE):
  1. Connectivity check verifies remote is reachable
  2. Last-sync timestamps read from `.git/git-w-state.json`
  3. Output distinguishes between "never synced" and "synced N ago"
**Canonical refs**: `.planning/v2/v2-commands.md` (remote status), `.planning/v2/v2-remote-management.md` (state file)
**Plans**: TBD

---

### Phase 25: `git w remote remove`
**Issue**: #99 | **Branch**: `99-remote-remove`
**Goal**: Users can remove a remote from config and local git remotes without deleting the remote repository
**Depends on**: Phase 24
**Requirements**: RMT-05
**Success Criteria** (what must be TRUE):
  1. Removes `[[remote]]` block from config
  2. Removes `gitw-<name>` git remote from all repos
  3. Removes associated `[[sync_pair]]` entries
  4. No API deletion of remote repository (config + local only)
**Canonical refs**: `.planning/v2/v2-commands.md` (remote remove), `.planning/v2/v2-remote-management.md`
**Plans**: TBD

---

### 📋 M5: Status + Checkout --from

Milestone branch: `v2-m5-status-checkout` | Depends on: M4

---

### Phase 26: Unified status command
**Issue**: #62 | **Branch**: `62-unified-status`
**Goal**: Users get a single `git w status` replacing the v1 split of `info` and `status` commands
**Depends on**: Phase 25
**Requirements**: STAT-01
**Success Criteria** (what must be TRUE):
  1. `git w status` displays repos section with branch, dirty state, ahead/behind
  2. Workstream section shows active workstreams with worktree counts
  3. Remote section shows configured remotes with sync staleness
  4. v1 `info` and `status` commands are removed or aliased
**Canonical refs**: `.planning/v2/v2-commands.md` (status output format), `.planning/v2/v2-milestones.md` (M5)
**Plans**: TBD

---

### Phase 27: Status filter flags
**Issue**: #63 | **Branch**: `63-status-filter-flags`
**Goal**: Users can scope status output to specific workspaces, workstreams, or repos
**Depends on**: Phase 26
**Requirements**: STAT-02
**Success Criteria** (what must be TRUE):
  1. `--workspace` flag filters to repos in a specific workspace
  2. `--workstream` flag filters to repos in a specific workstream
  3. `--repo` flag filters to a specific repo (or upstream name matching all aliases)
  4. Flags are composable (multiple can be used together)
**Canonical refs**: `.planning/v2/v2-commands.md` (status flags)
**Plans**: TBD

---

### Phase 28: Env-group display in status
**Issue**: #64 | **Branch**: `64-status-env-group`
**Goal**: Status groups alias repos under their upstream name with branch hints for checkout
**Depends on**: Phase 27
**Requirements**: STAT-03
**Success Criteria** (what must be TRUE):
  1. Alias repos display grouped under upstream name with `(env)` annotation
  2. Available-branch hints show branches from other remotes
  3. Hints include suggested `git w branch checkout --from` commands
**Canonical refs**: `.planning/v2/v2-commands.md` (status env-group display), `.planning/v2/v2-infra-patterns.md` (Pattern A display)
**Plans**: TBD

---

### Phase 29: Status `--json` output
**Issue**: #65 | **Branch**: `65-status-json`
**Goal**: Status output is available in structured JSON for programmatic consumption
**Depends on**: Phase 28
**Requirements**: STAT-04
**Success Criteria** (what must be TRUE):
  1. `--json` flag produces structured JSON with repos, workstreams, remotes arrays
  2. JSON schema includes all fields visible in human-readable output
  3. JSON output is stable (fields ordered consistently)
**Canonical refs**: `.planning/v2/v2-commands.md` (status --json)
**Plans**: TBD

---

### Phase 30: `branch checkout --from`
**Issue**: #66 | **Branch**: `66-checkout-from`
**Goal**: Users can fetch and create a local branch from a specific remote in one command
**Depends on**: Phase 29
**Requirements**: STAT-05
**Success Criteria** (what must be TRUE):
  1. `git w branch checkout <branch> --from <remote>` fetches from named remote
  2. Creates local tracking branch from the remote branch
  3. Works with `gitw-<name>` remote naming convention
**Canonical refs**: `.planning/v2/v2-commands.md` (branch checkout --from), `.planning/v2/v2-milestones.md` (M5)
**Plans**: TBD

---

### 📋 M6: Push Protection

Milestone branch: `v2-m6-push-protection` | Depends on: M5

---

### Phase 31: `reconcileHooks` in sync
**Issue**: #53 | **Branch**: `53-reconcile-hooks-sync`
**Goal**: Every sync operation ensures push protection hooks are up-to-date as a side effect
**Depends on**: Phase 30
**Requirements**: HOOK-02
**Success Criteria** (what must be TRUE):
  1. `git w sync` calls `reconcileHooks` on all repos after sync completes
  2. Hook reconciliation is silent on success (no output unless changes made)
  3. Hook errors do not fail the sync (warning only)
**Canonical refs**: `.planning/v2/v2-remote-management.md` (reconcileHooks), `.planning/v2/v2-milestones.md` (M6)
**Plans**: TBD

---

### Phase 32: `reconcileHooks` function
**Issue**: #67 | **Branch**: `67-reconcile-hooks`
**Goal**: Push protection hooks are installed, updated, and removed idempotently on repos
**Depends on**: Phase 31
**Requirements**: HOOK-01
**Success Criteria** (what must be TRUE):
  1. Installs git-w managed block in `.git/hooks/pre-push` when protection needed
  2. Updates existing block when config changes
  3. Removes block and cleans up empty hook files when protection lifted
  4. Appends to existing hooks; never overwrites non-git-w content
  5. Idempotent: calling multiple times produces same result
**Canonical refs**: `.planning/v2/v2-remote-management.md` (reconcileHooks spec), `.planning/v2/v2-milestones.md` (M6)
**Plans**: TBD

---

### Phase 33: `hook pre-push` subcommand
**Issue**: #68 | **Branch**: `68-hook-pre-push`
**Goal**: The pre-push hook can evaluate worktree context and block pushes to unauthorized remotes
**Depends on**: Phase 32
**Requirements**: HOOK-03
**Success Criteria** (what must be TRUE):
  1. `git-w hook pre-push` resolves current worktree path via `git rev-parse`
  2. Matches worktree against `[[worktree]]` entries in active `.gitw-stream` manifests
  3. Evaluates remote against workstream whitelist in `.git/.gitw`
  4. Blocks with formatted error if remote not whitelisted
  5. No-op (exit 0) if worktree not in any workstream
**Canonical refs**: `.planning/v2/v2-remote-management.md` (hook pre-push), `.planning/v2/v2-milestones.md` (M6)
**Plans**: TBD

---

### Phase 34: Push protection integration test
**Issue**: #69 | **Branch**: `69-push-protection-test`
**Goal**: End-to-end verification that direct git push from protected worktree is blocked
**Depends on**: Phase 33
**Requirements**: HOOK-04
**Success Criteria** (what must be TRUE):
  1. Integration test creates a workstream with worktrees and installs hooks
  2. Direct `git push` to non-whitelisted remote exits non-zero
  3. Error message includes workstream name and allowed remotes
  4. Push to whitelisted remote succeeds
**Canonical refs**: `.planning/v2/v2-remote-management.md` (integration test), `.planning/v2/v2-milestones.md` (M6)
**Plans**: TBD

---

### 📋 M7: Workspace + Workstream Lifecycle

Milestone branch: `v2-m7-workspace-lifecycle` | Depends on: M6

---

### Phase 35: `pkg/workspace` and `pkg/worktrees`
**Issue**: #70 | **Branch**: `70-workspace-worktrees-pkg`
**Goal**: Package scaffolding provides the foundation types for workspace and worktree management
**Depends on**: Phase 34
**Requirements**: WKSP-01
**Success Criteria** (what must be TRUE):
  1. `pkg/workspace` package exists with core workspace types and register function
  2. `pkg/worktrees` package exists with worktree management types and register function
  3. Both packages follow the domain package convention (export `Register(*cobra.Command)` only)
**Canonical refs**: `.planning/v2/v2-milestones.md` (M7), `.planning/v2/v2-commands.md` (command tree), `.planning/codebase/ARCHITECTURE.md`
**Plans**: TBD

---

### Phase 36: `workspace create` and `list`
**Issue**: #71 | **Branch**: `71-workspace-create-list`
**Goal**: Users can create workspaces with scaffolded directories and list existing workspaces
**Depends on**: Phase 35
**Requirements**: WKSP-02, WKSP-03
**Success Criteria** (what must be TRUE):
  1. `git w workspace create` scaffolds directory structure under `workspaces/`
  2. Creates `[[workspace]]` block in `.gitw`
  3. Generates `AGENTS.md` stub at workspace level
  4. Creates `.planning/` directory at workspace level
  5. `git w workspace list` shows workspaces with active/archived workstream counts and `--json` support
**Canonical refs**: `.planning/v2/v2-commands.md` (workspace create/list), `.planning/v2/v2-milestones.md` (M7)
**Plans**: TBD

---

### Phase 37: `workstream create` with `--repo`
**Issue**: #72 | **Branch**: `72-workstream-create`
**Goal**: Users can create workstreams with worktrees checked out from specified repos
**Depends on**: Phase 36
**Requirements**: WKSP-04
**Success Criteria** (what must be TRUE):
  1. `git w workstream create` writes `.gitw-stream` manifest
  2. Generates workstream-level `AGENTS.md` with framework content
  3. Creates `.planning/` directory for workstream
  4. `--repo <n>:<branch>` creates worktree from specified repo on specified branch
  5. Calls `reconcileHooks` on affected repos
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream create), `.planning/v2/v2-milestones.md` (M7), `.planning/v2/v2-agent-interop.md` (AGENTS.md)
**Plans**: TBD

---

### Phase 38: `--worktree` flag (Pattern B)
**Issue**: #73 | **Branch**: `73-workstream-pattern-b`
**Goal**: Users can create workstreams with multiple named worktrees from the same repo
**Depends on**: Phase 37
**Requirements**: WKSP-05
**Success Criteria** (what must be TRUE):
  1. `--worktree name=<n>,repo=<r>,branch=<b>[,path=<p>][,scope=<s>]` parses correctly
  2. Multiple `--worktree` flags composable with `--repo` in same command
  3. Duplicate repo without unique `name` values produces error with `--worktree` hint
  4. Cross-modification warning generated in workstream `AGENTS.md` when same repo appears multiple times
**Canonical refs**: `.planning/v2/v2-infra-patterns.md` (Pattern B creation), `.planning/v2/v2-milestones.md` (M7)
**Plans**: TBD

---

### Phase 39: `workstream list`, `status`, `switch`
**Issue**: #74 | **Branch**: `74-workstream-list-status`
**Goal**: Users can navigate between workstreams and see their current state
**Depends on**: Phase 38
**Requirements**: WKSP-06
**Success Criteria** (what must be TRUE):
  1. `git w workstream list` shows all workstreams with status and worktree counts
  2. `git w workstream status` shows worktrees with branch, dirty state, and scope info
  3. Multi-worktree repos display as `<repo> / <name>` format
  4. `git w workstream switch` changes active workstream context
  5. All commands support `--json` output
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream list/status/switch), `.planning/v2/v2-milestones.md` (M7)
**Plans**: TBD

---

### Phase 40: `workstream worktree add`
**Issue**: #75 | **Branch**: `75-workstream-worktree-add`
**Goal**: Users can add worktrees to an existing workstream after creation
**Depends on**: Phase 39
**Requirements**: WKSP-07
**Success Criteria** (what must be TRUE):
  1. `git w workstream worktree add` adds a worktree to the active workstream
  2. `--worktree-name` and `--scope` flags support Pattern B additions
  3. Updates `.gitw-stream` manifest with new worktree entry
  4. Calls `reconcileHooks` on affected repo
**Canonical refs**: `.planning/v2/v2-commands.md` (worktree add), `.planning/v2/v2-milestones.md` (M7)
**Plans**: TBD

---

### Phase 41: `git w restore` worktrees
**Issue**: #76 | **Branch**: `76-restore-worktrees`
**Goal**: Users can re-materialize missing worktrees on a new machine or after corruption
**Depends on**: Phase 40
**Requirements**: WKSP-08
**Success Criteria** (what must be TRUE):
  1. `git w restore` scans active workstreams for missing worktrees
  2. Missing worktrees are re-created via `git worktree add`
  3. `git worktree repair` is called to fix bidirectional links
  4. Already-present worktrees are skipped (idempotent)
**Canonical refs**: `.planning/v2/v2-commands.md` (restore), `.planning/v2/v2-milestones.md` (M7)
**Plans**: TBD

---

### 📋 M8: Infra Patterns (A + B)

Milestone branch: `v2-m8-infra-patterns` | Depends on: M7

---

### Phase 42: `--branch` and `--branch-map` on repo add
**Issue**: #77 | **Branch**: `77-repo-add-branch-map`
**Goal**: Users can create env alias repos with branch tracking during repo add
**Depends on**: Phase 41
**Requirements**: INFRA-01
**Success Criteria** (what must be TRUE):
  1. `git w repo add --branch <b>` creates a repo alias tracking specified branch
  2. `--branch-map env1:branch1,env2:branch2` creates multiple aliases in one command
  3. Auto-sets `upstream` field linking aliases to upstream repo
  4. Aliases are cloned and checked out to their tracked branches
**Canonical refs**: `.planning/v2/v2-infra-patterns.md` (Pattern A creation), `.planning/v2/v2-commands.md` (repo add)
**Plans**: TBD

---

### Phase 43: `ResolveEnvGroup` and `--env-group`
**Issue**: #78 | **Branch**: `78-resolve-env-group`
**Goal**: Env groups can be expanded from upstream name to all alias repos for workstream creation
**Depends on**: Phase 42
**Requirements**: INFRA-02
**Success Criteria** (what must be TRUE):
  1. `ResolveEnvGroup(upstream)` returns all alias repos for the upstream name
  2. `--env-group <upstream>[:<branch>]` works in `git w workstream create`
  3. Expansion stores explicit `[[worktree]]` entries in `.gitw-stream`
  4. Branch defaults to workstream name; overridable with `:<branch>` suffix
**Canonical refs**: `.planning/v2/v2-infra-patterns.md` (env-group expansion), `.planning/v2/v2-milestones.md` (M8)
**Plans**: TBD

---

### Phase 44: `--upstream` filter
**Issue**: #79 | **Branch**: `79-upstream-filter`
**Goal**: Users can filter repo list and status display to show alias repos grouped by upstream
**Depends on**: Phase 43
**Requirements**: INFRA-03
**Success Criteria** (what must be TRUE):
  1. `git w repo list --upstream <n>` shows only repos with matching upstream field
  2. `git w status --repo <upstream>` shows all aliases grouped under upstream name
  3. Grouped display uses consistent formatting with env annotation
**Canonical refs**: `.planning/v2/v2-infra-patterns.md` (Pattern A display), `.planning/v2/v2-commands.md`
**Plans**: TBD

---

### Phase 45: Pattern B name/path validation
**Issue**: #80 | **Branch**: `80-pattern-b-validation`
**Goal**: Pattern B worktree entries are validated for uniqueness with actionable errors
**Depends on**: Phase 44
**Requirements**: INFRA-04
**Success Criteria** (what must be TRUE):
  1. `name` uniqueness enforced within workstream at creation time
  2. `path` uniqueness enforced within workstream at creation time
  3. Duplicate repo without disambiguation produces error suggesting `--worktree` flag
**Canonical refs**: `.planning/v2/v2-infra-patterns.md` (Pattern B validation), `.planning/v2/v2-schema.md` (gitw-stream validation)
**Plans**: TBD

---

### Phase 46: Pattern B scope display
**Issue**: #81 | **Branch**: `81-pattern-b-scope-display`
**Goal**: Scope information is visible in status and cross-modification warnings appear in AGENTS.md
**Depends on**: Phase 45
**Requirements**: INFRA-05
**Success Criteria** (what must be TRUE):
  1. `git w workstream status` shows scope per worktree for multi-worktree repos
  2. `git w status --repo <n>` shows named worktrees with name and scope
  3. Workstream `AGENTS.md` includes cross-modification warning when same repo appears multiple times
  4. Warning lists each name, branch, and scope explicitly
**Canonical refs**: `.planning/v2/v2-infra-patterns.md` (Pattern B display), `.planning/v2/v2-agent-interop.md` (AGENTS.md warnings)
**Plans**: TBD

---

### Phase 47: Mirror push naming for aliases
**Issue**: #82 | **Branch**: `82-mirror-push-naming`
**Goal**: Alias repos share a single personal remote repo named after the upstream
**Depends on**: Phase 46
**Requirements**: INFRA-06
**Success Criteria** (what must be TRUE):
  1. Mirror push for alias repos uses upstream repo name on personal remote
  2. All aliases for the same upstream push to the same remote repo
  3. Branch names in the remote repo correspond to each alias's tracked branch
**Canonical refs**: `.planning/v2/v2-infra-patterns.md` (Pattern A mirror push), `.planning/v2/v2-remote-management.md`
**Plans**: TBD

---

### 📋 M9: Agent Context Layer

Milestone branch: `v2-m9-agent-context` | Depends on: M8

---

### Phase 48: `pkg/agents` package
**Issue**: #83 | **Branch**: `83-agents-pkg`
**Goal**: Agent interop layer provides extensible framework interface with GSD as first implementation
**Depends on**: Phase 47
**Requirements**: AGNT-01
**Success Criteria** (what must be TRUE):
  1. `SpecFramework` interface defines methods for prohibition content, init instructions, and command references
  2. `GSDFramework` implements `SpecFramework` with GSD-specific strings
  3. `FrameworkFor(name)` looks up single framework by name; error on unknown
  4. `FrameworksFor(names)` resolves slice; error on first unknown name
  5. Pure generator functions accept `[]SpecFramework` for AGENTS.md and CONTEXT.md content
**Canonical refs**: `.planning/v2/v2-agent-interop.md` (pkg/agents spec), `.planning/v2/v2-milestones.md` (M9)
**Plans**: TBD

---

### Phase 49: `git w context rebuild`
**Issue**: #84 | **Branch**: `84-context-rebuild`
**Goal**: Users can regenerate all agent context files with framework-specific content
**Depends on**: Phase 48
**Requirements**: AGNT-02
**Success Criteria** (what must be TRUE):
  1. Regenerates `CONTEXT.md` at meta-repo level
  2. Regenerates three-level `AGENTS.md` (meta-repo, workspace, workstream)
  3. Framework-specific content generated from all active frameworks in declaration order
  4. Env-group summaries included in context output
  5. Auto-commits regenerated files
**Canonical refs**: `.planning/v2/v2-agent-interop.md` (context rebuild), `.planning/v2/v2-commands.md` (context rebuild)
**Plans**: TBD

---

### Phase 50: `git w agent context --json`
**Issue**: #85 | **Branch**: `85-agent-context-json`
**Goal**: Agent tools can query structured JSON about the current workspace context
**Depends on**: Phase 49
**Requirements**: AGNT-03
**Success Criteria** (what must be TRUE):
  1. CWD-based scope resolution determines context level (meta-repo, workspace, workstream)
  2. JSON output includes workstream, env_groups, capabilities, commands, agentic_frameworks blocks
  3. Each worktree entry includes `name` and `scope` fields
  4. `env_groups` include `create_hint` field per group
  5. `agentic_frameworks` is an array (not single string)
**Canonical refs**: `.planning/v2/v2-agent-interop.md` (agent context --json contract), `.planning/v2/v2-milestones.md` (M9)
**Plans**: TBD

---

### 📋 M10: Ship Pipeline

Milestone branch: `v2-m10-ship-pipeline` | Depends on: M9

---

### Phase 51: Ship dirty check
**Issue**: #86 | **Branch**: `86-ship-dirty-check`
**Goal**: Ship pipeline validates worktree cleanliness before any shipping operations
**Depends on**: Phase 50
**Requirements**: SHIP-01
**Success Criteria** (what must be TRUE):
  1. `git w workstream ship` checks all worktrees for uncommitted changes
  2. Dirty worktrees produce warning with list of affected files
  3. User can proceed or abort after warning
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream ship), `.planning/v2/v2-milestones.md` (M10)
**Plans**: TBD

---

### Phase 52: Ship `--push-all`
**Issue**: #87 | **Branch**: `87-ship-push-all`
**Goal**: Ship can lift push protection and push all worktree branches to origin
**Depends on**: Phase 51
**Requirements**: SHIP-02
**Success Criteria** (what must be TRUE):
  1. `--push-all` adds `origin` to workstream remote whitelist in `.git/.gitw`
  2. Calls `reconcileHooks` to update hooks with new whitelist
  3. Pushes all worktree branches to origin
  4. Records `shipped_at` timestamp and updates workstream status
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream ship), `.planning/v2/v2-remote-management.md` (push protection lift)
**Plans**: TBD

---

### Phase 53: Ship `--open-prs`
**Issue**: #88 | **Branch**: `88-ship-open-prs`
**Goal**: Ship can open one PR per worktree branch on GitHub
**Depends on**: Phase 52
**Requirements**: SHIP-03
**Success Criteria** (what must be TRUE):
  1. `--open-prs` opens one PR per worktree branch on GitHub origin
  2. PR titles and bodies include workstream and worktree context
  3. PR URLs are recorded in `.gitw-stream` manifest
  4. Pattern B: scope noted in PR description for multi-worktree repos
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream ship --open-prs), `.planning/v2/v2-milestones.md` (M10)
**Plans**: TBD

---

### Phase 54: Ship `--dry-run`
**Issue**: #89 | **Branch**: `89-ship-dry-run`
**Goal**: Users can preview what ship would do without executing any operations
**Depends on**: Phase 53
**Requirements**: SHIP-04
**Success Criteria** (what must be TRUE):
  1. `--dry-run` shows what would happen for each step (push, PR, squash)
  2. No mutations occur (no pushes, no PRs, no config changes)
  3. Output clearly indicates dry-run mode
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream ship --dry-run)
**Plans**: TBD

---

### Phase 55: Ship `--squash` and backup
**Issue**: #100 | **Branch**: `100-ship-squash-backup`
**Goal**: Ship can create backup branches on personal remote and squash worktree histories
**Depends on**: Phase 54
**Requirements**: SHIP-05
**Success Criteria** (what must be TRUE):
  1. `--squash` creates backup branches on personal remote before squashing
  2. `pre_ship_branches` recorded in `.gitw-stream` manifest
  3. Squash produces one commit per worktree with combined message
  4. Backup branches are verifiable (contain full pre-squash history)
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream ship --squash), `.planning/v2/v2-milestones.md` (M10)
**Plans**: TBD

---

### 📋 M11: Close and Archival

Milestone branch: `v2-m11-close-archival` | Depends on: M10

---

### Phase 56: Close worktree removal
**Issue**: #90 | **Branch**: `90-close-worktree-removal`
**Goal**: Workstream close removes worktrees and cleans up push protection hooks
**Depends on**: Phase 55
**Requirements**: CLOSE-01
**Success Criteria** (what must be TRUE):
  1. `git w workstream close` removes all worktrees for the workstream
  2. Calls `reconcileHooks` to remove push protection for affected repos
  3. Worktree removal uses `git worktree remove`
  4. Checks shipped status before allowing close
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream close), `.planning/v2/v2-milestones.md` (M11)
**Plans**: TBD

---

### Phase 57: Close archive move
**Issue**: #91 | **Branch**: `91-close-archive`
**Goal**: Closed workstreams are archived with planning state preserved
**Depends on**: Phase 56
**Requirements**: CLOSE-02
**Success Criteria** (what must be TRUE):
  1. Moves workstream directory from `active/` to `archived/`
  2. Updates `.gitw-stream` manifest status to `"archived"`
  3. `.planning/` directory preserved in full in archived location
  4. `AGENTS.md` updated to reflect archived status
  5. Triggers context rebuild
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream close), `.planning/v2/v2-milestones.md` (M11)
**Plans**: TBD

---

### Phase 58: Close `--no-archive`
**Issue**: #92 | **Branch**: `92-close-no-archive`
**Goal**: Users can permanently delete a workstream directory instead of archiving
**Depends on**: Phase 57
**Requirements**: CLOSE-03
**Success Criteria** (what must be TRUE):
  1. `--no-archive` prompts for explicit confirmation before deletion
  2. Deletes workstream directory entirely (no move to archived)
  3. Still performs worktree removal and hook cleanup before deletion
**Canonical refs**: `.planning/v2/v2-commands.md` (workstream close --no-archive), `.planning/v2/v2-milestones.md` (M11)
**Plans**: TBD

---

### 📋 M12: Migration

Milestone branch: `v2-m12-migrate` | Depends on: M1 only (parallel with M2-M11)

---

### Phase 59: `MigrationPlan` and `DetectV1`
**Issue**: #93 | **Branch**: `93-migrate-plan-detect`
**Goal**: Migration package can detect v1 configs and create a structured migration plan
**Depends on**: Phase 11 (M1 complete)
**Requirements**: MIG-01
**Success Criteria** (what must be TRUE):
  1. `DetectV1` identifies v1 config by presence of `[[workgroup]]` blocks or non-`repos/<n>` paths
  2. `MigrationPlan` type captures all required moves, conversions, and pre-flight checks
  3. Plan includes path moves, workgroup-to-workstream conversions, and config rewrites
**Canonical refs**: `.planning/v2/v2-migration.md` (detection logic, plan type), `.planning/v2/v2-milestones.md` (M12)
**Plans**: TBD

---

### Phase 60: `ReportPlan` formatting
**Issue**: #94 | **Branch**: `94-migrate-report`
**Goal**: Migration plan can be displayed as a human-readable report
**Depends on**: Phase 59
**Requirements**: MIG-02
**Success Criteria** (what must be TRUE):
  1. `ReportPlan` produces formatted output showing all planned actions
  2. Report shows source and destination for each path move
  3. Report shows workgroup-to-workstream conversion details
  4. Output uses `output.Writef` pattern
**Canonical refs**: `.planning/v2/v2-migration.md` (report format), `.planning/v2/v2-milestones.md` (M12)
**Plans**: TBD

---

### Phase 61: `ApplyPlan` with pre-flight abort
**Issue**: #95 | **Branch**: `95-migrate-apply`
**Goal**: Migration plan executes with safety checks that abort on collisions or bare repos
**Depends on**: Phase 60
**Requirements**: MIG-03
**Success Criteria** (what must be TRUE):
  1. Pre-flight checks detect path collisions (target already exists)
  2. Pre-flight checks detect bare repos (unsupported for migration)
  3. Abort on any pre-flight failure with actionable error
  4. `git worktree repair` called after every directory move
  5. Config rewritten with v2 schema after all moves complete
**Canonical refs**: `.planning/v2/v2-migration.md` (apply sequence, pre-flight), `.planning/v2/v2-milestones.md` (M12)
**Plans**: TBD

---

### Phase 62: `git w migrate` command
**Issue**: #96 | **Branch**: `96-migrate-command`
**Goal**: Users can run migration as report-only or with `--apply` to execute
**Depends on**: Phase 61
**Requirements**: MIG-04
**Success Criteria** (what must be TRUE):
  1. `git w migrate` (no flags) runs report-only mode showing what would change
  2. `git w migrate --apply` executes the migration plan
  3. Command follows domain package convention with `Register` in `register.go`
  4. Clear messaging distinguishes report mode from apply mode
**Canonical refs**: `.planning/v2/v2-migration.md` (command spec), `.planning/v2/v2-commands.md` (migrate)
**Plans**: TBD

---

### Phase 63: Migration end-to-end tests
**Issue**: #97 | **Branch**: `97-migrate-tests`
**Goal**: Migration is verified end-to-end including config round-trip and directory moves
**Depends on**: Phase 62
**Requirements**: MIG-05
**Success Criteria** (what must be TRUE):
  1. Config round-trip test: v1 config -> detect -> plan -> apply -> v2 config loads cleanly
  2. Path move test: repos moved from arbitrary paths to `repos/<n>` with worktree repair
  3. Workgroup-to-workstream conversion test: workgroups become workstreams under `legacy` workspace
  4. Collision abort test: target path already exists -> clean abort, no partial state
  5. Bare repo abort test: bare repo detected -> clean abort with manual resolution instructions
**Canonical refs**: `.planning/v2/v2-migration.md` (unit tests), `.planning/v2/v2-milestones.md` (M12)
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute sequentially within milestones: 1 -> 2 -> ... -> 11, then 12 -> 13 -> 14, etc.
M12 (Phases 59-63) can run in parallel after M1 (Phases 1-11) completes.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Add `[[workspace]]` block | M1 | 0/? | Not started | - |
| 2. Add `track_branch`/`upstream` | M1 | 0/? | Not started | - |
| 3. Enforce `repos/<n>` path | M1 | 0/? | Not started | - |
| 4. Add `[[remote]]`/`[[remote.branch_rule]]` | M1 | 0/? | Not started | - |
| 5. Add `[[sync_pair]]` parsing | M1 | 0/? | Not started | - |
| 6. Add `[[workstream]]` root block | M1 | 0/? | Not started | - |
| 7. Two-file config merge | M1 | 1/2 | In Progress|  |
| 8. Parse `.gitw-stream` manifest | M1 | 0/? | Not started | - |
| 9. Default remotes cascade | M1 | 0/? | Not started | - |
| 10. Detect v1 `[[workgroup]]` | M1 | 0/? | Not started | - |
| 11. `UpdatePreservingComments` round-trip | M1 | 0/? | Not started | - |
| 12. `BranchInfo` + glob package | M2 | 0/? | Not started | - |
| 13. `EvaluateRule` pure function | M2 | 0/? | Not started | - |
| 14. Rule criteria tests | M2 | 0/? | Not started | - |
| 15. `sync_pair` fan-out executor | M3 | 0/? | Not started | - |
| 16. Resolve effective remote list | M3 | 0/? | Not started | - |
| 17. `track_branch` pull target | M3 | 0/? | Not started | - |
| 18. Sync flags | M3 | 0/? | Not started | - |
| 19. Sync output + state file | M3 | 0/? | Not started | - |
| 20. Branch rule eval in sync push | M3 | 0/? | Not started | - |
| 21. `git w remote list` | M4 | 0/? | Not started | - |
| 22. API providers | M4 | 0/? | Not started | - |
| 23. `git w remote add` | M4 | 0/? | Not started | - |
| 24. `git w remote status` | M4 | 0/? | Not started | - |
| 25. `git w remote remove` | M4 | 0/? | Not started | - |
| 26. Unified status command | M5 | 0/? | Not started | - |
| 27. Status filter flags | M5 | 0/? | Not started | - |
| 28. Env-group display in status | M5 | 0/? | Not started | - |
| 29. Status `--json` output | M5 | 0/? | Not started | - |
| 30. `branch checkout --from` | M5 | 0/? | Not started | - |
| 31. `reconcileHooks` in sync | M6 | 0/? | Not started | - |
| 32. `reconcileHooks` function | M6 | 0/? | Not started | - |
| 33. `hook pre-push` subcommand | M6 | 0/? | Not started | - |
| 34. Push protection integration test | M6 | 0/? | Not started | - |
| 35. `pkg/workspace` + `pkg/worktrees` | M7 | 0/? | Not started | - |
| 36. `workspace create` + `list` | M7 | 0/? | Not started | - |
| 37. `workstream create` + `--repo` | M7 | 0/? | Not started | - |
| 38. `--worktree` flag (Pattern B) | M7 | 0/? | Not started | - |
| 39. `workstream list`/`status`/`switch` | M7 | 0/? | Not started | - |
| 40. `workstream worktree add` | M7 | 0/? | Not started | - |
| 41. `git w restore` worktrees | M7 | 0/? | Not started | - |
| 42. `--branch`/`--branch-map` on repo add | M8 | 0/? | Not started | - |
| 43. `ResolveEnvGroup` + `--env-group` | M8 | 0/? | Not started | - |
| 44. `--upstream` filter | M8 | 0/? | Not started | - |
| 45. Pattern B name/path validation | M8 | 0/? | Not started | - |
| 46. Pattern B scope display | M8 | 0/? | Not started | - |
| 47. Mirror push naming for aliases | M8 | 0/? | Not started | - |
| 48. `pkg/agents` package | M9 | 0/? | Not started | - |
| 49. `git w context rebuild` | M9 | 0/? | Not started | - |
| 50. `git w agent context --json` | M9 | 0/? | Not started | - |
| 51. Ship dirty check | M10 | 0/? | Not started | - |
| 52. Ship `--push-all` | M10 | 0/? | Not started | - |
| 53. Ship `--open-prs` | M10 | 0/? | Not started | - |
| 54. Ship `--dry-run` | M10 | 0/? | Not started | - |
| 55. Ship `--squash` + backup | M10 | 0/? | Not started | - |
| 56. Close worktree removal | M11 | 0/? | Not started | - |
| 57. Close archive move | M11 | 0/? | Not started | - |
| 58. Close `--no-archive` | M11 | 0/? | Not started | - |
| 59. `MigrationPlan` + `DetectV1` | M12 | 0/? | Not started | - |
| 60. `ReportPlan` formatting | M12 | 0/? | Not started | - |
| 61. `ApplyPlan` + pre-flight abort | M12 | 0/? | Not started | - |
| 62. `git w migrate` command | M12 | 0/? | Not started | - |
| 63. Migration end-to-end tests | M12 | 0/? | Not started | - |
