# git-w V2 Branching & Implementation Strategy

## Overview

- **Current stable version**: v1.6.0 on `main`
- **Goal**: Deliver a clean v2.0.0 cut-over without disrupting the v1.x stable surface
- **Method**: All v2 work happens on a long-lived `v2` branch and milestone sub-branches; `main` is untouched until final cut-over
- **Agent tooling**: Each issue is implemented by GSD using the workflow described below

---

## Branch Hierarchy

```
main  (v1.6.x — stable, Release Please continues here, Homebrew tap points here)
 └── v2  (long-lived base, protected — all milestone branches target this)
      ├── v2-m1-config-schema
      │    ├── 36-add-workspace-block         → PR → v2-m1-config-schema  (Closes #36)
      │    ├── 37-track-branch-upstream       → PR → v2-m1-config-schema  (Closes #37)
      │    ├── 38-repos-path-convention       → PR → v2-m1-config-schema  (Closes #38)
      │    ├── 39-remote-branch-rule-parsing  → PR → v2-m1-config-schema  (Closes #39)
      │    ├── 40-sync-pair-cycle-detection   → PR → v2-m1-config-schema  (Closes #40)
      │    ├── 41-workstream-root-block       → PR → v2-m1-config-schema  (Closes #41)
      │    ├── 42-two-file-config-merge       → PR → v2-m1-config-schema  (Closes #42)
      │    ├── 43-gitw-stream-manifest        → PR → v2-m1-config-schema  (Closes #43)
      │    ├── 44-default-remotes-cascade     → PR → v2-m1-config-schema  (Closes #44)
      │    ├── 45-detect-v1-workgroup         → PR → v2-m1-config-schema  (Closes #45)
      │    └── 46-round-trip-tests            → PR → v2-m1-config-schema  (Closes #46)
      │    └── Milestone PR: v2-m1-config-schema → v2  (closes V2 M1)
      │
      ├── v2-m2-branch-rules  (opens after M1 merges to v2)
      │    ├── 47-branchinfo-glob-package     → PR → v2-m2-branch-rules
      │    ├── 48-evaluate-rule               → PR → v2-m2-branch-rules
      │    └── 49-rule-criteria-tests         → PR → v2-m2-branch-rules
      │    └── Milestone PR: v2-m2-branch-rules → v2
      │
      ├── v2-m3-sync-fanout  (opens after M2 merges to v2)
      ├── v2-m4-remote-subcommand
      ├── v2-m5-status-checkout
      ├── v2-m6-push-protection
      ├── v2-m7-workspace-lifecycle
      ├── v2-m8-infra-patterns
      ├── v2-m9-agent-context
      ├── v2-m10-ship-pipeline
      ├── v2-m11-close-archival
      └── v2-m12-migrate  (can be unlocked in parallel after M1 merges; see Sequencing)

Cut-over: v2 → main PR → Release Please detects breaking changes → tags 2.0.0
```

---

## Naming Conventions

| Layer | Pattern | Example |
|---|---|---|
| Long-lived base | `v2` | `v2` |
| Milestone branch | `v2-m<N>-<short-slug>` | `v2-m1-config-schema` |
| Issue branch | `<issue-number>-<kebab-description>` | `36-add-workspace-block` |

Note: `v2/m*` naming is not usable — git ref path semantics prevent a branch named `v2/anything` from coexisting with a branch named `v2`.

Issue branches use the same naming convention as v1 feature branches — consistent with the existing repo history.

---

## GSD Issue Workflow (per issue)

**Before branching or planning any issue, read `.planning/v2/v2-issue-map.md`.** It is the authoritative source for every issue number, issue title, exact branch name, milestone branch, and dependency. Do not invent branch names or issue numbers — use only what is listed there.

1. **Branch**: create issue branch off the active milestone branch using the exact name from `v2-issue-map.md`
   - e.g. `git checkout -b 36-add-workspace-block` from `v2-m1-config-schema`
