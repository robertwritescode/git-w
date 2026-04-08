# git-w v2

## What This Is

A major version upgrade of git-w, the Go CLI tool that manages multiple git repos via `git w <cmd>`. v2 replaces the workgroup model with a two-level workspace/workstream hierarchy, adds multi-destination remote management with push protection, supports flexible infra repo patterns (branch-per-env and folder-per-env), and introduces an agent interop layer so AI coding tools can operate within git-w-managed environments. Ships as a single compiled binary with a `git w migrate` upgrade path from v1.

## Core Value

Multi-repo orchestration that keeps repos organized, synced, and safe from accidental pushes while giving AI agents full visibility into the workspace structure.

## Current Milestone: M2 Branch Rule Engine

**Goal:** Implement the branch rule evaluation engine — `BranchInfo` type, internal glob matching (`*` and `**`), and `EvaluateRule` pure function with all four action tiers (allow, block, warn, require-flag).

**Target features:**
- `BranchInfo` type with branch name, remote name, and tracking metadata
- Internal glob package supporting `*` (no `/` crossing) and `**` (crosses `/`) patterns
- `EvaluateRule` pure function: four action tiers, all criteria combinations
- Table-driven tests covering every criteria × action tier combination

**M1 shipped:** 2026-04-08 — full v2 config schema + loader (13 phases, 26 plans) · [archive](.planning/milestones/v2.0-m1-ROADMAP.md)

## Requirements

### Validated

- ✓ CLI plugin system (`git w <cmd>` via git plugin) — existing
- ✓ TOML config file (`.gitw`) with filesystem discovery — existing
- ✓ Multi-repo parallel execution (`RunFanOut`) — existing
- ✓ Repo management (add, clone, list) — existing
- ✓ Branch operations (create, checkout) across repos — existing
- ✓ Cross-repo commit with `--workgroup` flag — existing
- ✓ Arbitrary git command execution (`git w exec`) — existing
- ✓ Config local overrides (`.gitw.local`) — existing
- ✓ Shell completion — existing
- ✓ GoReleaser + Homebrew distribution — existing
- ✓ `[[workspace]]` blocks with name, description, repos list; `agentic_frameworks` validation against framework registry (`CFG-01`, `CFG-11`) — v2.0 M1, Phase 01
- ✓ `track_branch` and `upstream` fields on `[[repo]]` for env alias annotation; `IsAlias()` method; `validateAliasFields` co-presence + uniqueness checks (`CFG-02`) — v2.0 M1, Phase 02
- ✓ Load-time warnings for non-conforming repo paths with `repos/<n>` suggestion; `cfg.Warnings` field; `warnNonConformingRepoPaths` (`CFG-03`) — v2.0 M1, Phase 03
- ✓ `[[remote]]` and `[[remote.branch_rule]]` blocks with full field set; `validateRemotes`; branch rules preserve declaration order (`CFG-04`) — v2.0 M1, Phase 04
- ✓ `[[sync_pair]]` blocks with DFS cycle detection at load time; `validateSyncPairFields`; `detectSyncCycles` (`CFG-05`) — v2.0 M1, Phase 05
- ✓ `[[workstream]]` root config block with strict-key validation, remote reference integrity, duplicate rejection (`CFG-06`) — v2.0 M1, Phase 06
- ✓ Two-file config merge (`.gitw` + `.git/.gitw`) with field-level semantics; private file wins on conflict; `private = true` rejected (`CFG-07`) — v2.0 M1, Phase 07
- ✓ `.gitw-stream` manifest with `[[worktree]]` entries (name, path, scope); `LoadStream` with parse-default-validate pipeline (`CFG-08`) — v2.0 M1, Phase 08
- ✓ `[metarepo] default_remotes` cascade: metarepo → workstream → repo (innermost wins); `ResolveRepoRemotes` / `ResolveWorkstreamRemotes` (`CFG-09`) — v2.0 M1, Phase 09
- ✓ v1 `[[workgroup]]` detection: hard error at load time directing user to `git w migrate` (`CFG-10`) — v2.0 M1, Phase 10
- ✓ `UpdatePreservingComments` round-trip fidelity for all v2 fields; `interface{}` → `any` tech debt resolved; `applySmartUpdate` error propagation fixed (`CFG-12`) — v2.0 M1, Phase 11
- ✓ Post-merge validation: `revalidateWorkstreamRemotes` after merge (INT-01); `sync_pair` from/to remote name validation (INT-02); path warnings preserved on alias error (INT-03) — v2.0 M1, Phase 13

