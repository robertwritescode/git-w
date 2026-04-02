# Feature Research

**Domain:** Multi-repo Git management CLI (v2 major upgrade)
**Researched:** 2026-04-01
**Confidence:** HIGH

## Feature Landscape

This analysis categorizes the v2 feature set against what the multi-repo management ecosystem expects. Competitor tools analyzed: `meta` (Node.js, 2.2k stars), `gita` (Python, 1.8k stars), `repo` (Android/Google), `myrepos`, `mu-repo`. None of these offer workspace/workstream hierarchy, push protection, infra repo patterns, or agent interop. These tools are largely limited to batch git operations and repo registration.

### Table Stakes (Users Expect These)

Features users of a multi-repo CLI assume exist. Missing these means the product feels incomplete or broken relative to v1.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **v2 config schema + loader** | Foundation for everything; users need config to work. v1 users need a working loader before any v2 feature is usable. | HIGH | M1. Two-file merge, field-level semantics, `[[workspace]]`, `[[repo]]` v2 fields, `[[remote]]`, `[[sync_pair]]`, `.gitw-stream` parsing. Most complex foundational piece. |
| **Multi-remote sync fan-out** | The core v2 value proposition. Users managing open source or cross-machine workflows need repos synced to multiple remotes. Every competitor does single-remote only; this is baseline for v2's story. | HIGH | M3. `[[sync_pair]]` routing, parallel fetch/push, branch rule evaluation, `track_branch` pull, grouped output. Depends on M1 + M2. |
| **Branch rule engine** | Enables push protection and sync safety. Without it, sync fan-out can't enforce branch-level policies on different remotes. | MEDIUM | M2. Pure function `EvaluateRule`, glob matching, four action tiers (`allow`/`block`/`warn`/`require-flag`), `untracked` + `explicit` criteria. Self-contained, testable. |
| **Remote management (`git w remote`)** | Users need to add/configure remotes without hand-editing TOML. Gitea/GitHub API provisioning for repo creation is expected if the tool manages multiple remotes. | HIGH | M4. Wizard flow, Gitea/GitHub/generic providers, repo existence check, remote upsert, initial mirror push, `[[sync_pair]]` wiring. Most external-facing complexity. |
| **Unified `git w status`** | v1 has both `info` and `status` which is confusing. Users expect one command to see everything: repos, workstreams, remote staleness. | MEDIUM | M5. Merges v1 `info`+`status`, env-group display, remote staleness from state file, available-branch hints, `--json` output. |
| **`git w branch checkout --from`** | Natural companion to multi-remote. Users on machine B need to pull branches from personal remote without manual git commands. | LOW | M5. Thin wrapper around fetch + checkout from named remote. |
| **Workspace lifecycle (`create`, `list`)** | Workspaces replace v1 groups. Users need to create/list them. Without this, the two-level hierarchy doesn't exist. | LOW | M7. Directory scaffolding, `[[workspace]]` block write, `AGENTS.md` stub generation. Simple file operations. |
| **Workstream lifecycle (create, list, status, switch)** | Workstreams replace v1 workgroups. The core organizational unit for v2. Without full lifecycle, multi-repo work units don't exist. | HIGH | M7. `.gitw-stream` write, worktree creation via `git worktree add`, `--repo`/`--env-group`/`--worktree` flag parsing, `reconcileHooks`, auto-commit. Most user-facing complexity. |
| **Workstream push protection** | The safety guarantee. WIP must not reach org remotes via direct `git push` from IDEs/agents. Git-level enforcement via pre-push hook is the only reliable mechanism. | HIGH | M6. `reconcileHooks`, hook install/append/update/remove, `git-w hook pre-push` subcommand, worktree path resolution, remote whitelist check. Critical safety feature. |
| **Ship pipeline** | Users need a way to complete work: squash, push, open PRs. Without ship, workstreams are a dead end. | HIGH | M10. Squash pass, backup branches on personal remote, push protection lift, `--push-all`, `--open-prs`, PR URL recording, status update. Complex multi-step workflow. |
| **Close and archival** | Workstreams need a clean end-of-life: remove worktrees, prune branches, archive planning state. Without close, the workspace fills with stale workstreams. | MEDIUM | M11. Worktree removal, hook cleanup, branch pruning prompts, directory move to `archived/`, `.planning/` preservation. |
| **v1 to v2 migration** | Breaking changes require an upgrade path. Users with existing v1 configs must be able to migrate without losing data or history. | MEDIUM | M12. Config detection, path migration, workgroup-to-workstream conversion, `git worktree repair`, collision/bare repo detection. Parallelizable after M1. |

