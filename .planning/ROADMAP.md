# Roadmap: git-w v2

## Overview

git-w v2 replaces the workgroup model with workspace/workstream hierarchy, adds multi-remote sync with push protection, supports infra repo patterns (branch-per-env and folder-per-env), and introduces an agent interop layer. The roadmap follows the dependency chain: config types first (everything imports them), then branch rule engine, sync fan-out, remotes, status, hooks, workstream lifecycle, infra patterns, agent context, ship, close, and migration (parallel after M1). 63 phases across 12 milestones.

## Milestones

- ✅ **M1: Config Schema + Loader** — Phases 1-13 (shipped 2026-04-08) · [archive](.planning/milestones/v2.0-m1-ROADMAP.md)
- 📋 **M2: Branch Rule Engine** - Phases 15-17
- 📋 **M3: Sync Multi-Remote Fan-Out** - Phases 18-23
- 📋 **M4: Remote Subcommand** - Phases 24-28
- 📋 **M5: Status + Checkout --from** - Phases 29-33
- 📋 **M6: Push Protection** - Phases 34-37
- 📋 **M7: Workspace + Workstream Lifecycle** - Phases 38-44
- 📋 **M8: Infra Patterns (A + B)** - Phases 45-50
- 📋 **M9: Agent Context Layer** - Phases 51-53
- 📋 **M10: Ship Pipeline** - Phases 54-58
- 📋 **M11: Close and Archival** - Phases 59-61
- 📋 **M12: Migration** - Phases 62-66

## Phases

<details>
<summary>✅ M1: Config Schema + Loader (Phases 1-13) — SHIPPED 2026-04-08</summary>

- [x] Phase 1: Add `[[workspace]]` block (3/3 plans) — completed 2026-04-04
- [x] Phase 2: Add `track_branch` and `upstream` fields (2/2 plans) — completed 2026-04-04
- [x] Phase 3: Enforce `repos/<n>` path convention (1/1 plan) — completed 2026-04-04
- [x] Phase 4: Add `[[remote]]` and `[[remote.branch_rule]]` (2/2 plans) — completed 2026-04-04
- [x] Phase 5: Add `[[sync_pair]]` parsing (2/2 plans) — completed 2026-04-04
- [x] Phase 6: Add `[[workstream]]` root config block (2/2 plans) — completed 2026-04-05
- [x] Phase 7: Two-file config merge (2/2 plans) — completed 2026-04-05
- [x] Phase 8: Parse `.gitw-stream` manifest (2/2 plans) — completed 2026-04-06
- [x] Phase 9: Default remotes cascade (2/2 plans) — completed 2026-04-07
- [x] Phase 10: Detect v1 `[[workgroup]]` blocks (1/1 plan) — completed 2026-04-07
- [x] Phase 11: `UpdatePreservingComments` round-trip (2/2 plans) — completed 2026-04-07
- [x] Phase 12: Verify M1 Phases 01–05 and 09 (4/4 plans) — completed 2026-04-07
- [x] Phase 13: Fix post-merge validation and sync_pair remote validation (1/1 plan) — completed 2026-04-08

Full details: [.planning/milestones/v2.0-m1-ROADMAP.md](.planning/milestones/v2.0-m1-ROADMAP.md)

</details>

### 📋 M2: Branch Rule Engine (In Progress / Planned)

Milestone branch: `v2-m2-branch-rules` | Depends on: M1

- [ ] Phase 15: `BranchInfo` type and glob package
- [ ] Phase 16: `EvaluateRule` pure function
- [ ] Phase 17: Rule criteria combination tests

### 📋 M3: Sync Multi-Remote Fan-Out (Planned)

Milestone branch: `v2-m3-sync-fanout` | Depends on: M2

- [ ] Phase 18: `sync_pair` fan-out executor
- [ ] Phase 19: Resolve effective remote list
- [ ] Phase 20: `track_branch` as pull target
- [ ] Phase 21: Sync flags
- [ ] Phase 22: Sync output and state file
- [ ] Phase 23: Branch rule eval in sync push

### 📋 M4: Remote Subcommand (Planned)

Milestone branch: `v2-m4-remote-subcommand` | Depends on: M3

- [ ] Phase 24: `git w remote list`
- [ ] Phase 25: API providers
- [ ] Phase 26: `git w remote add`
- [ ] Phase 27: `git w remote status`
- [ ] Phase 28: `git w remote remove`

### 📋 M5: Status + Checkout --from (Planned)

Milestone branch: `v2-m5-status-checkout` | Depends on: M4

- [ ] Phase 29: Unified status command
- [ ] Phase 30: Status filter flags
- [ ] Phase 31: Env-group display in status
- [ ] Phase 32: Status `--json` output
- [ ] Phase 33: `branch checkout --from`

### 📋 M6: Push Protection (Planned)

Milestone branch: `v2-m6-push-protection` | Depends on: M5

- [ ] Phase 34: `reconcileHooks` in sync
- [ ] Phase 35: `reconcileHooks` function
- [ ] Phase 36: `hook pre-push` subcommand
- [ ] Phase 37: Push protection integration test

