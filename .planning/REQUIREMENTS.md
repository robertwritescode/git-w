# Requirements: git-w v2

**Defined:** 2026-04-01
**Core Value:** Multi-repo orchestration that keeps repos organized, synced, and safe from accidental pushes while giving AI agents full visibility into the workspace structure.

## v1 Requirements

Requirements for v2.0 release. Each maps to roadmap phases (GSD phases = GitHub issues).

### Config Schema

- [x] **CFG-01**: User can define `[[workspace]]` blocks with name, description, and repos list in `.gitw`
- [ ] **CFG-02**: User can add `track_branch` and `upstream` fields to `[[repo]]` blocks for env aliases
- [ ] **CFG-03**: Tool enforces `repos/<n>` path convention and warns on v1 paths with migration suggestion
- [ ] **CFG-04**: User can define `[[remote]]` and `[[remote.branch_rule]]` blocks with all specified fields
- [ ] **CFG-05**: User can define `[[sync_pair]]` blocks with cycle detection at load time
- [ ] **CFG-06**: User can define `[[workstream]]` root config blocks for lightweight remote overrides
- [ ] **CFG-07**: Tool merges `.gitw` and `.git/.gitw` with field-level semantics (private file wins on conflicts)
- [ ] **CFG-08**: User can define `.gitw-stream` manifest with `[[worktree]]` entries including `name`, `path`, `scope` fields
- [ ] **CFG-09**: Tool resolves `[metarepo] default_remotes` cascade: metarepo -> workstream -> repo (innermost wins)
- [ ] **CFG-10**: Tool detects v1 `[[workgroup]]` blocks at load time with actionable error directing to `git w migrate`
- [x] **CFG-11**: Tool validates `[metarepo] agentic_frameworks` field against known framework registry
- [ ] **CFG-12**: `UpdatePreservingComments` round-trips all v2 fields without losing comments or field order

### Branch Rules

- [ ] **BRULE-01**: Tool provides `BranchInfo` type and internal glob package supporting `*` and `**` patterns
- [ ] **BRULE-02**: Tool provides `EvaluateRule` pure function supporting all four action tiers (allow, block, warn, require-flag)
- [ ] **BRULE-03**: All criteria combinations (pattern, untracked, explicit) work correctly with all action tiers

### Sync

- [ ] **SYNC-01**: `git w sync` executes `[[sync_pair]]` fan-out with parallel fetch then push via errgroup
- [ ] **SYNC-02**: Tool resolves effective remote list per repo using cascade (metarepo -> workstream -> repo)
- [ ] **SYNC-03**: `git w sync` uses `track_branch` as pull target for alias repos
- [ ] **SYNC-04**: `git w sync` supports `--remote`, `--workspace`, `--workstream`, `--no-push`, `--push-wip`, `--dry-run` flags
- [ ] **SYNC-05**: `git w sync` prints per-remote grouped output and writes state to `.git/git-w-state.json`
- [ ] **SYNC-06**: `git w sync` evaluates branch rules during push phase (repo overrides -> remote rules -> default allow)

### Remote Management

- [ ] **RMT-01**: `git w remote list` displays configured remotes with `--json` support
- [ ] **RMT-02**: Tool provides Gitea/Forgejo and GitHub API providers for repo existence check and creation
- [ ] **RMT-03**: `git w remote add` works as interactive wizard and via non-interactive flags
- [ ] **RMT-04**: `git w remote status` shows connectivity and last-sync timestamps from state file
- [ ] **RMT-05**: `git w remote remove` removes config and local git remotes without API deletion

### Status

- [ ] **STAT-01**: `git w status` merges v1 `info` and `status` into unified display with repos, workstreams, and remote sections
- [ ] **STAT-02**: `git w status` supports `--workspace`, `--workstream`, `--repo` filter flags
- [ ] **STAT-03**: `git w status` displays env-group aliases grouped under upstream name with available-branch hints
- [ ] **STAT-04**: `git w status` supports `--json` output
- [ ] **STAT-05**: `git w branch checkout <branch> --from <remote>` fetches and creates local branch from named remote

### Push Protection

- [ ] **HOOK-01**: `reconcileHooks` installs/updates/removes git-w managed block in `.git/hooks/pre-push` idempotently
- [ ] **HOOK-02**: `git w sync` calls `reconcileHooks` on all repos as a side effect
- [ ] **HOOK-03**: `git-w hook pre-push` subcommand evaluates worktree path against workstream manifests and blocks unauthorized remotes
- [ ] **HOOK-04**: Direct `git push` from a protected worktree to a non-whitelisted remote is blocked (integration test verified)

### Workspace and Workstream