### Differentiators (Competitive Advantage)

Features that set git-w v2 apart from every competitor. Not strictly required for a working tool, but are the reason to choose git-w over alternatives.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Infra Pattern A: branch-per-env repo aliases** | No multi-repo tool handles the branch-per-env infrastructure pattern. `track_branch` + `upstream` grouping + `--env-group` expansion is novel. Developers managing Terraform/Kubernetes across envs get first-class support. | MEDIUM | M8. `track_branch`/`upstream` fields, `--branch-map` on `repo add`, `ResolveEnvGroup`, `--env-group` in workstream create, alias-aware mirror push. |
| **Infra Pattern B: folder-per-env multi-worktree** | Multiple named worktrees from the same repo within one workstream, with `scope` advisory. No competitor handles this. Solves consolidated-infra repos where a ticket touches `environments/dev/` and `environments/prod/` simultaneously. | HIGH | M8. `name`/`path`/`scope` fields, `--worktree` flag parsing, duplicate-repo validation, `<repo> / <name>` display, cross-modification warnings in AGENTS.md, one PR per worktree on ship. |
| **Agent interop layer** | AI coding tools (GSD, Claude Code, opencode) are becoming primary users of CLIs. No multi-repo tool provides structured context for agents. Three-level `AGENTS.md` + `git w agent context --json` + `SpecFramework` interface makes git-w agent-first. | HIGH | M9. `pkg/agents`, `SpecFramework` Go interface, `GSDFramework` impl, three-level AGENTS.md generation, JSON context output with capabilities/commands/env_groups, framework registry. |
| **`git w context rebuild`** | Regenerating `CONTEXT.md` and all `AGENTS.md` files from current state gives both humans and agents an always-current system map. | MEDIUM | M9. Pure generator functions, idempotent, auto-called by workstream create/close. |
| **Two-level workspace/workstream hierarchy** | Competitors have flat grouping (meta: projects, gita: groups). Workspace → workstream hierarchy with active/archived split is more natural for product-area organization. | MEDIUM | Spread across M7. Directory convention, `.gitw-stream` self-contained manifests, CWD-based scope resolution. |
| **Planning state preservation** | `.planning/` directories at three levels (meta-repo, workspace, workstream) are committed and archived. Work history survives workstream close. No competitor preserves planning context. | LOW | Architectural decision. Directory creation + archival logic in M7/M11. |
| **Pre-ship squash with backup branches** | `--squash` in ship creates a backup branch on personal remote before squashing, then produces clean single commits for PR. Safety net for messy WIP history. | MEDIUM | M10. Divergence detection, backup branch push to personal, soft-reset + recommit. |
| **Command surface reduction (39 → 27)** | Simpler, more intentional CLI. v1 had redundant commands. Merging `info`/`status`, `fetch`/`pull`/`push` into `sync`, cutting `worktree`/`group`/`workgroup` families. | LOW | Spread across milestones. Primarily command registration and migration docs. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems. git-w deliberately does NOT build these.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Delete/remove commands for repos, workspaces, remotes** | "I want `git w repo remove`" feels natural. | Destructive operations on multi-repo state are dangerous. A typo could remove tracking for a repo used by multiple workstreams. The blast radius is too large for a quick command. | Edit `.gitw` directly; delete `repos/<n>` manually. Explicit config editing is safer for irreversible operations. |
| **TUI framework (bubbletea/lipgloss)** | Modern CLI tools have rich terminal UIs. | Adds significant dependency weight, breaks in non-TTY contexts (CI, agent runners), and forces a runtime model incompatible with single-binary simplicity. | Plain formatted output via `text/tabwriter`. `--json` for machine consumption. |
| **Ambient scope-setting command** | v1 had `git w context` to set active group. "I shouldn't have to pass `--workspace` every time." | Hidden state causes confusion: "why is this only showing 3 repos?" Agents and scripts can't introspect ambient state reliably. Explicit is better than implicit. | `--workspace`/`--workstream`/`--repo` flags at call site. CWD-based resolution for workstream context. |
| **Token storage via keychain/1Password** | Managing `token_env` environment variables is tedious for multiple remotes. | Adds OS-specific dependencies (macOS Keychain, 1Password CLI), complicates testing, and introduces security surface area that's better handled by dedicated tools. | `token_env` references environment variables. Users manage secrets with their preferred secret manager (1Password, direnv, etc.). Post-v2.0 consideration. |
| **Cross-workstream dependency tracking** | "Workstream A depends on workstream B finishing first." | Introduces complex DAG scheduling into what should be a git orchestration tool. The right place for dependency tracking is the project management tool (Jira, Linear), not the git CLI. | Track dependencies in your project management tool. Workstreams are independent by design. |
| **`[[sync_pair]]` ref filtering beyond globs** | "I want to sync only refs newer than X" or "exclude tags". | Complex filtering logic adds cognitive overhead to an already-configurable system. Globs cover 95% of real routing needs. | Glob patterns on `refs` field. If you need finer control, use `git push` directly for edge cases. |
| **Pattern A promotion tracking** | "git-w should know that dev → test → prod is the promotion chain." | git-w manages repos and sync, not deployment pipelines. Promotion is a CI/CD concern with its own tooling (ArgoCD, Flux, etc.). Conflating the two creates a fragile coupling. | Promotion is a merge/PR on the upstream repo. git-w syncs; CI/CD deploys. |
| **Pattern B scope enforcement via pre-commit hook** | "If scope says `environments/dev/`, block commits touching other paths." | Advisory scope is sufficient for agents (they read AGENTS.md). Hard enforcement via pre-commit hooks would frustrate humans making legitimate cross-scope changes (README updates, shared modules). | Advisory `scope` field in `.gitw-stream`. Cross-modification warning in AGENTS.md. Agents respect it; humans override when needed. |
| **Per-worktree devcontainer support** | "Each worktree should have its own dev container config." | Massively increases complexity. Devcontainers are a per-repo concern, not a git-w concern. Worktrees inherit the repo's devcontainer config naturally. | Use standard devcontainer support from the repo itself. git-w manages worktree lifecycle, not dev environments. |
| **`git w workstream ship --open-prs` for non-GitHub remotes** | "I use Gitea for PRs too." | Gitea/Forgejo PR APIs differ enough from GitHub's that supporting them requires separate provider implementations with different field semantics. Ship to origin; open PRs manually on non-GitHub remotes. | Post-v2.0. `--push-all` works for any remote. PR opening is GitHub-only at launch. |

