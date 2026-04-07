# git-w v2

## What This Is

A major version upgrade of git-w, the Go CLI tool that manages multiple git repos via `git w <cmd>`. v2 replaces the workgroup model with a two-level workspace/workstream hierarchy, adds multi-destination remote management with push protection, supports flexible infra repo patterns (branch-per-env and folder-per-env), and introduces an agent interop layer so AI coding tools can operate within git-w-managed environments. Ships as a single compiled binary with a `git w migrate` upgrade path from v1.

## Core Value

Multi-repo orchestration that keeps repos organized, synced, and safe from accidental pushes while giving AI agents full visibility into the workspace structure.

## Current Milestone: M1 Config Schema + Loader

**Goal:** Implement the complete v2 TOML config schema — new block types, two-file merge, field-level semantics, and v1 compatibility detection.

**Target features:**
- `[[workspace]]` block with name, description, repos list
- `[[remote]]` and `[[remote.branch_rule]]` blocks with all specified fields
- `[[sync_pair]]` blocks with cycle detection at load time
- `[[workstream]]` root config blocks for remote overrides
- Two-file merge (`.gitw` + `.git/.gitw`) with field-level semantics
- `.gitw-stream` manifest with `[[worktree]]` entries (name, path, scope)
- `[metarepo]` default_remotes cascade resolution (metarepo → workstream → repo)
- v1 `[[workgroup]]` detection with actionable error directing to `git w migrate`
- `agentic_frameworks` field validation against known registry
- `UpdatePreservingComments` round-trip fidelity for all v2 fields
- `repos/<n>` path convention enforcement with v1 warning

## Requirements

### Validated

- ✓ CLI plugin system (`git w <cmd>` via git plugin) -- existing
- ✓ TOML config file (`.gitw`) with filesystem discovery -- existing
- ✓ Multi-repo parallel execution (`RunFanOut`) -- existing
- ✓ Repo management (add, clone, list) -- existing
- ✓ Branch operations (create, checkout) across repos -- existing
- ✓ Cross-repo commit with `--workgroup` flag -- existing
- ✓ Arbitrary git command execution (`git w exec`) -- existing
- ✓ Config local overrides (`.gitw.local`) -- existing
- ✓ Shell completion -- existing
- ✓ GoReleaser + Homebrew distribution -- existing
- ✓ Root `[[workstream]]` config block contract + loader parse/validate/normalize (`CFG-06`) -- validated in Phase 06
- ✓ Two-file config merge with field-level semantics (`.gitw` + `.git/.gitw`) (`CFG-07`) -- validated in Phase 07
- ✓ `.gitw-stream` manifest with `[[worktree]]` entries (name, path, scope) + `LoadStream` loader (`CFG-08`) -- validated in Phase 08
- ✓ v1 `[[workgroup]]` detection: hard error at load time directing user to `git w migrate` (`CFG-10`) -- validated in Phase 10
- ✓ `UpdatePreservingComments` round-trip fidelity for all v2 fields + `interface{}` tech debt resolved (`CFG-12`) -- validated in Phase 11
- ✓ M1 phases 01–05 and 09 verified passed; Phase 02 SUMMARY.md files reconstructed; all 6 phases confirmed complete (`CFG-01`, `CFG-02`, `CFG-03`, `CFG-04`, `CFG-05`, `CFG-09`, `CFG-11`) -- validated in Phase 12

### Active

