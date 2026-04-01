# v2 Issue Map

Reference for planner agents (GSD or similar). Maps each GitHub milestone and issue to its logical number, branch name, and dependencies. Use this file to name and number work units precisely when executing the v2 implementation plan.

---

## Naming rules

- **Milestone branches**: `v2-m<N>-<kebab-desc>` (e.g. `v2-m1-config-schema`)
- **Issue branches**: `<issue-number>-<kebab-desc>` (e.g. `36-add-workspace-block`)
- Issue numbers are GitHub issue numbers (`#36`, `#37`, …). They are fixed — do not renumber.
- Branch names are derived from the issue/milestone title; use the exact names in the tables below.

---

## Milestone summary

| # | Name | Branch | Depends on | Issues |
|---|------|--------|------------|--------|
| 1 | V2 M1: Config schema + loader | `v2-m1-config-schema` | — | 11 |
| 2 | V2 M2: Branch rule engine | `v2-m2-branch-rules` | M1 | 3 |
| 3 | V2 M3: git w sync multi-remote fan-out | `v2-m3-sync-fanout` | M2 | 6 |
| 4 | V2 M4: git w remote subcommand | `v2-m4-remote-subcommand` | M3 | 5 |
| 5 | V2 M5: git w status + checkout --from | `v2-m5-status-checkout` | M4 | 5 |
| 6 | V2 M6: Workstream push protection | `v2-m6-push-protection` | M5 | 4 |
| 7 | V2 M7: Workspace and workstream lifecycle | `v2-m7-workspace-lifecycle` | M6 | 7 |
| 8 | V2 M8: Infra repo patterns (Pattern A + B) | `v2-m8-infra-patterns` | M7 | 6 |
| 9 | V2 M9: Agent context layer | `v2-m9-agent-context` | M8 | 3 |
| 10 | V2 M10: Ship pipeline | `v2-m10-ship-pipeline` | M9 | 5 |
| 11 | V2 M11: Close and archival | `v2-m11-close-archival` | M10 | 3 |
| 12 | V2 M12: git w migrate | `v2-m12-migrate` | M1 only (parallel) | 5 |

---

## M1 — Config schema + loader

Milestone branch: `v2-m1-config-schema` · Depends on: none

| Issue # | Title | Branch |
|---------|-------|--------|
| #36 | Add `[[workspace]]` block to config schema | `36-add-workspace-block` |
| #37 | Add `track_branch` and `upstream` fields to `[[repo]]` | `37-track-branch-upstream` |
| #38 | Enforce `repos/<n>` path convention with v1 warning | `38-repos-path-convention` |
| #39 | Add `[[remote]]` and `[[remote.branch_rule]]` parsing | `39-remote-branch-rule-parsing` |
| #40 | Add `[[sync_pair]]` parsing with cycle detection | `40-sync-pair-cycle-detection` |
| #41 | Add `[[workstream]]` root config block | `41-workstream-root-block` |
| #42 | Implement two-file config merge with field-level semantics | `42-two-file-config-merge` |
| #43 | Parse and validate `.gitw-stream` manifest | `43-gitw-stream-manifest` |
| #44 | Add `[workspace]` default_remotes cascade resolution | `44-default-remotes-cascade` |
| #45 | Detect v1 `[[workgroup]]` blocks at load time | `45-detect-v1-workgroup` |
| #46 | `UpdatePreservingComments` round-trip tests for all v2 fields | `46-round-trip-tests` |

---

## M2 — Branch rule engine

Milestone branch: `v2-m2-branch-rules` · Depends on: M1

| Issue # | Title | Branch |
|---------|-------|--------|
| #47 | Add `BranchInfo` type and internal glob package | `47-branchinfo-glob-package` |
| #48 | Implement `EvaluateRule` pure function | `48-evaluate-rule` |
| #49 | Table-driven tests for all rule criteria combinations | `49-rule-criteria-tests` |

---

## M3 — git w sync multi-remote fan-out

Milestone branch: `v2-m3-sync-fanout` · Depends on: M2

| Issue # | Title | Branch |
|---------|-------|--------|
| #50 | Implement `sync_pair` fan-out executor with errgroup | `50-sync-pair-fanout` |
| #51 | Resolve effective remote list per repo in sync | `51-resolve-remote-list` |
| #52 | Support `track_branch` as pull target in sync | `52-track-branch-pull` |
| #54 | Add sync flags: `--remote`, `--workspace`, `--workstream`, `--no-push`, `--push-wip`, `--dry-run` | `54-sync-flags` |
| #55 | Implement per-remote grouped output and state file writes | `55-sync-output-state` |
| #98 | Wire branch rule evaluation into sync push phase | `98-sync-branch-rule-eval` |

---

## M4 — git w remote subcommand

Milestone branch: `v2-m4-remote-subcommand` · Depends on: M3

| Issue # | Title | Branch |
|---------|-------|--------|
| #56 | Implement `git w remote list` | `56-remote-list` |
| #59 | Implement Gitea/Forgejo and GitHub API providers | `59-api-providers` |
| #60 | Implement `git w remote add` (wizard + non-interactive) | `60-remote-add` |
| #61 | Implement `git w remote status` | `61-remote-status` |
| #99 | Implement `git w remote remove` | `99-remote-remove` |