## Feature Dependencies

```
[v2 config schema + loader] (M1)
    └──requires──> nothing (foundation)

[Branch rule engine] (M2)
    └──requires──> [v2 config schema + loader] (M1)

[Multi-remote sync fan-out] (M3)
    └──requires──> [v2 config schema + loader] (M1)
    └──requires──> [Branch rule engine] (M2)

[Remote management] (M4)
    └──requires──> [Multi-remote sync fan-out] (M3)

[Unified status + branch checkout --from] (M5)
    └──requires──> [Multi-remote sync fan-out] (M3)
    └──requires──> [Remote management] (M4)

[Workstream push protection] (M6)
    └──requires──> [v2 config schema + loader] (M1)

[Workspace + workstream lifecycle] (M7)
    └──requires──> [Workstream push protection] (M6)

[Infra patterns A + B] (M8)
    └──requires──> [Workspace + workstream lifecycle] (M7)

[Agent context layer] (M9)
    └──requires──> [Workspace + workstream lifecycle] (M7)
    └──requires──> [Infra patterns A + B] (M8)

[Ship pipeline] (M10)
    └──requires──> [Agent context layer] (M9)

[Close and archival] (M11)
    └──requires──> [Ship pipeline] (M10)

[v1 to v2 migration] (M12)
    └──requires──> [v2 config schema + loader] (M1)
    (parallelizable with M2-M11)
```