- [ ] v2 config schema (`[[workspace]]`, `[[remote]]`, `[[sync_pair]]`, `[[workstream]]`, `.gitw-stream`)
- [ ] Branch rule engine (glob patterns, criteria combos, four action tiers)
- [ ] Multi-remote sync fan-out with `[[sync_pair]]` routing
- [ ] Remote management (`git w remote add/list/status/remove`) with Gitea/GitHub providers
- [ ] Unified `git w status` replacing `info` + `status`
- [ ] `git w branch checkout --from <remote>`
- [ ] Workstream push protection via `reconcileHooks` and pre-push hook
- [ ] Workspace lifecycle (`create`, `list`)
- [ ] Workstream lifecycle (`create`, `list`, `status`, `switch`, `worktree add`)
- [ ] Infra Pattern A: branch-per-env repo aliases (`track_branch`, `upstream`, `--env-group`)
- [ ] Infra Pattern B: folder-per-env multi-worktree (`name`, `path`, `scope` fields)
- [ ] Agent context layer (`pkg/agents`, `SpecFramework` interface, `GSDFramework`)
- [ ] `git w context rebuild` generating three-level `AGENTS.md` + `CONTEXT.md`
- [ ] `git w agent context --json` with CWD-based scope resolution
- [ ] Ship pipeline (`git w workstream ship` with squash, backup, push, PR opening)
- [ ] Close and archival (`git w workstream close` with worktree removal, hook cleanup, archive)
- [ ] `git w migrate` for v1-to-v2 upgrade path
- [ ] `repos/<n>` path convention enforcement with v1 warning
  - [ ] v1 `[[workgroup]]` detection with actionable error -- **done, Phase 10**
- [ ] `agentic_frameworks` config field with registry validation
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

**Existing codebase:** Go 1.26 CLI with cobra command tree, domain-oriented package architecture (`pkg/workspace`, `pkg/repo`, `pkg/worktree`, `pkg/branch`, `pkg/workgroup`, `pkg/git`), TOML config, parallel execution via `parallel.RunFanOut`, and test infrastructure using testify suites.

**v2 consolidates three design tracks:**
1. Remote management and multi-destination sync
2. Workspace and workstream management (replacing workgroups)
3. Flexible infra repo patterns (branch-per-env and folder-per-env)

**Branching strategy:** `main` (v1 stable) -> `v2` (long-lived base) -> `v2-m<N>-<slug>` (milestone branches) -> `<issue>-<desc>` (issue branches). Milestones sequential M1-M11; M12 parallel after M1.

**GSD workflow mapping:** Each v2 milestone is one GSD milestone. Each GitHub issue is one GSD phase. GSD branching_strategy is `none` (commits directly to active branch; branch/PR management is manual).

**Spec documents:** Full design specs live in `.planning/v2/` covering schema, commands, remote management, infra patterns, agent interop, migration, and milestones.

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
| Major version bump (2.0) | Workgroup retirement, command cuts, path convention changes are breaking | -- Pending |
| Workgroups replaced by workstreams | Two-level hierarchy (workspace/workstream) is more flexible than flat workgroups | -- Pending |
| `.gitw-stream` self-contained manifests | Workstream config travels with the workstream directory, not in root `.gitw` | -- Pending |
| Scope filters at call site (no ambient context) | Explicit `--workspace`/`--workstream`/`--repo` flags instead of `git w context` scope-setter | -- Pending |
| `[[sync_pair]]` explicit routing | Fan-out sync needs clear ref routing; explicit pairs are auditable | -- Pending |
| Pre-push hook for push protection | WIP must not reach org remotes; hook is the only reliable git-level enforcement | -- Pending |
| `SpecFramework` interface for agent interop | Multiple agentic frameworks (GSD, future speckit/openspec) without coupling | -- Pending |
| No delete commands | Repos, workspaces, remotes are not deleted via CLI; edit config directly | -- Pending |
| Net command reduction 39 to 27 | Simpler surface; cut redundant commands (`info`/`status` merged, `fetch`/`pull` merged into `sync`) | -- Pending |
| M12 parallel after M1 | Migration needs config types but nothing else; can develop concurrently | -- Pending |

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
*Last updated: 2026-04-07 — Phase 12 complete (M1 phases 01–05 and 09 all verified passed; Phase 02 SUMMARY.md files reconstructed; CFG-01/02/03/04/05/09/11 all validated)*