2. **Implement**: GSD implements the issue according to the issue spec, coding standards, and architecture docs
3. **Test**: `mage testfast` passes; `go vet ./...` clean
4. **PR**: GSD opens PR targeting the milestone branch (NOT `v2` directly, NOT `main`)
   - PR title: matches issue title exactly as listed in `v2-issue-map.md`
   - PR body: `Closes #<N>` — auto-closes the GitHub issue on merge
5. **Review + merge**: PR merges into milestone branch
6. **Next issue**: GSD moves to the next issue in the milestone (strictly sequential — no parallel issue branches within a milestone)

---

## Milestone Sequencing

**Default rule: strictly sequential.** A milestone branch is only opened after the previous milestone's PR has merged into `v2`.

**Dependency chain (implied order):**
M1 → M2 → M3 → M4 → M5 → M6 → M7 → M8 → M9 → M10 → M11

**Parallel unlock rule:** M12 (`git w migrate`) only depends on M1 (config loader). After M1 merges to `v2`, M12 may be explicitly unlocked to run in parallel with M2+. This is a conscious decision — update the Active State section below when unlocking.

**Milestone branch lifecycle:**
1. Branch created off `v2` at the start of milestone work
2. Issues merged in sequentially
3. Milestone PR opened targeting `v2`, body includes `Closes #<all issue numbers>`
4. Merged → milestone branch deleted → next milestone branch created

---

## Cut-Over Plan

When all 12 milestones are merged to `v2`:

1. Open PR: `v2 → main`
2. PR title: `feat!: git-w v2.0.0`
3. PR body describes the full v2 feature set
4. Release Please detects `feat!` (breaking change) commits accumulated on `v2` and generates `2.0.0` release
5. Homebrew tap auto-updates via the existing GoReleaser + release workflow

**v1.x patches**: Any critical v1 bug fixes during v2 development are committed directly to `main` (not `v2`). After the fix merges to `main`, cherry-pick to `v2` if the affected code is shared.

---

## Active State

> This section is updated as work progresses. GSD should read this first to understand the current position.

| Field | Value |
|---|---|
| Active milestone | V2 M1: Config schema + loader |
| Milestone branch | `v2/m1-config-schema` |
| Current issue branch | none |
| Next issue to implement | #44 — Add `[metarepo]` default_remotes cascade resolution |
| M12 parallel unlock | not yet unlocked |

---

## Milestone & Issue Map

> **This section is a summary for orientation only.** The authoritative source with exact branch names, issue numbers, and per-milestone issue lists is `.planning/v2/v2-issue-map.md`. Read that file before branching or planning any issue.

### V2 M1: Config schema + loader (milestone #1)
11 issues — foundational; all other milestones depend on this.

| # | Title |
|---|---|
| 36 | Add `[[workspace]]` block to config schema |
| 37 | Add `track_branch` and `upstream` fields to `[[repo]]` |
| 38 | Enforce `repos/<n>` path convention with v1 warning |
| 39 | Add `[[remote]]` and `[[remote.branch_rule]]` parsing |
| 40 | Add `[[sync_pair]]` parsing with cycle detection |
| 41 | Add `[[workstream]]` root config block |
| 42 | Implement two-file config merge with field-level semantics |
| 43 | Parse and validate `.gitw-stream` manifest |
| 44 | Add `[metarepo]` default_remotes cascade resolution |
| 45 | Detect v1 `[[workgroup]]` blocks at load time |
| 46 | `UpdatePreservingComments` round-trip tests for all v2 fields |

### V2 M2: Branch rule engine (milestone #2)
Depends on: M1

| # | Title |
|---|---|
| 47 | Add `BranchInfo` type and internal glob package |
| 48 | Implement `EvaluateRule` pure function |
| 49 | Table-driven tests for all rule criteria combinations |

### V2 M3: git w sync multi-remote fan-out (milestone #3)
Depends on: M2