### Active

- [ ] Branch rule engine (glob patterns, criteria combos, four action tiers) — M2
- [ ] Multi-remote sync fan-out with `[[sync_pair]]` routing — M3
- [ ] Remote management (`git w remote add/list/status/remove`) with Gitea/GitHub providers — M4
- [ ] Unified `git w status` replacing `info` + `status` — M5
- [ ] `git w branch checkout --from <remote>` — M5
- [ ] Workstream push protection via `reconcileHooks` and pre-push hook — M6
- [ ] Workspace lifecycle (`create`, `list`) — M7
- [ ] Workstream lifecycle (`create`, `list`, `status`, `switch`, `worktree add`) — M7
- [ ] Infra Pattern A: branch-per-env repo aliases (`track_branch`, `upstream`, `--env-group`) — M8
- [ ] Infra Pattern B: folder-per-env multi-worktree (`name`, `path`, `scope` fields) — M8
- [ ] Agent context layer (`pkg/agents`, `SpecFramework` interface, `GSDFramework`) — M9
- [ ] `git w context rebuild` generating three-level `AGENTS.md` + `CONTEXT.md` — M9
- [ ] `git w agent context --json` with CWD-based scope resolution — M9
- [ ] Ship pipeline (`git w workstream ship` with squash, backup, push, PR opening) — M10
- [ ] Close and archival (`git w workstream close` with worktree removal, hook cleanup, archive) — M11
- [ ] `git w migrate` for v1-to-v2 upgrade path — M12
- [ ] Net command reduction: 39 to 27 commands

### Out of Scope

- Token storage via keychain or 1Password CLI -- post-v2.0; `token_env` sufficient
- `git w workstream ship --open-prs` for non-GitHub remotes -- post-v2.0
- Cross-workstream dependency tracking -- post-v2.0
- `git w context rebuild` heuristic repo descriptions from README parsing -- post-v2.0
- Forgejo API divergence from Gitea -- treated as compatible for v2.0
- `[[sync_pair]]` ref filtering beyond globs -- post-v2.0
- Per-worktree devcontainer support -- post-v2.0
- Pattern A promotion tracking (dev/test/prod chain awareness) -- post-v2.0
- Pattern B cross-PR linking -- post-v2.0
- Pattern B scope enforcement via pre-commit hook -- post-v2.0
- Delete/remove commands for repos or workspaces -- by design; edit config directly
- TUI framework (bubbletea/lipgloss) -- by design; plain formatted output only

## Context

**Codebase state (after M1):** Go 1.26 CLI, cobra command tree, domain-oriented package architecture (`pkg/workspace`, `pkg/repo`, `pkg/worktree`, `pkg/branch`, `pkg/workgroup`, `pkg/git`, `pkg/agents`). M1 added `pkg/agents` (SpecFramework interface + GSDFramework), significantly extended `pkg/config` (WorkspaceBlock, RemoteConfig, BranchRuleConfig, SyncPairConfig, WorkstreamConfig, WorkstreamManifest, WorktreeEntry, ShipState, StreamContext), and added `pkg/config/stream.go` (LoadStream). ~147 files changed, ~21k LOC added in M1.

**v2 consolidates three design tracks:**
1. Remote management and multi-destination sync
2. Workspace and workstream management (replacing workgroups)
3. Flexible infra repo patterns (branch-per-env and folder-per-env)

**Branching strategy:** `main` (v1 stable) → `v2` (long-lived base) → `v2-m<N>-<slug>` (milestone branches) → `<issue>-<desc>` (issue branches). Milestones sequential M1-M11; M12 parallel after M1.