### 📋 M7: Workspace + Workstream Lifecycle (Planned)

Milestone branch: `v2-m7-workspace-lifecycle` | Depends on: M6

- [ ] Phase 38: `pkg/workspace` and `pkg/worktrees`
- [ ] Phase 39: `workspace create` and `list`
- [ ] Phase 40: `workstream create` with `--repo`
- [ ] Phase 41: `--worktree` flag (Pattern B)
- [ ] Phase 42: `workstream list`, `status`, `switch`
- [ ] Phase 43: `workstream worktree add`
- [ ] Phase 44: `git w restore` worktrees

### 📋 M8: Infra Patterns (A + B) (Planned)

Milestone branch: `v2-m8-infra-patterns` | Depends on: M7

- [ ] Phase 45: `--branch` and `--branch-map` on repo add
- [ ] Phase 46: `ResolveEnvGroup` and `--env-group`
- [ ] Phase 47: `--upstream` filter
- [ ] Phase 48: Pattern B name/path validation
- [ ] Phase 49: Pattern B scope display
- [ ] Phase 50: Mirror push naming for aliases

### 📋 M9: Agent Context Layer (Planned)

Milestone branch: `v2-m9-agent-context` | Depends on: M8

- [ ] Phase 51: `pkg/agents` package
- [ ] Phase 52: `git w context rebuild`
- [ ] Phase 53: `git w agent context --json`

### 📋 M10: Ship Pipeline (Planned)

Milestone branch: `v2-m10-ship-pipeline` | Depends on: M9

- [ ] Phase 54: Ship dirty check
- [ ] Phase 55: Ship `--push-all`
- [ ] Phase 56: Ship `--open-prs`
- [ ] Phase 57: Ship `--dry-run`
- [ ] Phase 58: Ship `--squash` and backup

### 📋 M11: Close and Archival (Planned)

Milestone branch: `v2-m11-close-archival` | Depends on: M10

- [ ] Phase 59: Close worktree removal
- [ ] Phase 60: Close archive move
- [ ] Phase 61: Close `--no-archive`

### 📋 M12: Migration (Planned, parallel after M1)

Milestone branch: `v2-m12-migration` | Depends on: M1 only

- [ ] Phase 62: `MigrationPlan` and `DetectV1`
- [ ] Phase 63: `ReportPlan` formatting
- [ ] Phase 64: `ApplyPlan` with pre-flight abort
- [ ] Phase 65: `git w migrate` command
- [ ] Phase 66: Migration end-to-end tests

## Progress

| Phase | Milestone | Plans Complete | Status   | Completed  |
|-------|-----------|----------------|----------|------------|
| 1. Add `[[workspace]]` block | M1 | 3/3 | Complete | 2026-04-04 |
| 2. Add `track_branch`/`upstream` | M1 | 2/2 | Complete | 2026-04-04 |
| 3. Enforce `repos/<n>` convention | M1 | 1/1 | Complete | 2026-04-04 |
| 4. Add `[[remote]]`/`[[remote.branch_rule]]` | M1 | 2/2 | Complete | 2026-04-04 |
| 5. Add `[[sync_pair]]` parsing | M1 | 2/2 | Complete | 2026-04-04 |
| 6. Add `[[workstream]]` root block | M1 | 2/2 | Complete | 2026-04-05 |
| 7. Two-file config merge | M1 | 2/2 | Complete | 2026-04-05 |
| 8. Parse `.gitw-stream` manifest | M1 | 2/2 | Complete | 2026-04-06 |
| 9. Default remotes cascade | M1 | 2/2 | Complete | 2026-04-07 |
| 10. Detect v1 workgroup blocks | M1 | 1/1 | Complete | 2026-04-07 |
| 11. UpdatePreservingComments | M1 | 2/2 | Complete | 2026-04-07 |
| 12. Verify M1 phases | M1 | 4/4 | Complete | 2026-04-07 |
| 13. Fix post-merge validation | M1 | 1/1 | Complete | 2026-04-08 |
| 15. `BranchInfo` type and glob | M2 | 0/TBD | Not started | - |
| 16. `EvaluateRule` pure function | M2 | 0/TBD | Not started | - |
| 17. Rule criteria combination tests | M2 | 0/TBD | Not started | - |
| 18-23. Sync fan-out (6 phases) | M3 | 0/TBD | Not started | - |
| 24-28. Remote subcommand (5 phases) | M4 | 0/TBD | Not started | - |
| 29-33. Status + checkout (5 phases) | M5 | 0/TBD | Not started | - |
| 34-37. Push protection (4 phases) | M6 | 0/TBD | Not started | - |
| 38-44. Workspace lifecycle (7 phases) | M7 | 0/TBD | Not started | - |
| 45-50. Infra patterns (6 phases) | M8 | 0/TBD | Not started | - |
| 51-53. Agent context (3 phases) | M9 | 0/TBD | Not started | - |
| 54-58. Ship pipeline (5 phases) | M10 | 0/TBD | Not started | - |
| 59-61. Close and archival (3 phases) | M11 | 0/TBD | Not started | - |
| 62-66. Migration (5 phases) | M12 | 0/TBD | Not started | - |