### Dependency Notes

- **M3 requires M1 + M2:** Sync fan-out needs config types loaded and branch rules evaluatable before it can route refs through sync pairs.
- **M4 requires M3:** Remote management (add/list/status) needs the sync infrastructure to exist so `remote add` can wire sync pairs and offer initial mirror push.
- **M6 requires M1 only:** Push protection needs config types for `.gitw-stream` parsing but is independent of remote management. Could theoretically parallelize with M2-M5 but milestone ordering keeps it sequential.
- **M7 requires M6:** Workstream creation calls `reconcileHooks` on repos, so push protection must exist first.
- **M8 requires M7:** Infra patterns (env-groups, named worktrees) are expressed through workstream creation flags that need the workstream lifecycle to exist.
- **M9 requires M7 + M8:** Agent context generation needs to understand workstreams, env-groups, and Pattern B multi-worktree to produce complete AGENTS.md and JSON output.
- **M12 is parallelizable after M1:** Migration only needs config types; it reads v1 format and writes v2 format. Can develop concurrently with M2-M11.

## MVP Definition

### Launch With (v2.0)

Everything listed in Table Stakes and Differentiators ships together as v2.0. This is a major version upgrade, not an incremental release. The milestone structure (M1-M12) defines the build order.

- [x] v2 config schema + loader (M1) — everything depends on this
- [x] Branch rule engine (M2) — enables safe sync
- [x] Multi-remote sync fan-out (M3) — core v2 value
- [x] Remote management (M4) — users need to configure remotes
- [x] Unified status + branch checkout --from (M5) — replaces v1 commands
- [x] Workstream push protection (M6) — safety guarantee
- [x] Workspace + workstream lifecycle (M7) — organizational unit
- [x] Infra patterns A + B (M8) — differentiator
- [x] Agent context layer (M9) — differentiator
- [x] Ship pipeline (M10) — completes the workstream lifecycle
- [x] Close and archival (M11) — completes the workstream lifecycle
- [x] v1 to v2 migration (M12) — upgrade path

### Add After v2.0 (Post-Launch)

Features to add once v2 is validated and users have migrated.

- [ ] Token storage via keychain/1Password — triggered by user friction with `token_env`
- [ ] `--open-prs` for non-GitHub remotes — triggered by Gitea/Forgejo PR usage
- [ ] `git w context rebuild` heuristic repo descriptions from README parsing — polish
- [ ] Pattern A promotion tracking (dev→test→prod chain awareness) — triggered by infra user demand
- [ ] Pattern B cross-PR linking — triggered by Pattern B adoption
- [ ] Pattern B scope enforcement via pre-commit hook — triggered by agent mis-scoping incidents
- [ ] `[[sync_pair]]` ref filtering beyond globs — triggered by edge-case user needs
- [ ] Per-worktree devcontainer support — triggered by Codespaces/devcontainer adoption

### Future Consideration (v3+)

- [ ] Cross-workstream dependency tracking — only if project management tools can't handle it
- [ ] Additional `SpecFramework` implementations (speckit, openspec) — triggered by framework adoption
- [ ] Coordinated PR sequencing for Pattern B — requires deep GitHub API integration

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| v2 config schema + loader | HIGH | HIGH | P1 |
| Branch rule engine | HIGH | MEDIUM | P1 |
| Multi-remote sync fan-out | HIGH | HIGH | P1 |
| Remote management | HIGH | HIGH | P1 |
| Unified status | HIGH | MEDIUM | P1 |
| Workstream push protection | HIGH | HIGH | P1 |
| Workspace + workstream lifecycle | HIGH | HIGH | P1 |
| Ship pipeline | HIGH | HIGH | P1 |
| Close and archival | MEDIUM | MEDIUM | P1 |
| v1 to v2 migration | MEDIUM | MEDIUM | P1 |
| Infra Pattern A (branch-per-env) | HIGH | MEDIUM | P1 |
| Infra Pattern B (folder-per-env) | MEDIUM | HIGH | P1 |
| Agent context layer | HIGH | HIGH | P1 |
| `git w context rebuild` | MEDIUM | MEDIUM | P1 |
| Command surface reduction | MEDIUM | LOW | P1 |
| Token storage alternatives | MEDIUM | MEDIUM | P2 |
| Non-GitHub PR opening | LOW | MEDIUM | P2 |
| Pattern A promotion tracking | LOW | MEDIUM | P3 |
| Cross-workstream dependencies | LOW | HIGH | P3 |