| # | Title |
|---|---|
| 50 | Implement `sync_pair` fan-out executor with errgroup |
| 51 | Resolve effective remote list per repo in sync |
| 52 | Support `track_branch` as pull target in sync |
| 53 | Add `reconcileHooks` side effect to `git w sync` |
| 54 | Add sync flags: `--remote`, `--workspace`, `--workstream`, `--no-push`, `--push-wip`, `--dry-run` |
| 55 | Implement per-remote grouped output and state file writes |

### V2 M4: git w remote subcommand (milestone #4)
Depends on: M3

| # | Title |
|---|---|
| 56 | Implement `git w remote list` |
| 59 | Implement Gitea/Forgejo and GitHub API providers |
| 60 | Implement `git w remote add` (wizard + non-interactive) |
| 61 | Implement `git w remote status` |

### V2 M5: git w status + checkout --from (milestone #5)
Depends on: M4

| # | Title |
|---|---|
| 62 | Merge `git w info` and `git w status` into unified status command |
| 63 | Add `--workspace`, `--workstream`, `--repo` filter flags to status |
| 64 | Add env-group display and available-branch hints to status |
| 65 | Add `--json` output to `git w status` |
| 66 | Implement `git w checkout --from <remote>` |

### V2 M6: Workstream push protection (milestone #6)
Depends on: M5

| # | Title |
|---|---|
| 67 | Implement `reconcileHooks` internal function |
| 68 | Implement `git-w hook pre-push` subcommand |
| 69 | Integration test: direct git push from protected worktree is blocked |

### V2 M7: Workspace and workstream lifecycle (milestone #7)
Depends on: M6

| # | Title |
|---|---|
| 70 | Add `pkg/workspace` and `pkg/worktrees` packages |
| 71 | Implement `git w workspace create` and `list` |
| 72 | Implement `git w workstream create` with `--repo` flag |
| 73 | Add `--worktree` flag to workstream create (Pattern B) |
| 74 | Implement `git w workstream list`, `status`, and `switch` |
| 75 | Implement `git w workstream worktree add` |
| 76 | Extend `git w restore` to re-materialize worktrees |

### V2 M8: Infra repo patterns (Pattern A + B) (milestone #8)
Depends on: M7

| # | Title |
|---|---|
| 77 | Add `--branch` and `--branch-map` flags to `git w repo add` |
| 78 | Implement `ResolveEnvGroup` and `--env-group` flag |
| 79 | Add `--upstream` filter to `git w repo list` and status |
| 80 | Pattern B: validate `[[worktree]]` name/path uniqueness |
| 81 | Pattern B: scope display in status and cross-mod warning in AGENTS.md |
| 82 | Mirror push naming for alias repos |

### V2 M9: Agent context layer (milestone #9)
Depends on: M8

| # | Title |
|---|---|
| 83 | Add `pkg/agents` package with pure generator functions |
| 84 | Implement `git w context rebuild` |
| 85 | Implement `git w agent context` with `--json` output |

### V2 M10: Ship pipeline (milestone #10)
Depends on: M9

| # | Title |
|---|---|
| 86 | Implement dirty worktree detection in workstream ship |
| 87 | Implement push protection lift and `--push-all` in workstream ship |
| 88 | Implement `--open-prs` in workstream ship |
| 89 | Add `--dry-run` to workstream ship |

### V2 M11: Close and archival (milestone #11)
Depends on: M10

| # | Title |
|---|---|
| 90 | Implement workstream close: worktree removal and hook cleanup |
| 91 | Implement workstream close: archive move and manifest update |
| 92 | Add `--no-archive` flag to workstream close |

### V2 M12: git w migrate (milestone #12)
Depends on: M1 only — can run in parallel with M2+ after M1 merges (requires explicit unlock)

| # | Title |
|---|---|
| 93 | Add `pkg/migrate`: `MigrationPlan` and `DetectV1` |
| 94 | Add `pkg/migrate`: `ReportPlan` formatting |
| 95 | Add `pkg/migrate`: `ApplyPlan` with pre-flight abort |
| 96 | Add `git w migrate` command with `--apply` flag |
| 97 | Migration unit tests: end-to-end with config round-trip |
