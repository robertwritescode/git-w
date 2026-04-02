# Project Research Summary

**Project:** git-w v2
**Domain:** Go CLI tool — multi-repo git orchestration (major version upgrade)
**Researched:** 2026-04-01
**Confidence:** HIGH

## Executive Summary

git-w v2 is a major upgrade to an existing Go CLI that manages multiple git repositories. The v2 scope is large: multi-remote sync with branch-level push rules, workstream-based work units with push protection via git hooks, API-driven remote provisioning (GitHub/Gitea), infrastructure repo patterns (branch-per-env and folder-per-env), and an agent interop layer for AI coding tools. The existing codebase (16 packages, cobra/testify/go-toml stack) is solid and well-tested. v2 adds 9 new packages and modifies 6 existing ones, but retains the same architectural spine: cobra command registration, TOML config, parallel subprocess execution via `os/exec`.

The recommended approach is to build bottom-up along the dependency chain: config types first (everything imports them), then the pure branch rule engine, then sync fan-out (the backbone operation), then remote management, hooks, and finally workstreams which integrate everything. This matches the spec's M1-M12 milestone structure. Four new dependencies are needed — Gitea SDK, go-github, doublestar for glob matching, and x/sync errgroup for bounded parallelism — all well-maintained with HIGH confidence versions confirmed on pkg.go.dev. The existing `parallel.RunFanOut` coexists with errgroup (different semantics: fire-and-forget vs. first-error cancellation).