---

## M5 — git w status + checkout --from

Milestone branch: `v2-m5-status-checkout` · Depends on: M4

| Issue # | Title | Branch |
|---------|-------|--------|
| #62 | Merge `git w info` and `git w status` into unified status command | `62-unified-status` |
| #63 | Add `--workspace`, `--workstream`, `--repo` filter flags to status | `63-status-filter-flags` |
| #64 | Add env-group display and available-branch hints to status | `64-status-env-group` |
| #65 | Add `--json` output to `git w status` | `65-status-json` |
| #66 | Implement `git w checkout --from <remote>` | `66-checkout-from` |

---

## M6 — Workstream push protection

Milestone branch: `v2-m6-push-protection` · Depends on: M5

| Issue # | Title | Branch |
|---------|-------|--------|
| #53 | Add `reconcileHooks` side effect to `git w sync` | `53-reconcile-hooks-sync` |
| #67 | Implement `reconcileHooks` internal function | `67-reconcile-hooks` |
| #68 | Implement `git-w hook pre-push` subcommand | `68-hook-pre-push` |
| #69 | Integration test: direct git push from protected worktree is blocked | `69-push-protection-test` |

---

## M7 — Workspace and workstream lifecycle

Milestone branch: `v2-m7-workspace-lifecycle` · Depends on: M6

| Issue # | Title | Branch |
|---------|-------|--------|
| #70 | Add `pkg/workspace` and `pkg/worktrees` packages | `70-workspace-worktrees-pkg` |
| #71 | Implement `git w workspace create` and `list` | `71-workspace-create-list` |
| #72 | Implement `git w workstream create` with `--repo` flag | `72-workstream-create` |
| #73 | Add `--worktree` flag to workstream create (Pattern B) | `73-workstream-pattern-b` |
| #74 | Implement `git w workstream list`, `status`, and `switch` | `74-workstream-list-status` |
| #75 | Implement `git w workstream worktree add` | `75-workstream-worktree-add` |
| #76 | Extend `git w restore` to re-materialize worktrees | `76-restore-worktrees` |

---

## M8 — Infra repo patterns (Pattern A + B)

Milestone branch: `v2-m8-infra-patterns` · Depends on: M7

| Issue # | Title | Branch |
|---------|-------|--------|
| #77 | Add `--branch` and `--branch-map` flags to `git w repo add` | `77-repo-add-branch-map` |
| #78 | Implement `ResolveEnvGroup` and `--env-group` flag | `78-resolve-env-group` |
| #79 | Add `--upstream` filter to `git w repo list` and status | `79-upstream-filter` |
| #80 | Pattern B: validate `[[worktree]]` name/path uniqueness | `80-pattern-b-validation` |
| #81 | Pattern B: scope display in status and cross-mod warning in AGENTS.md | `81-pattern-b-scope-display` |
| #82 | Mirror push naming for alias repos | `82-mirror-push-naming` |

---

## M9 — Agent context layer

Milestone branch: `v2-m9-agent-context` · Depends on: M8

| Issue # | Title | Branch |
|---------|-------|--------|
| #83 | Add `pkg/agents` package with pure generator functions | `83-agents-pkg` |
| #84 | Implement `git w context rebuild` | `84-context-rebuild` |
| #85 | Implement `git w agent context` with `--json` output | `85-agent-context-json` |

---

## M10 — Ship pipeline

Milestone branch: `v2-m10-ship-pipeline` · Depends on: M9

| Issue # | Title | Branch |
|---------|-------|--------|
| #86 | Implement dirty worktree detection in workstream ship | `86-ship-dirty-check` |
| #87 | Implement push protection lift and `--push-all` in workstream ship | `87-ship-push-all` |
| #88 | Implement `--open-prs` in workstream ship | `88-ship-open-prs` |
| #89 | Add `--dry-run` to workstream ship | `89-ship-dry-run` |
| #100 | Implement `--squash` and pre-ship backup branch in workstream ship | `100-ship-squash-backup` |

---

## M11 — Close and archival

Milestone branch: `v2-m11-close-archival` · Depends on: M10

| Issue # | Title | Branch |
|---------|-------|--------|
| #90 | Implement workstream close: worktree removal and hook cleanup | `90-close-worktree-removal` |
| #91 | Implement workstream close: archive move and manifest update | `91-close-archive` |
| #92 | Add `--no-archive` flag to workstream close | `92-close-no-archive` |

---

## M12 — git w migrate

Milestone branch: `v2-m12-migrate` · Depends on: M1 only (runs parallel to M2–M11)

| Issue # | Title | Branch |
|---------|-------|--------|
| #93 | Add `pkg/migrate`: `MigrationPlan` and `DetectV1` | `93-migrate-plan-detect` |
| #94 | Add `pkg/migrate`: `ReportPlan` formatting | `94-migrate-report` |
| #95 | Add `pkg/migrate`: `ApplyPlan` with pre-flight abort | `95-migrate-apply` |
| #96 | Add `git w migrate` command with `--apply` flag | `96-migrate-command` |
| #97 | Migration unit tests: end-to-end with config round-trip | `97-migrate-tests` |