- [ ] **WKSP-01**: `pkg/workspace` and `pkg/worktrees` packages provide workspace and worktree management
- [ ] **WKSP-02**: `git w workspace create` scaffolds directories, writes `[[workspace]]` block, generates `AGENTS.md`, creates `.planning/`
- [ ] **WKSP-03**: `git w workspace list` shows workspaces with active/archived counts and `--json` support
- [ ] **WKSP-04**: `git w workstream create` writes `.gitw-stream`, generates `AGENTS.md`, creates `.planning/`, supports `--repo` flag
- [ ] **WKSP-05**: `git w workstream create` supports `--worktree` flag for Pattern B (same repo, multiple named worktrees)
- [ ] **WKSP-06**: `git w workstream list`, `status`, and `switch` work with `--json` support
- [ ] **WKSP-07**: `git w workstream worktree add` adds worktrees post-creation with `--worktree-name` and `--scope` flags
- [ ] **WKSP-08**: `git w restore` re-materializes missing worktrees for active workstreams via `git worktree repair`

### Infra Patterns

- [ ] **INFRA-01**: `git w repo add` supports `--branch` and `--branch-map` flags for creating env aliases
- [ ] **INFRA-02**: `ResolveEnvGroup` expands upstream name to all alias repos; `--env-group` flag works in workstream create
- [ ] **INFRA-03**: `git w repo list --upstream <n>` and `git w status --repo <upstream>` filter and group alias repos
- [ ] **INFRA-04**: Pattern B validates `[[worktree]]` name/path uniqueness with actionable error on duplicate repo without disambiguation
- [ ] **INFRA-05**: Pattern B scope displays in status and cross-modification warning generates in workstream AGENTS.md
- [ ] **INFRA-06**: Mirror push for alias repos uses upstream repo name (not alias name) on personal remote

### Agent Context

- [ ] **AGNT-01**: `pkg/agents` provides `SpecFramework` interface, `GSDFramework` implementation, and `FrameworkFor`/`FrameworksFor` registry
- [ ] **AGNT-02**: `git w context rebuild` regenerates `CONTEXT.md` and three-level `AGENTS.md` with framework-specific content
- [ ] **AGNT-03**: `git w agent context --json` emits workstream, env_groups, capabilities, commands, and agentic_frameworks blocks

### Ship Pipeline

- [ ] **SHIP-01**: `git w workstream ship` validates worktrees and warns on uncommitted changes
- [ ] **SHIP-02**: `git w workstream ship --push-all` lifts push protection and pushes all worktree branches to origin
- [ ] **SHIP-03**: `git w workstream ship --open-prs` opens one PR per worktree branch on GitHub and records URLs
- [ ] **SHIP-04**: `git w workstream ship --dry-run` shows what would happen without executing
- [ ] **SHIP-05**: `git w workstream ship --squash` creates backup branches on personal remote and squashes per worktree

### Close and Archival

- [ ] **CLOSE-01**: `git w workstream close` removes worktrees and cleans up pre-push hooks
- [ ] **CLOSE-02**: `git w workstream close` moves workstream directory to `archived/` and updates manifest status
- [ ] **CLOSE-03**: `git w workstream close --no-archive` deletes directory with explicit confirmation

### Migration

- [ ] **MIG-01**: `pkg/migrate` provides `MigrationPlan` type and `DetectV1` function for v1 config detection
- [ ] **MIG-02**: `pkg/migrate` provides `ReportPlan` for formatted migration report output
- [ ] **MIG-03**: `pkg/migrate` provides `ApplyPlan` with pre-flight abort on collisions and bare repos
- [ ] **MIG-04**: `git w migrate` command with `--apply` flag (report-only by default, execute with flag)
- [ ] **MIG-05**: Migration end-to-end tests verify config round-trip, path moves, and workgroup-to-workstream conversion

## v2 Requirements

Deferred to post-v2.0. Tracked but not in current roadmap.

### Post-Launch Enhancements

- **POST-01**: Token storage via keychain or 1Password CLI (currently `token_env` only)
- **POST-02**: `git w workstream ship --open-prs` for non-GitHub remotes (Gitea/Forgejo)
- **POST-03**: `git w context rebuild` heuristic repo descriptions from README parsing
- **POST-04**: Forgejo API divergence handling (currently treated as compatible with Gitea)
- **POST-05**: `[[sync_pair]]` ref filtering beyond globs (by age, exclude tags)
- **POST-06**: Per-worktree devcontainer support
- **POST-07**: Pattern A promotion tracking (dev->test->prod chain awareness)
- **POST-08**: Pattern B cross-PR linking when `--open-prs` opens multiple PRs for same repo
- **POST-09**: Pattern B scope enforcement via pre-commit hook (warn, not block)
- **POST-10**: `git w workstream ship` coordinated PR sequencing for Pattern B

## Out of Scope