The critical risks are: concurrent config file writes (v2 adds frequent write paths that v1 didn't have), worktree `.git` file breakage during migration directory moves, and TOML comment preservation at scale as the schema grows. All three must be addressed in M1 (foundation) before any feature work begins. The provider abstraction (GitHub vs. Gitea) is a design risk — design the interface from Gitea's simpler API first, then implement GitHub. Hook scripts must be tested in worktree contexts from day one, not just the main working tree.

## Key Findings

### Recommended Stack

v2 builds on the existing stack (cobra, go-toml/v2, testify, text/tabwriter, os/exec) without replacing anything. Four new runtime dependencies are needed, all with zero or minimal transitive deps.

**Core technologies:**
- **Gitea SDK** (`code.gitea.io/sdk/gitea` v0.24.1): First-party Gitea API provider — typed CRUD methods, handles pagination and auth
- **go-github** (`google/go-github/v84` v84.0.0): GitHub API provider — Google-maintained, tracks REST API changes within weeks
- **doublestar** (`bmatcuk/doublestar/v4` v4.10.0): Glob pattern matching for branch rules — supports `**` crossing `/`, zero deps, matches spec exactly
- **errgroup** (`golang.org/x/sync` v0.20.0): Bounded parallel fan-out — `SetLimit(n)` for worker pools with first-error cancellation

**Key decisions:**
- Do NOT use `go-git`. Continue with `os/exec` via `pkg/gitutil/` for all git operations. Consistency with v1, zero new deps.
- No TUI frameworks, no viper, no oauth2. Minimal dependency philosophy continues.
- `encoding/json` stdlib is sufficient for `--json` output. No third-party JSON libraries.
- Both `parallel.RunFanOut` (fire-and-forget) and `errgroup` (cancelling) coexist for different use cases.

### Expected Features

v2 is a single major version release, not incremental. All 12 milestones ship together. The feature set has zero competitors for its core value props (multi-remote sync, push protection, workstreams).

**Must have (table stakes):**
- v2 config schema + two-file merge loader (M1) — foundation for everything
- Multi-remote sync fan-out with branch rules (M2-M3) — core v2 value proposition
- Remote management with Gitea/GitHub provisioning (M4) — users need to configure remotes
- Unified status replacing v1's split info/status (M5) — UX fix
- Workstream push protection via pre-push hooks (M6) — safety guarantee
- Workspace + workstream lifecycle (M7) — organizational unit replacing v1 groups
- Ship pipeline: squash, push, open PRs (M10) — completes workstream lifecycle
- Close and archival (M11) — clean end-of-life for workstreams
- v1 to v2 migration (M12) — upgrade path for existing users

**Should have (differentiators):**
- Infra Pattern A: branch-per-env repo aliases with `track_branch` (M8)
- Infra Pattern B: folder-per-env multi-worktree with `scope` (M8)
- Agent interop: three-level AGENTS.md, `agent context --json`, SpecFramework (M9)
- Pre-ship squash with backup branches on personal remote (M10)
- Command surface reduction from 39 to 27 commands

**Anti-features (deliberately excluded):**
- Delete/remove commands for repos, workspaces, remotes — too dangerous, edit config manually
- TUI framework — breaks non-TTY contexts
- Ambient scope-setting — explicit flags and CWD-based resolution instead
- Cross-workstream dependency tracking — belongs in project management tools
- Pattern B scope enforcement via pre-commit hook — advisory scope is sufficient

### Architecture Approach

v2 extends v1's architecture rather than replacing it. The same cobra registration pattern, config loading flow, and parallel execution model persist. Nine new packages are added with clean dependency boundaries: `config` and `stream` own data, `branchrule` is a pure engine with zero deps, `hook` and `provider` are infrastructure, and `workstream` is the integration package that ties everything together. The critical architectural patterns are: config cascade resolution (metarepo < workstream < repo, innermost wins), self-contained `.gitw-stream` manifests (workstream state lives in its directory, not root config), self-healing hooks (reconcileHooks runs on every sync), and the SpecFramework interface for agent interop extensibility.

**Major components:**
1. **config + stream + state** — data layer owning `.gitw`, `.gitw-stream`, and `.git/git-w-state.json` respectively
2. **branchrule** — pure function rule engine (zero I/O, table-driven testable)
3. **hook** — pre-push hook installation, evaluation, reconciliation (self-healing on every sync)
4. **provider** — Gitea/GitHub/generic API providers behind `Provider` interface
5. **workstream** — largest new package; CRUD, ship, close, worktree management (replaces workgroup + worktree)
6. **agents** — SpecFramework interface, AGENTS.md/CONTEXT.md generators, framework registry
7. **migrate** — one-shot v1-to-v2 migration with resumable plan file

### Critical Pitfalls

1. **Concurrent config file writes** — v2 adds sync as a frequent writer. Implement `flock(2)` advisory locking on `.gitw.lock` in M1. Without this, simultaneous syncs silently lose data.
2. **Worktree `.git` file breakage during migration** — Directory moves break bidirectional worktree links. `git worktree repair` MUST be atomic with every directory move, never optional cleanup.
3. **TOML comment loss at scale** — v1's 525-line custom parser will break with v2's expanded schema. Decide comment strategy in M1: either accept comment loss with `.gitw.bak`, or replace the parser.
4. **Hook scripts breaking in worktree context** — `$GIT_DIR` differs in worktrees. Hooks must use `git rev-parse --git-common-dir`. Test from worktree contexts from day one.
5. **Branch rule evaluation order ambiguity** — TOML maps are unordered. Branch rules MUST use `[[array_of_tables]]` syntax, not maps. Order-dependent tests with 3+ rules required.
6. **Provider API differences leaking through abstraction** — Design `Provider` interface from Gitea (simpler API) first. Use `ProviderError` type, never expose raw HTTP status codes.
7. **Context cancellation ignored** — v1 already has `context.Background()` debt. v2's network operations make this a UX disaster. Enforce `cmd.Context()` in all new code from M1.

## Implications for Roadmap

Based on combined research, the build order follows the dependency chain discovered in architecture research, validated against pitfall timing requirements.

### Phase 1: Foundation (Config v2 + Stream + State)
**Rationale:** Every package imports config types. Nothing useful can happen without them. Pitfalls research mandates file locking and comment strategy decisions here.
**Delivers:** v2 config types, two-file field-level merge, `.gitw-stream` manifest format, `.git/git-w-state.json` format, file locking infrastructure, context.Context enforcement pattern.
**Features:** v2 config schema + loader (M1)
**Avoids:** Concurrent config writes (Pitfall 3), TOML comment loss (Pitfall 2), context cancellation debt (Pitfall 7)
**Stack:** go-toml/v2 (existing), encoding/json (stdlib)

### Phase 2: Branch Rule Engine
**Rationale:** Pure engine with zero dependencies. Must exist before sync can evaluate rules and before hooks can enforce push protection. Self-contained and independently testable.
**Delivers:** `EvaluateRule()` pure function, glob matching via doublestar, four action tiers (allow/block/warn/require-flag), criteria evaluation (untracked/explicit).
**Features:** Branch rule engine (M2)
**Avoids:** Branch rule evaluation order ambiguity (Pitfall 9) — uses ordered arrays from M1 schema
**Stack:** doublestar/v4

### Phase 3: Multi-Remote Sync Fan-Out
**Rationale:** Sync is the backbone operation and the core v2 value proposition. Validates cascade resolution, fan-out patterns, and state file writes that other milestones reuse.
**Delivers:** `[[sync_pair]]` routing, parallel fetch/push per-remote, branch rule evaluation during push, state file timestamp tracking, `reconcileHooks` as sync side effect.
**Features:** Multi-remote sync fan-out (M3)
**Avoids:** Two-file config merge surprises (Pitfall 10) — merge validation implemented
**Stack:** errgroup (new), state file (stdlib JSON)

### Phase 4: Remote Management + Providers
**Rationale:** Remote provisioning must exist before workstreams can meaningfully sync to secondary remotes. Provider interface design must be validated against both APIs before implementation.
**Delivers:** `git w remote add/list/status` wizard flow, Provider interface, Gitea + GitHub implementations, repo existence check, remote upsert, initial mirror push, `[[sync_pair]]` wiring.
**Features:** Remote management (M4)
**Avoids:** Provider API differences leaking (Pitfall 6) — interface designed from Gitea first
**Stack:** Gitea SDK, go-github/v84, stdlib net/http

### Phase 5: Unified Status + Branch Enhancements
**Rationale:** Depends on sync infrastructure (state file for staleness) and remote management (remote-fetched refs). Replaces confusing v1 split commands.
**Delivers:** Merged info+status command, env-group display, remote staleness from state file, `--json` output, `branch checkout --from <remote>`.
**Features:** Unified status (M5), `branch checkout --from` (M5)
**Stack:** text/tabwriter (existing), encoding/json (stdlib)

### Phase 6: Push Protection (Hooks)
**Rationale:** Workstream creation calls `reconcileHooks`, so hooks must exist first. This is the safety guarantee that makes workstreams viable.
**Delivers:** `reconcileHooks` function, pre-push hook script generation, `git-w hook pre-push` subcommand, worktree path resolution, remote whitelist enforcement.
**Features:** Workstream push protection (M6)
**Avoids:** Hook scripts breaking in worktree context (Pitfall 4) — integration tests from worktree contexts required

### Phase 7: Workspace + Workstream Lifecycle
**Rationale:** The integration milestone that ties config, stream, hook, agents together. Largest new-code phase. Must come after hooks and before ship/close.
**Delivers:** Workspace create/list, workstream create/list/status/switch, worktree management, `.gitw-stream` manifest writes, AGENTS.md generation, `.planning/` directory creation, `reconcileHooks` integration.
**Features:** Workspace lifecycle (M7), workstream lifecycle (M7)

### Phase 8: Infrastructure Patterns A + B
**Rationale:** Extends existing packages (repo, workstream) rather than creating new ones. Depends on workstream lifecycle existing.
**Delivers:** `track_branch`/`upstream` fields (Pattern A), `--env-group` expansion, named multi-worktree with `scope` (Pattern B), `--branch-map`, alias-aware mirror push.
**Features:** Infra Pattern A (M8), Infra Pattern B (M8)

### Phase 9: Agent Context Layer
**Rationale:** Fills in generator implementations that M7 calls with stubs. Depends on workstreams and infra patterns to produce complete context.
**Delivers:** Full AGENTS.md generators (meta-repo, workspace, workstream levels), CONTEXT.md generator, `git w agent context --json`, SpecFramework registry, GSDFramework implementation.
**Features:** Agent context layer (M9), `context rebuild` (M9)

### Phase 10: Ship Pipeline
**Rationale:** Requires everything else to be solid. Multi-step workflow touching hooks, providers, sync, and workstream state.
**Delivers:** Dirty check, squash pass with backup branches, push protection lift, scoped push to origin, PR opening (GitHub-only), status/timestamp updates.
**Features:** Ship pipeline (M10)

### Phase 11: Close and Archival
**Rationale:** Final workstream lifecycle stage. Depends on ship being complete.
**Delivers:** Worktree removal, hook cleanup, branch pruning prompts, `active/` to `archived/` move, `.planning/` preservation, context rebuild.
**Features:** Close and archival (M11)

### Phase 12: v1 to v2 Migration
**Rationale:** Parallelizable after Phase 1 since it only needs config types. Can develop concurrently with Phases 2-11.
**Delivers:** v1 config detection, `DetectV1`/`ReportPlan`/`ApplyPlan`, resumable migration with plan file, `git worktree repair`, collision/bare repo detection, `--dry-run`.
**Features:** v1 to v2 migration (M12)
**Avoids:** Worktree breakage on migration (Pitfall 1), partial failure leaves inconsistent state (Pitfall 5) — two-phase plan+execute with resume

### Phase Ordering Rationale

- **Critical path is P1 → P2 → P3 → P6 → P7.** Everything else can be parallelized within constraints.
- **Phase 12 (migration) is independent** after Phase 1 — can run in parallel with any other phase.
- **Phases 4-5 can parallelize with Phase 6** since they share no code dependencies beyond config.
- **Phase 7 is the integration bottleneck** — it pulls together config, stream, hook, and agents. Must be complete before infra patterns, agent context, and ship/close.
- **Pitfall timing drives Phase 1 scope** — file locking, comment strategy, and context enforcement must land in foundation, not be retrofitted.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1 (Foundation):** TOML comment preservation strategy needs a concrete decision — replace the parser vs. accept loss with backup. Research the go-toml/v2 AST API.
- **Phase 4 (Remote Management):** Provider interface design needs validation against both Gitea and GitHub API shapes before implementation. Verify Gitea SDK v0.24.1 endpoint coverage.
- **Phase 7 (Workstream Lifecycle):** Largest phase with most integration points. May need splitting into sub-phases during planning.
- **Phase 12 (Migration):** Five path cases and two abort conditions from spec. Resumable plan file design needs concrete schema definition.

Phases with standard patterns (skip research-phase):
- **Phase 2 (Branch Rules):** Pure function engine with table-driven tests. Well-understood pattern.
- **Phase 5 (Unified Status):** Straightforward command refactoring with existing patterns.
- **Phase 11 (Close):** Inverse of create with standard cleanup operations.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All 4 new dependencies confirmed on pkg.go.dev with recent versions. Version compatibility verified against Go 1.26. |
| Features | HIGH | All features sourced from authoritative v2 spec documents (.planning/v2/). Competitor analysis based on direct inspection of meta and gita. |
| Architecture | HIGH | v1 architecture fully documented. v2 architecture derived from v2 specs with clear package boundaries and dependency graph. No ambiguity in build order. |
| Pitfalls | HIGH | Critical pitfalls (worktree breakage, concurrent writes, TOML comments) sourced from existing v1 tech debt docs and git worktree documentation. Provider pitfalls based on direct API documentation comparison. |

**Overall confidence:** HIGH

### Gaps to Address

- **TOML comment strategy:** Research identified two options but didn't make the final call. Must be decided in Phase 1 planning — the choice affects whether the custom parser is replaced or extended.
- **Provider interface validation:** The interface is designed conceptually but hasn't been validated against actual Gitea SDK v0.24.1 method signatures. Phase 4 planning should verify endpoint coverage.
- **Phase 7 scope:** At 6+ major deliverables (workspace CRUD, workstream CRUD, worktree management, hook integration, AGENTS.md generation, planning directories), this phase may need splitting. Planner should evaluate during Phase 7 planning.
- **Concurrent git subprocess limits:** v1 uses `runtime.NumCPU()` workers. v2 adds network operations that may need different concurrency limits. Needs profiling during Phase 3 implementation.
- **`git w restore` reconciliation:** Manifest-vs-disk drift handling is specified conceptually but the exact diff algorithm for restore needs definition during Phase 7 or later planning.

## Sources

### Primary (HIGH confidence)
- v2 spec documents (`.planning/v2/v2-*.md`) — all features, architecture, milestones, migration, infra patterns, agent interop
- v1 codebase analysis (`.planning/codebase/*.md`) — architecture, conventions, structure, testing, stack, concerns
- pkg.go.dev — Gitea SDK v0.24.1, go-github v84.0.0, doublestar v4.10.0, x/sync v0.20.0 (all confirmed)
- Go 1.26 release notes — stdlib capabilities confirmed
- Git official documentation — `git-worktree(1)`, `githooks(5)` for hook and worktree semantics

### Secondary (MEDIUM confidence)
- `repo` (Google Android tool) — competitor analysis based on training data
- GitHub REST API / Gitea Swagger docs — provider API shape comparison

### Tertiary (LOW confidence)
- `myrepos`, `mu-repo` — competitor analysis from training data only, not re-verified

---
*Research completed: 2026-04-01*
*Ready for roadmap: yes*
