# Project Research Summary

**Project:** git-w v2 — Milestone 1: Config Schema + Loader
**Domain:** Go CLI tool — multi-repo git orchestration, v2 TOML config schema expansion
**Researched:** 2026-04-02
**Confidence:** HIGH

## Executive Summary

M1 is a pure infrastructure milestone: it ships the v2 config schema and a production-ready loader, but no user-visible commands change. Every downstream milestone (M2–M12) imports `pkg/config` types, so M1 is the critical foundation. The 11 phases within M1 must be executed in strict order because each adds a type family that later phases depend on — the type dependency chain is fixed, not a preference. All 12 CFG requirements ship together as one atomic milestone; any incomplete phase leaves the loader in an inconsistent state that will break M2+.

The recommended approach is surgical extension of the existing `pkg/config/` and `pkg/toml/` packages rather than package replacement or library swaps. `go-toml/v2` (bump to v2.3.0) remains the sole TOML library; comment-preservation is extended at the application layer via `UpdatePreservingComments`. Cycle detection, field-level merge, and cascade resolution are all implemented with stdlib primitives — no external graph, validation, or merge libraries are justified at this scale. The only new package stub required in M1 is `pkg/agents/registry.go` (a ~20-line stub to satisfy the `agentic_frameworks` registry check).

The two highest-risk tasks are CFG-07 (two-file field-level merge) and CFG-12 (comment-preserving round-trips for array-of-tables). Both require design decisions that must be made in Phase 1 — specifically, the choice between pointer types (`*bool`, `*string`) and value types for mergeable fields — because retrofitting after several phases are implemented cascades across all test fixtures. The existing `UpdatePreservingComments` has a documented silent-failure path that will silently strip user comments from Phase 4 onward if not addressed before `[[remote]]` blocks are introduced.

## Key Findings

### Recommended Stack

The M1 stack requires only one dependency change: bump `go-toml/v2` from v2.2.4 to v2.3.0. This is a patch-level upgrade with no API breaks and adds `unstable.RawMessage` for deferred raw TOML decoding. The `unstable.Parser{KeepComments: true}` sub-package (same module, no separate install) is the correct foundation for extending comment preservation to array-of-tables. All other M1 logic — cycle detection, merge functions, cascade resolution, validation — uses stdlib only.

**Core technologies:**
- `github.com/pelletier/go-toml/v2` v2.3.0: TOML parsing, marshaling, comment-preserving round-trips — only library needed; extend application-layer `UpdatePreservingComments` rather than switching
- `go-toml/v2/unstable.Parser{KeepComments: true}`: AST-level comment anchoring for array-of-tables — same module, no separate install, practically stable despite the name
- `encoding/json` (stdlib): JSON output for `--json` flags — sufficient for flat output structs
- Hand-written DFS (`map[string][]string` + three-color marking): cycle detection — ~20 lines, zero deps; gonum/graph unjustified at 3–5 node scale
- Hand-written `MergeX(base, override X) X` functions: field-level merge — per-field semantics (replace vs. nil vs. zero-value) cannot be expressed by generic merge libraries like `mergo`

**Libraries explicitly ruled out:** `BurntSushi/toml`, `naoina/toml`, `go-playground/validator`, `gonum/graph`, `mergo`, `viper`, `bubbletea/lipgloss`.

### Expected Features

All 12 CFG requirements are P1 — there is no optionality within M1. The features separate into two risk tiers.

**Must have — LOW implementation risk (table stakes for v2 schema):**
- CFG-01: `[[workspace]]` block parsing + `agentic_frameworks` field (registry validation against `pkg/agents` stub)
- CFG-02: `[[repo]]` v2 field additions (`track_branch`, `upstream`)
- CFG-03: `repos/<n>` path convention warning (non-fatal, emitted at command layer — not inside `Load`)
- CFG-06: `[[workstream]]` root block in config
- CFG-10: v1 `[[workgroup]]` detection — actionable error directing to `git w migrate`
- CFG-11: `agentic_frameworks` validation (bundled with CFG-01)