**Priority key:**
- P1: Ships with v2.0 — all milestones are part of the release
- P2: Post-v2.0, add when user demand justifies
- P3: Future consideration, may never be needed

## Competitor Feature Analysis

| Feature | meta (Node.js) | gita (Python) | repo (Google) | git-w v2 |
|---------|----------------|---------------|---------------|----------|
| **Multi-repo registration** | `.meta` JSON file | `gita add` to CSV | `repo init` manifest XML | `.gitw` TOML with two-file merge |
| **Batch git operations** | `meta exec`, `meta git` | `gita super`, delegates | `repo forall -c` | `git w exec`, domain commands |
| **Parallel execution** | `--parallel` flag | Async subprocess | Parallel by default | `parallel.RunFanOut` goroutines |
| **Repo grouping** | Nested meta repos | `gita group` (flat) | `<project>` groups in manifest | Two-level workspace/workstream hierarchy |
| **Multi-remote sync** | Not supported | Not supported | Not supported | `[[sync_pair]]` fan-out with branch rules |
| **Push protection** | Not supported | Not supported | Not supported | Pre-push hook via `reconcileHooks` |
| **Worktree management** | Not supported | Not supported | Not supported | `git w workstream create` with auto-worktree |
| **Infra repo patterns** | Not supported | Not supported | Not supported | Pattern A (branch-per-env) + Pattern B (folder-per-env) |
| **Agent interop** | Not supported | Not supported | Not supported | Three-level AGENTS.md + `--json` context |
| **Machine portability** | `meta git clone` | `gita freeze`/`gita clone -f` | `repo init`/`repo sync` | `git w restore` re-materializes worktrees |
| **Planning state** | Not supported | Not supported | Not supported | `.planning/` at three levels, archived on close |
| **Plugin architecture** | Commander.js plugins | Custom JSON commands | Not extensible | Not extensible (by design; single binary) |
| **Config privacy** | Not supported | Not supported | Not supported | `.git/.gitw` for private overrides (tokens, personal remotes) |
| **Status display** | Per-repo output | Color-coded branch status with symbols | `repo status` | Unified status with env-groups, remote staleness, workstream context |

**Key takeaway:** The competitor landscape is feature-sparse. Most tools stop at "batch git commands across repos." git-w v1 already exceeds competitors. v2's table-stakes features (sync, push protection, workstreams) have zero competitors. The differentiators (infra patterns, agent interop) are in an entirely unoccupied space.

## Sources

- `meta` (mateodelnorte/meta) — GitHub README, 2.2k stars, last release 2021. Node.js. Plugin-based. [HIGH confidence — direct inspection]
- `gita` (nosarthur/gita) — GitHub README, 1.8k stars, active. Python. Group/context model. [HIGH confidence — direct inspection]
- `repo` (Google) — Android repo tool. XML manifest. Enterprise scale but rigid. [MEDIUM confidence — training data, not re-verified]
- `myrepos` — Perl-based, `.mrconfig`. Simple exec-across-repos. [LOW confidence — training data only]
- `mu-repo` — Python, `.mu_repo` config. Similar scope to gita. [LOW confidence — training data only]
- v2 spec documents — `.planning/v2/v2-*.md`. Authoritative for all git-w v2 features. [HIGH confidence — primary source]

---
*Feature research for: git-w v2 multi-repo management CLI*
*Researched: 2026-04-01*