**GSD workflow mapping:** Each v2 milestone is one GSD milestone. Each GitHub issue is one GSD phase. GSD branching_strategy is `none` (commits directly to active branch; branch/PR management is manual).

**Spec documents:** Full design specs live in `.planning/v2/` covering schema, commands, remote management, infra patterns, agent interop, migration, and milestones.

**Known tech debt from M1:**
- `pkg/agents` SpecFramework interface has zero call sites outside `pkg/config` validation — consumer call sites are M9 deliverable
- `track_branch`/`upstream` alias fields parsed and validated, but not yet consumed by any sync or status command (M3, M5 work)
- `repos/<n>` path warnings generated at load time but no auto-migration (M12 work)

## Constraints

- **Tech stack**: Go 1.26, cobra, go-toml/v2. No new runtime dependencies without justification. Single compiled binary.
- **No TUI**: Plain formatted output via `text/tabwriter`. No bubbletea/lipgloss.
- **Breaking changes**: v2.0 is a major version. Workgroup retirement, command surface reduction, directory layout migration all require `git w migrate` path.
- **Compatibility**: v1 configs must be detected at load time with actionable migration instructions.
- **Output**: `output.Writef` for stdout/stderr. No `fmt.Fprintf` directly.
- **Testing**: testify suites for shared setup/teardown, table-driven tests for combinatorial cases, `mage test` with race detector before marking work complete.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Major version bump (2.0) | Workgroup retirement, command cuts, path convention changes are breaking | — Pending |
| Workgroups replaced by workstreams | Two-level hierarchy (workspace/workstream) is more flexible than flat workgroups | — Pending |
| `.gitw-stream` self-contained manifests | Workstream config travels with the workstream directory, not in root `.gitw` | — Pending |
| Scope filters at call site (no ambient context) | Explicit `--workspace`/`--workstream`/`--repo` flags instead of `git w context` scope-setter | — Pending |
| `[[sync_pair]]` explicit routing | Fan-out sync needs clear ref routing; explicit pairs are auditable | — Pending |
| Pre-push hook for push protection | WIP must not reach org remotes; hook is the only reliable git-level enforcement | — Pending |
| `SpecFramework` interface for agent interop | Multiple agentic frameworks (GSD, future speckit/openspec) without coupling | ✓ Good — interface + GSDFramework shipped in M1, pkg/agents wired into config validation |
| No delete commands | Repos, workspaces, remotes are not deleted via CLI; edit config directly | — Pending |
| Net command reduction 39 to 27 | Simpler surface; cut redundant commands (`info`/`status` merged, `fetch`/`pull` merged into `sync`) | — Pending |
| M12 parallel after M1 | Migration needs config types but nothing else; can develop concurrently | ✓ Good — M1 complete, M12 can now start |
| Remotes/SyncPairs/Workstreams on WorkspaceConfig directly (no diskConfig split) | Consistent with WorkspaceBlock array-of-tables pattern; simpler loader wiring | ✓ Good — pattern held cleanly through Phases 04-06 |
| mergePrivateConfig between loadMainConfig and mergeLocalConfig | `.gitw.local` context must always win; private file is an intermediate merge | ✓ Good — ordering correct, plus post-merge re-validation added (Phase 13) |
| revalidateWorkstreamRemotes after mergePrivateConfig | Private file can add workstreams referencing private-only remotes; single-pass validation missed these | ✓ Good — INT-01 gap closed cleanly without duplicating full validation (Phase 13) |
| WorkstreamStatus follows typed string alias pattern | Consistent with BranchAction; avoids iota enum for string-serialized values | ✓ Good — consistent across config package |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? -> Move to Out of Scope with reason
2. Requirements validated? -> Move to Validated with phase reference
3. New requirements emerged? -> Add to Active
4. Decisions to log? -> Add to Key Decisions
5. "What This Is" still accurate? -> Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check -- still the right priority?
3. Audit Out of Scope -- reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-08 after v2.0 M1 milestone — M1 Config Schema + Loader complete (13 phases, 26 plans); all CFG requirements validated and archived; M2 Branch Rule Engine is next*