**Must have — MEDIUM/HIGH implementation risk (load-bearing for M3+):**
- CFG-04: `[[remote]]` + `[[remote.branch_rule]]` parsing (16 fields, nested sub-blocks, MEDIUM risk)
- CFG-05: `[[sync_pair]]` + DFS cycle detection at load time (MEDIUM risk)
- CFG-07: Two-file field-level merge (`.gitw` + `.git/.gitw`) with `private = true` enforcement (HIGH risk — pointer type design decision affects all other phases)
- CFG-08: `.gitw-stream` manifest parsing with uniqueness validation (MEDIUM risk)
- CFG-09: `[metarepo] default_remotes` three-level cascade resolution (pure function, MEDIUM risk)
- CFG-12: `UpdatePreservingComments` extended for array-of-tables round-trips (HIGH risk — silent-failure path already exists in the codebase)

**Defer to later milestones:**
- M2: `EvaluateRule` consuming `[[remote.branch_rule]]`
- M3: Sync fan-out consuming `[[sync_pair]]` and `[[remote]]`
- M7: Workspace/workstream lifecycle commands
- M12: `pkg/migrate` consuming v1 detection results from CFG-10

**Anti-features (don't build in M1):**
- Load-time migration logic in the config loader (migration belongs in `git w migrate`, M12)
- Lazy/partial config loading (full merge + full validation on every load)
- Schema version field in config (structural v1 detection via `[[workgroup]]` presence is more robust)
- Accumulated error lists (fail-fast with specific, named errors)

### Architecture Approach

M1 modifies or creates exactly 9 files, all confined to `pkg/config/`, `pkg/toml/`, and a new `pkg/agents/` stub. The v2 types live in the same `pkg/config` package as v1 types — a parallel `LoadV2Config(cmd)` entry point lets new v2-aware commands (M3+) opt in without changing existing command signatures. No `pkg/configv2/` package is created (would require touching all existing domain commands). The load-time data flow is: raw scan for v1 signals → unmarshal shared `.gitw` → validate per-file constraints → load optional `.git/.gitw` → field-level merge → post-merge validations (privacy enforcement, cycle detection) → return `*V2Config`.

**Major components created in M1:**
1. `pkg/config/config.go` (MODIFY): Add all v2 struct types — `V2Config`, `MetarepoConfig`, `WorkspaceBlock`, `RemoteConfig`, `BranchRule`, `SyncPair`, `WorkstreamBlock`; keep v1 types
2. `pkg/config/merge.go` (CREATE): Field-level merge functions per block type — `MergeRemote`, `MergeRepo`, `MergeWorkstream`, `MergeSyncPair`
3. `pkg/config/validate.go` (CREATE): `DetectCycles(pairs []SyncPair) error` (DFS) and `validatePathConvention` (warning, not error)
4. `pkg/config/detect.go` (CREATE): `DetectV1(raw []byte) error` — raw `[[workgroup]]` scan before unmarshal
5. `pkg/config/stream.go` (CREATE): `StreamConfig`, `WorktreeEntry` types, `LoadStream`, `DiscoverStream`, `validateStream` with uniqueness enforcement
6. `pkg/config/cascade.go` (CREATE): `ResolveDefaultRemotes(cfg, workstreamName, repoName) []string` — pure function, not called at load time
7. `pkg/config/loader.go` (MODIFY): Wire `LoadV2Config` entry point, two-file merge, v1 detection hook, path convention warning at call site
8. `pkg/toml/preserve.go` (MODIFY): Extend for `[[array table]]` AoT headers; fix silent-failure path
9. `pkg/agents/registry.go` (CREATE): Stub — `FrameworkFor(name string) error` + `knownFrameworks = []string{"gsd"}`

### Critical Pitfalls

1. **`UpdatePreservingComments` silently strips comments on AoT blocks** — The existing `applySmartUpdate` has a documented silent-return path that discards comment preservation failures. The current `findSectionBounds` regex matches `[section]` only, not `[[section]]`. Every new v2 block type will hit this path and silently strip all user comments on first write. **Fix:** Add AoT regex handling to `findSectionBounds` before Phase 4 introduces `[[remote]]`; fix the silent-return to emit a warning; add golden-file round-trip tests per block type.

2. **Two-file merge conflates absent fields with zero-value overrides** — Using `if override.Field != zero { result.Field = override.Field }` incorrectly treats an explicit `critical = false` in `.git/.gitw` as "not set," causing the base value to win silently. **Fix:** Use `*bool`, `*string`, `*[]string` for all fields where zero-value is a valid override; treat `nil` as absent, non-nil as explicit. This design decision must be made in Phase 1 — the struct field types cascade into every test fixture.

3. **`private = true` enforcement runs on merged config, losing provenance** — After merge, the loader cannot distinguish which remote came from `.gitw` vs. `.git/.gitw`. **Fix:** Validate each file before merging: run the `private = true` check on the raw-parsed public config before any merge step.

4. **Cycle detection catches only direct 2-node cycles** — Naive pair-matching misses indirect `A→B→C→A` cycles. At runtime, a missed sync cycle fans out in parallel and mirrors until storage is exhausted. **Fix:** Implement DFS with three-color marking (white/gray/black) from the start; include the full cycle path in the error message; test 3-node indirect cycles explicitly.

5. **`repos/<n>` warning fires inside `Load`, breaking all existing tests** — Existing test fixtures use `apps/frontend`-style paths. If the warning is emitted inside `Load`, every test that asserts exact stdout/stderr output fails. **Fix:** Keep `Load` side-effect-free; expose `WarnV1Paths(cfg, path, w io.Writer)` for command-layer callers only.

## Implications for Roadmap

Based on the research, M1's 11 phases map directly to the roadmap phases. The ordering is forced by type dependencies, not preference — a phase cannot be started until all types it requires are defined.

### Phase 1: Workspace block + agents stub + pointer-type decision
**Rationale:** First new block type; establishes the pointer-vs-value-type pattern that all subsequent struct definitions must follow. The `pkg/agents` stub must exist before Phase 1 completes (the `agentic_frameworks` validation calls `agents.FrameworkFor`). Making the pointer-type decision here avoids a cascade refactor later.
**Delivers:** `WorkspaceBlock`, `MetarepoConfig` with `AgenticFrameworks`, `pkg/agents/registry.go` stub; pointer-type pattern established
**Addresses:** CFG-01, CFG-11
**Avoids:** Pitfall 2 (zero-value merge) — pointer types established before merge functions are written

### Phase 2: `[[repo]]` v2 field additions
**Rationale:** Extends existing `RepoConfig` type with `TrackBranch` and `Upstream`. No new block type; depends on Phase 1 only to ensure `RepoConfig` exists in its final location.
**Delivers:** `RepoConfig.TrackBranch`, `RepoConfig.Upstream`
**Addresses:** CFG-02

### Phase 3: `repos/<n>` path convention warning
**Rationale:** Adds `validatePathConvention` as a separate function (NOT inside `Load`); creates `WarnV1Paths` for command-layer callers.
**Delivers:** Non-fatal path convention check; `pkg/config/validate.go` shell created
**Addresses:** CFG-03
**Avoids:** Pitfall 5 (warning-inside-Load breaking existing tests)

### Phase 4: `[[remote]]` + `[[remote.branch_rule]]` parsing
**Rationale:** First array-of-tables (AoT) block type. Before writing `RemoteConfig`, `UpdatePreservingComments` must be confirmed to handle AoT headers, or the silent-comment-loss pitfall activates immediately. The AoT regex fix in `pkg/toml/preserve.go` must land in this phase or the limitation must be explicitly documented with a warning emitted.
**Delivers:** `RemoteConfig`, `BranchRule` types; AoT regex fix or documented limitation
**Addresses:** CFG-04
**Avoids:** Pitfall 1 (silent comment strip) — AoT regex fix lands here

### Phase 5: `[[sync_pair]]` + cycle detection
**Rationale:** `SyncPair` requires `RemoteConfig.Name` to exist for graph node validation. Cycle detection must use full DFS from day one — not patchable from pair-check later.
**Delivers:** `SyncPair` type, `DetectCycles()` in `validate.go`, three-color DFS with actionable error messages
**Addresses:** CFG-05
**Avoids:** Pitfall 4 (missed indirect cycles)

### Phase 6: `[[workstream]]` root block
**Rationale:** Final root-config block type. All four block families (workspace, repo, remote, sync_pair, workstream) must exist before the merge functions in Phase 7 can be written.
**Delivers:** `WorkstreamBlock` type
**Addresses:** CFG-06

### Phase 7: Two-file field-level merge
**Rationale:** Integration point for all block types. All types must be stable. `private = true` enforcement must run on the pre-merge public config, not the merged result. Composite `(from, to)` key required for `SyncPair` merge.
**Delivers:** `pkg/config/merge.go` with `MergeRemote`, `MergeRepo`, `MergeWorkstream`, `MergeSyncPair`; `LoadV2Config` wired in `loader.go`; `private = true` pre-merge check
**Addresses:** CFG-07
**Avoids:** Pitfall 2 (zero-value merge — pointer types already established), Pitfall 3 (private enforcement on wrong stage), Pitfall 10 (partial-key sync pair merge)

### Phase 8: `.gitw-stream` manifest
**Rationale:** Separate file format, independent of root config. V2Config is stable after Phase 7; stream types can be finalized against it. Uniqueness validation must be in `LoadStream`, not deferred to command layer.
**Delivers:** `pkg/config/stream.go` with `StreamConfig`, `WorktreeEntry`, `LoadStream`, `DiscoverStream`, `validateStream`; name/path uniqueness enforcement; atomic write (temp + rename)
**Addresses:** CFG-08
**Avoids:** Pitfall 8 (missing uniqueness validation)

### Phase 9: Default remotes cascade
**Rationale:** Pure function; requires `WorkstreamBlock.Remotes` (Phase 6) and the merged `V2Config` (Phase 7) as its inputs. Must return `[]string{}` (not nil) when all three cascade levels are absent.
**Delivers:** `pkg/config/cascade.go` with `ResolveDefaultRemotes(cfg, wsName, repoName) []string`; exhaustive 8-combination table-driven tests
**Addresses:** CFG-09
**Avoids:** Pitfall 7 (cascade nil-vs-empty, wrong winner semantics)

### Phase 10: v1 `[[workgroup]]` detection
**Rationale:** Placed last among loader-wiring phases to avoid repeated adjustment of the load entry point. Detection runs on the primary `.gitw` file only — not `.gitw.local` or `.git/.gitw`. Must detect both `[[workgroup]]` (AoT) and `[workgroup.name]` (map-of-tables) v1 syntaxes.
**Delivers:** `pkg/config/detect.go` with `DetectV1(raw []byte) error`; wired into `LoadV2Config` before unmarshal
**Addresses:** CFG-10
**Avoids:** Pitfall 5 (detection firing on wrong files, partial-migration blocking)

### Phase 11: `UpdatePreservingComments` full AoT round-trip coverage
**Rationale:** Test-coverage phase that can only be "done" when all block types exist and load correctly. All new `[[double-bracket]]` block types need golden-file round-trip tests. Fix the silent-fallback error path (emit warning, never silently discard).
**Delivers:** Extended `pkg/toml/preserve.go` with AoT index-namespaced comment anchors; golden-file fixtures for all new block types; silent-fallback warning fix
**Addresses:** CFG-12
**Avoids:** Pitfall 1 (complete resolution), Pitfall 2 (AoT anchor collision in multiple-instance blocks)

### Phase Ordering Rationale

- **Phases 1–6 are strictly sequential** because each adds a type family that later phases reference. You cannot write `MergeRemote` before `RemoteConfig` exists; you cannot write `DetectCycles` before `SyncPair` exists.
- **Phase 7 is the integration milestone** — the first phase that spans the entire type space simultaneously. All prior phases are additive; Phase 7 is the first cross-cutting concern.
- **Phase 8 (stream) is independent** of the root config load pipeline and could start after Phase 7 finishes; placing it here keeps the milestone sequential and ensures stream types are finalized against a stable `V2Config`.
- **Phase 9 (cascade) is a pure function** with no load-time wiring — it cannot be called until Phase 7 provides a merged config as input.
- **Phase 10 (v1 detection) modifies `loader.go` at the top of the load pipeline** — placing it last among loader-wiring phases means `loader.go` gets one final structural change rather than repeated adjustments.
- **Phase 11 (round-trip)** is a test-completeness phase; it verifies correctness across all block types and cannot be "done" before all types exist.

### Research Flags

Phases that benefit from deeper investigation before task creation:

- **Phase 7 (Two-file merge):** The pointer-type decision cascades into every struct field and every test fixture. Requires explicit pre-phase alignment: which fields use `*bool`/`*string` vs. concrete types, and what the exact merge semantics are for `refs = []` (empty slice as intentional override vs. absent). See `.planning/v2/v2-schema.md` §merge semantics.
- **Phase 11 (UpdatePreservingComments):** The AoT index-namespacing approach for comment anchors is a non-trivial redesign of `extractCommentAnchors` and `injectSectionComments`. Recommend a focused spike on the algorithm before task creation — the "best-effort AoT preservation" fallback (preserve region between `[[block]]` headers rather than individual keys) may be more practical.

Phases with well-documented standard patterns (skip deeper research):

- **Phases 1–3:** Straightforward struct additions following existing `pkg/config/` patterns; direct codebase inspection is sufficient.
- **Phase 4:** `[[remote]]` struct definition is large but mechanical; the only research-worthy item is the AoT regex fix, which is documented.
- **Phase 5:** Three-color DFS is a standard algorithm; ready-to-use implementation is documented in STACK.md.
- **Phases 8–10:** Requirements fully specified in `.planning/v2/v2-schema.md`; no external pattern research needed.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All library versions confirmed on pkg.go.dev; go-toml/v2 maintainer intent confirmed from GitHub discussion #506; existing codebase read directly |
| Features | HIGH | Primary sources: `.planning/REQUIREMENTS.md` (CFG-01–CFG-12), `.planning/v2/v2-schema.md`, direct codebase inspection of `loader.go`, `config.go`, `preserve.go` |
| Architecture | HIGH | Authoritative v2 spec docs in `.planning/v2/`; v1 codebase structure confirmed via direct read; package map derived from actual file inspection |
| Pitfalls | HIGH | All pitfalls identified from direct code inspection (`preserve.go` line 77-83 silent return, `findSectionBounds` regex, `WorkspaceMeta` pointer-field pattern); no inference from external sources |

**Overall confidence:** HIGH

### Gaps to Address

- **`[[sync_pair]]` additive semantics (Pitfall 10):** The spec is silent on whether `.git/.gitw` can add new `[[sync_pair]]` entries not present in `.gitw`, or only override existing ones. This must be decided before Phase 7 task creation. Recommend: additive (private file can add new pairs); document in code comments.
- **Comment anchor collision strategy for multiple-instance AoT blocks:** The spec does not prescribe whether Phase 11 must perfectly preserve per-key comments inside each `[[remote]]` block, or whether region-level preservation (comments above the entire `[[remote]]` block) is sufficient. This trade-off affects implementation complexity significantly. Recommend: define the acceptable fidelity level before Phase 11 task creation.
- **`diskConfig` coverage across all phases:** Every new v2 field must appear in `prepareDiskConfig` output to survive a `Save` + `Load` round-trip. This structural concern applies to every phase (1–10); each phase's definition of done should include a round-trip regression test.

## Sources

### Primary (HIGH confidence)
- `.planning/REQUIREMENTS.md` — CFG-01 through CFG-12 requirements, phase assignments, acceptance criteria
- `.planning/v2/v2-schema.md` — authoritative v2 config schema, merge semantics, cascade rules
- `.planning/v2/v2-milestones.md` — M1 scope, phase dependencies, build order
- `.planning/v2/v2-remote-management.md` — sync pair semantics, remote field definitions
- `.planning/codebase/ARCHITECTURE.md` — v1 architecture, package responsibilities
- `.planning/codebase/CONCERNS.md` — documented silent-failure path in `UpdatePreservingComments`
- `pkg/toml/preserve.go` — direct inspection: `applySmartUpdate` line 77-83, `findSectionBounds` regex, `anchorIdentity` flat-key logic
- `pkg/config/loader.go` — direct inspection: `loadMainConfig`, `mergeLocalConfig`, `prepareDiskConfig`, `ensureWorkspaceMaps`
- `pkg/config/config.go` — direct inspection: `WorkspaceConfig`, `WorkspaceMeta` pointer-field pattern
- `github.com/pelletier/go-toml/discussions/506` — maintainer confirmed document editing out of scope; `unstable.Parser{KeepComments: true}` is the correct hook

### Secondary (HIGH confidence — version confirmation)
- `pkg.go.dev/github.com/pelletier/go-toml/v2` — v2.3.0 published 2026-03-24 confirmed
- `pkg.go.dev/github.com/pelletier/go-toml/v2/unstable` — `Kind.Comment`, `Parser.KeepComments` API confirmed
- `pkg.go.dev/code.gitea.io/sdk/gitea` — v0.24.1 confirmed
- `pkg.go.dev/github.com/google/go-github` — v84.0.0 confirmed
- `pkg.go.dev/github.com/bmatcuk/doublestar/v4` — v4.10.0 confirmed
- `pkg.go.dev/golang.org/x/sync/errgroup` — v0.20.0 confirmed

---
*Research completed: 2026-04-02*
*Ready for roadmap: yes*