| Feature | Reason |
|---------|--------|
| Delete/remove commands for repos, workspaces, remotes | Destructive operations on multi-repo state are dangerous; edit config directly |
| TUI framework (bubbletea/lipgloss) | Breaks non-TTY contexts, adds dependency weight; plain formatted output only |
| Ambient scope-setting command | Hidden state causes confusion; explicit flags at call site preferred |
| Cross-workstream dependency tracking | Belongs in project management tools, not git CLI |
| go-git library adoption | Contradicts existing os/exec architecture; would create two incompatible git execution paths |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CFG-01 | Phase 1 (M1 #36) | Complete |
| CFG-02 | Phase 2 (M1 #37) | Pending |
| CFG-03 | Phase 3 (M1 #38) | Pending |
| CFG-04 | Phase 4 (M1 #39) | Pending |
| CFG-05 | Phase 5 (M1 #40) | Pending |
| CFG-06 | Phase 6 (M1 #41) | Pending |
| CFG-07 | Phase 7 (M1 #42) | Pending |
| CFG-08 | Phase 8 (M1 #43) | Pending |
| CFG-09 | Phase 9 (M1 #44) | Pending |
| CFG-10 | Phase 10 (M1 #45) | Pending |
| CFG-11 | Phase 1 (M1 #36) | Complete |
| CFG-12 | Phase 11 (M1 #46) | Pending |
| BRULE-01 | Phase 12 (M2 #47) | Pending |
| BRULE-02 | Phase 13 (M2 #48) | Pending |
| BRULE-03 | Phase 14 (M2 #49) | Pending |
| SYNC-01 | Phase 15 (M3 #50) | Pending |
| SYNC-02 | Phase 16 (M3 #51) | Pending |
| SYNC-03 | Phase 17 (M3 #52) | Pending |
| SYNC-04 | Phase 18 (M3 #54) | Pending |
| SYNC-05 | Phase 19 (M3 #55) | Pending |
| SYNC-06 | Phase 20 (M3 #98) | Pending |
| RMT-01 | Phase 21 (M4 #56) | Pending |
| RMT-02 | Phase 22 (M4 #59) | Pending |
| RMT-03 | Phase 23 (M4 #60) | Pending |
| RMT-04 | Phase 24 (M4 #61) | Pending |
| RMT-05 | Phase 25 (M4 #99) | Pending |
| STAT-01 | Phase 26 (M5 #62) | Pending |
| STAT-02 | Phase 27 (M5 #63) | Pending |
| STAT-03 | Phase 28 (M5 #64) | Pending |
| STAT-04 | Phase 29 (M5 #65) | Pending |
| STAT-05 | Phase 30 (M5 #66) | Pending |
| HOOK-01 | Phase 32 (M6 #67) | Pending |
| HOOK-02 | Phase 31 (M6 #53) | Pending |
| HOOK-03 | Phase 33 (M6 #68) | Pending |
| HOOK-04 | Phase 34 (M6 #69) | Pending |
| WKSP-01 | Phase 35 (M7 #70) | Pending |
| WKSP-02 | Phase 36 (M7 #71) | Pending |
| WKSP-03 | Phase 36 (M7 #71) | Pending |
| WKSP-04 | Phase 37 (M7 #72) | Pending |
| WKSP-05 | Phase 38 (M7 #73) | Pending |
| WKSP-06 | Phase 39 (M7 #74) | Pending |
| WKSP-07 | Phase 40 (M7 #75) | Pending |
| WKSP-08 | Phase 41 (M7 #76) | Pending |
| INFRA-01 | Phase 42 (M8 #77) | Pending |
| INFRA-02 | Phase 43 (M8 #78) | Pending |
| INFRA-03 | Phase 44 (M8 #79) | Pending |
| INFRA-04 | Phase 45 (M8 #80) | Pending |
| INFRA-05 | Phase 46 (M8 #81) | Pending |
| INFRA-06 | Phase 47 (M8 #82) | Pending |
| AGNT-01 | Phase 48 (M9 #83) | Pending |
| AGNT-02 | Phase 49 (M9 #84) | Pending |
| AGNT-03 | Phase 50 (M9 #85) | Pending |
| SHIP-01 | Phase 51 (M10 #86) | Pending |
| SHIP-02 | Phase 52 (M10 #87) | Pending |
| SHIP-03 | Phase 53 (M10 #88) | Pending |
| SHIP-04 | Phase 54 (M10 #89) | Pending |
| SHIP-05 | Phase 55 (M10 #100) | Pending |
| CLOSE-01 | Phase 56 (M11 #90) | Pending |
| CLOSE-02 | Phase 57 (M11 #91) | Pending |
| CLOSE-03 | Phase 58 (M11 #92) | Pending |
| MIG-01 | Phase 59 (M12 #93) | Pending |
| MIG-02 | Phase 60 (M12 #94) | Pending |
| MIG-03 | Phase 61 (M12 #95) | Pending |
| MIG-04 | Phase 62 (M12 #96) | Pending |
| MIG-05 | Phase 63 (M12 #97) | Pending |

**Coverage:**
- v1 requirements: 63 total
- Mapped to phases: 63
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-01*
*Last updated: 2026-04-01 after initial definition*
