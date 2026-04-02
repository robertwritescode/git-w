# Feature Research

**Domain:** v2 config schema and loader — multi-file TOML with merge semantics, cycle detection, cascade/inheritance, comment-preserving round-trips, and v1/v2 migration detection
**Researched:** 2026-04-02
**Confidence:** HIGH (primary sources: existing codebase, v2 spec docs in `.planning/v2/`)

---

## M1 Scope Anchor

This file is scoped to **Milestone 1: v2 config schema + loader** (CFG-01 through CFG-12). It answers:
- Which M1 features are table stakes (must work, foundation-load-bearing)?
- Which are differentiators (novel config patterns not found elsewhere)?
- Which are anti-features (patterns that look helpful but add cost with no benefit)?
- What are the complexity ratings and dependencies between M1's own phases?

---

## Feature Landscape

### Table Stakes (Foundation Load-Bearing)

These features are load-blocking: if any of them don't work, M1 is incomplete and M2–M12 cannot proceed.

| Feature | Why Required | Complexity | M1 Phase |
|---------|-------------|------------|----------|
| **`[[workspace]]` block parsing** (CFG-01) | First new block type. `[[workspace]]` is how workspaces are defined. Every downstream feature that filters by workspace depends on this being loaded correctly. | LOW | Phase 1 (#36) |
| **`[metarepo]` + `agentic_frameworks` field** (CFG-11) | Validated alongside CFG-01 in the same phase. Load-time registry check against `pkg/agents`. Missing defaults to `["gsd"]`. Error on unknown value. | LOW | Phase 1 (#36) |
| **`[[repo]]` v2 additions** (CFG-02) | `track_branch` and `upstream` fields on existing `[[repo]]` blocks. Used by M3 sync (pull target) and M8 infra patterns (env-group expansion). | LOW | Phase 2 (#37) |
| **`repos/<n>` path convention enforcement** (CFG-03) | Every v2 repo path must match `repos/<n>`. Load-time warning for v1 paths. Blocks incorrect usage early; prevents silent breakage when M7+ workstream commands assume path convention. | LOW | Phase 3 (#38) |
| **`[[remote]]` block parsing** (CFG-04) | Core block for M3 sync fan-out, M4 remote management, M6 push protection. Without this, no multi-remote feature can proceed. Includes all fields: `name`, `kind`, `url`, `user`, `token_env`, `org`, `repo_prefix`, `repo_suffix`, `direction`, `push_mode`, `fetch_mode`, `use_ssh`, `ssh_host`, `critical`, `private`. | MEDIUM | Phase 4 (#39) |
| **`[[remote.branch_rule]]` parsing** (CFG-04) | Inline sub-blocks on `[[remote]]`. Parsed together in Phase 4. Fields: `pattern`, `untracked`, `explicit`, `action`, `reason`, `flag`. | MEDIUM | Phase 4 (#39) |
| **`[[sync_pair]]` with cycle detection** (CFG-05) | Explicit ref routing between remotes. Cycle detection at load time prevents infinite sync loops. Without this, M3 fan-out executor has no routing config. | MEDIUM | Phase 5 (#40) |
| **`[[workstream]]` root config block** (CFG-06) | Lightweight remote override for workstream worktrees. Lives only in `.git/.gitw`. Enables per-workstream remote isolation during active work. | LOW | Phase 6 (#41) |
| **Two-file merge (`.gitw` + `.git/.gitw`)** (CFG-07) | The privacy model for the entire config system. Shared config committed; private config (tokens, personal remotes) machine-local. Without this merge, tokens cannot exist in any sensible place. | HIGH | Phase 7 (#42) |
| **`.gitw-stream` manifest parsing** (CFG-08) | Self-contained per-workstream config. Machine-readable source of truth for ship, close, and agent context. Fields: `name`, `path`, `scope` on `[[worktree]]`. Uniqueness validation. | MEDIUM | Phase 8 (#43) |
| **`[metarepo] default_remotes` cascade** (CFG-09) | Three-level inheritance: metarepo → workstream → repo, innermost wins. Without this, every repo needs explicit remote config and the default-remotes pattern is useless. | MEDIUM | Phase 9 (#44) |
| **v1 `[[workgroup]]` detection** (CFG-10) | Detection only — no migration logic. Actionable error message directing user to `git w migrate`. Guards v2 users from accidentally running on a v1 config, which would silently misinterpret blocks. | LOW | Phase 10 (#45) |
| **`UpdatePreservingComments` round-trip for all v2 fields** (CFG-12) | Every write operation (`Save`, `SaveLocal`, `.gitw-stream` writes) must preserve user comments. Without this, a single `git w remote add` strips all comments from the user's carefully annotated config — a show-stopping regression. | HIGH | Phase 11 (#46) |

### Differentiators (Novel Config Patterns)

Config patterns not found in competing tools or in the v1 codebase. These are the design decisions that make v2's config system meaningfully better.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Field-level merge with `private = true` enforcement** | Most multi-file config systems do file-level override (last file wins all). Field-level merge allows `.git/.gitw` to add a `token_env` to an `origin` remote defined in `.gitw`, without having to redeclare all fields. Privacy enforcement prevents tokens landing in committed files — a real secret-exposure risk. | MEDIUM | Enforcement is a load-time structural check: if `private = true` appears in `.gitw`, reject with a named error. If in `.git/.gitw`, allow. No runtime policy evaluation needed. |
| **Cycle detection in `[[sync_pair]]` graph** | Explicit A→B, B→C, C→A sync loops are user errors, not valid config. Detecting them at load time (not at sync execution time) means failure is immediate and attributable, not a confusing eventual-loop. Algorithm: topological sort over `(from, to)` pairs; non-zero cycle = load error. | MEDIUM | Keep detection simple: build directed graph, run DFS, first back-edge = error. The graph is small (typically 1-3 pairs). No need for Tarjan's or Kahn's for this scale. |
| **`[metarepo] default_remotes` cascade with innermost-wins** | Cascade patterns are common in config systems (git's own `includeIf`, AWS profiles, etc.). But most are file-level or section-level. This is a named-remote-list cascade: workspace default → workstream override → repo override. Innermost wins. Empty list at any level means "no defaults from this level"; it is not a wildcard. | MEDIUM | Implementation: resolve at call site (not at load time) — return effective remote list given a repo name and optional workstream name. Tests: metarepo-only, workstream-override, repo-override, empty-at-each-level. |
| **`.gitw-stream` self-contained manifests** | Workstream config travels with the workstream directory, not in root `.gitw`. This means workstream files are committed to the meta-repo and version-controlled separately from root config. They can be read without loading root config (agent context scenario). | MEDIUM | Two uniqueness invariants: `name` unique within workstream, `path` unique within workstream. Pattern B (same repo twice) requires `name` on both entries. Error message must suggest `--worktree` flag. |
| **`agentic_frameworks` registry validation** | Config fields validated against a live Go registry (`pkg/agents.FrameworkFor`), not a hardcoded string list. New frameworks can be registered in code without changing the config parser. Missing field defaults to `["gsd"]` — backward compatible default. | LOW | The config loader does NOT import framework behavior — it only calls the registry lookup. Dependency direction is: config → agents registry (names only), not agents → config. |

### Anti-Features (Commonly Requested, Often Problematic)

Config patterns that seem helpful but would add cost without benefit, or create problems downstream.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Load-time migration logic in the config loader** | "Detect v1 `[[workgroup]]` at load time and auto-convert to v2 format." | The config loader must stay pure: parse, validate, merge. Migration is a separate command (`git w migrate`) with pre-flight checks, filesystem moves, and user confirmation. Embedding migration in the loader makes the loader stateful and creates silent config mutations. | Detection only in CFG-10. Error message names the problem and provides the exact command to run. M12 handles migration. |
| **Lazy merge (merge only what's needed)** | "Only merge the sections the calling command needs." | Partial loading creates subtle bugs: a command that only loads `[[repo]]` blocks won't detect `private = true` violations in `[[remote]]` blocks. Config must be fully merged and validated on every load. | Load fully, validate fully. Config files are small; the overhead is negligible. |
| **Comment-preserving via TOML AST library** | "Use a TOML library with native comment support (e.g., `github.com/BurntSushi/toml` with AST mode) for perfect round-trips." | `go-toml/v2` is already in use and working. Replacing it for comment preservation would require rewriting all existing TOML usage. The existing `UpdatePreservingComments` function in `pkg/toml/preserve.go` handles the round-trip use case with comment anchoring already. | Extend `UpdatePreservingComments` to handle array-of-tables (`[[remote]]`, `[[sync_pair]]`, `[[workspace]]`, `[[worktree]]`) which it does not currently do. This is scoped work, not a library replacement. |
| **Schema versioning field in config** | "Add `schema_version = 2` to `.gitw` so the loader knows which parser to use." | v1 detection is structural (presence of `[[workgroup]]` blocks) not declarative. A schema version field creates a coordination problem: which version wrote the file? What if the field is absent? The structural detection approach is more robust and requires no user action. | v1 detection via `[[workgroup]]` block presence (CFG-10). v2 is the default if no v1 signals are present. |
| **Per-field validation errors accumulated into a list** | "Return all config errors at once instead of fail-fast." | Accumulating errors is harder to implement and can create confusing error messages that reference fields loaded after a broken dependency (e.g., validate `[[sync_pair]]` references to non-existent remotes). Fail-fast on first error is simpler and the errors are actionable. | Fail-fast with specific, named error messages (e.g., "remote 'personal' referenced in [[sync_pair]] but not defined"). |
| **Optional fields with `omitempty` everywhere** | "All new fields should be optional with zero-value defaults." | Omitempty is correct for truly optional fields. But several v2 fields have semantic meaning at zero-value that differs from absent: `private = false` (explicit opt-in to public placement) vs absent (default behavior). Using `omitempty` on these fields would make it impossible to distinguish "explicitly set to false" from "not set". | Use pointer fields (`*bool`) for boolean fields with meaningful zero-vs-absent distinction. Use `omitempty` only where zero-value and absent are semantically equivalent. This is consistent with the existing `WorkspaceMeta` struct pattern. |

---

## Feature Dependencies Within M1

```
[CFG-01: [[workspace]] + agentic_frameworks] (Phase 1)
    └── no M1 dependencies (first new block)

[CFG-02: [[repo]] v2 fields] (Phase 2)
    └── no M1 dependencies (extends existing block)

[CFG-03: repos/<n> path enforcement] (Phase 3)
    └── no M1 dependencies (path validation only)

[CFG-04: [[remote]] + [[remote.branch_rule]]] (Phase 4)
    └── no M1 dependencies (new block family)

[CFG-05: [[sync_pair]] + cycle detection] (Phase 5)
    └──requires──> [CFG-04: [[remote]]] (Phase 4)
       (cycle detection validates remote names in sync_pairs)

[CFG-06: [[workstream]] root block] (Phase 6)
    └──requires──> [CFG-04: [[remote]]] (Phase 4)
       (workstream.remotes is a list of remote names)

[CFG-07: two-file merge] (Phase 7)
    └──requires──> [CFG-04: [[remote]]] (Phase 4)
       (private = true enforcement requires [[remote]] to exist)
    └──requires──> [CFG-05: [[sync_pair]]] (Phase 5)
       (merge semantics include sync_pair by (from, to) key)
    └──requires──> [CFG-06: [[workstream]]] (Phase 6)
       (workstream blocks are merged by name)

[CFG-08: .gitw-stream manifest] (Phase 8)
    └── no M1 dependencies (separate file, separate loader function)

[CFG-09: default_remotes cascade] (Phase 9)
    └──requires──> [CFG-04: [[remote]]] (Phase 4)
    └──requires──> [CFG-06: [[workstream]] root block] (Phase 6)
    └──requires──> [CFG-07: two-file merge] (Phase 7)
       (cascade resolution requires merged config to be available)

[CFG-10: v1 [[workgroup]] detection] (Phase 10)
    └── no M1 dependencies (structural detection only, no new types needed)

[CFG-12: UpdatePreservingComments round-trips] (Phase 11)
    └──requires──> [ALL CFG-01 through CFG-10]
       (must round-trip every new field; needs all types defined first)
```

### Dependency Notes

- **CFG-05 requires CFG-04:** Cycle detection validates `from` and `to` names against loaded `[[remote]]` blocks. Can't detect invalid remote references without `[[remote]]` types existing.
- **CFG-07 requires CFG-04 through CFG-06:** Two-file merge must handle all block types. Each new block type (remote, sync_pair, workstream) must be defined before merge semantics can be implemented for it.
- **CFG-09 requires CFG-07:** Cascade resolution operates on the merged config result. Implementing cascade on a single-file config first would require refactoring after CFG-07.
- **CFG-08 is independent within M1:** `.gitw-stream` is a separate file with its own loader function. No dependency on `.gitw` parsing. Can be implemented in parallel with or after any CFG phase.
- **CFG-12 is always last:** Round-trip tests require all new fields to be defined. Running it last is correct. Any field added after CFG-12 must extend the round-trip test suite.
- **CFG-10 has no M1 dependencies:** v1 detection is purely structural — scan for `[[workgroup]]` key presence before any other parsing. Can technically be implemented first, but is lowest-risk so scheduling last (before CFG-12) is fine.

---

## Complexity Notes Per Feature

### HIGH complexity

**CFG-07 (Two-file merge)** — The merge function must handle:
1. Scalar field merge: override value wins if non-zero, else base value used.
2. Array-of-tables merge by primary key: `[[remote]]` by `name`, `[[sync_pair]]` by `(from, to)`, `[[repo]]` by `name`, `[[workspace]]` by `name`, `[[workstream]]` by `name`.
3. `private = true` enforcement: reject if found in `.gitw` (not `.git/.gitw`).
4. The `.git/.gitw` file may not exist — that is the common case (no private overrides). Merge must handle absent second file gracefully.
5. Test surface is large: every field on every block type, including zero-value fields, partial overrides, and array field behavior.

**CFG-12 (UpdatePreservingComments round-trips)** — The existing `UpdatePreservingComments` in `pkg/toml/preserve.go` handles section-level updates but its `findSectionBounds` uses regex matching `\[sectionname\]` which does not handle TOML array-of-tables syntax (`[[remote]]`, `[[workspace]]`, etc.). v2 requires writing these array-of-tables back to disk without stripping comments. The implementation must either:
- Extend `preserve.go` to handle `[[...]]` headers, or
- Write a separate array-of-tables update path.
This is the highest-risk implementation task in M1. The existing anchor-based comment injection approach may need redesign for array-of-tables where multiple entries exist under the same key.

### MEDIUM complexity

**CFG-04 (`[[remote]]` + `[[remote.branch_rule]]`)** — Large struct with many fields (16 on `[[remote]]`, 6 on `[[remote.branch_rule]]`). Validation rules: `kind` must be one of `gitea|forgejo|github|generic`; `direction` must be `push|fetch|both`; `push_mode` must be `mirror|branch`; `fetch_mode` must be `all|tracked`. Empty `url` and `token_env` are valid (fields may come from `.git/.gitw` overlay). The sub-block relationship (`[[remote.branch_rule]]` inside `[[remote]]`) requires understanding how go-toml/v2 serializes nested array-of-tables.

**CFG-05 (`[[sync_pair]]` + cycle detection)** — Cycle detection is straightforward DFS. The complexity is in the graph structure: pairs reference remote names, not remote structs. Name resolution must happen before cycle detection, which means CFG-05 is only fully testable after CFG-04 defines what valid remote names look like.

**CFG-08 (`.gitw-stream` manifest)** — Two uniqueness constraints (`name` and `path` unique within workstream) interact with Pattern B defaulting rules (`name` defaults to repo name, `path` defaults to `name`). The error message for duplicate repo without unique `name` must suggest `--worktree`. Atomic write (temp + rename) required since the manifest is committed to the meta-repo.

**CFG-09 (cascade resolution)** — The cascade is three-level with "innermost wins" semantics. An empty list (`remotes = []`) at any level is a valid override (means "no default remotes from this level"), not a pass-through. The function signature must accept repo name, optional workstream name, and the merged config, and return the effective remote list. Resolution must be a pure function for testability.

### LOW complexity

**CFG-01**, **CFG-02**, **CFG-03**, **CFG-06**, **CFG-10**, **CFG-11** — Straightforward struct additions, field validation, and detection logic. These are incremental extensions to the existing config loader pattern.

---

## Existing Config Loader — Integration Points

The v2 config types are a **replacement, not an extension** of the current `WorkspaceConfig` struct. Key observations from the existing codebase:

1. **`WorkspaceConfig` uses maps (`map[string]RepoConfig`) for repos and groups.** v2 switches to arrays-of-tables (`[[repo]]`). This is a TOML encoding change — `go-toml/v2` handles both, but the Go struct must use slice fields (`[]RepoConfig`) not map fields. The `name` field becomes the primary key for merge operations, not the map key.

2. **`loader.go` already has `UpdatePreservingComments` via `saveWithCommentPreservation`.** The plumbing exists. CFG-12 extends coverage, it does not introduce a new mechanism.

3. **`preserve.go` `findSectionBounds` uses `\[sectionname\]` regex.** This matches singular TOML tables only. Array-of-tables headers are `[[sectionname]]`. The regex must be updated to handle both forms, and the update logic must handle multiple array entries under the same key.

4. **`loader.go` `Load` function signature currently takes `configPath string` only.** CFG-07 requires it to also read `.git/.gitw`. The function signature or the internal call chain needs updating. A clean approach: `Load(configPath)` continues to work for single-file scenarios; `LoadMerged(configPath)` reads and merges both files.

5. **`localDiskConfig` pattern shows the existing two-file approach for `.gitw.local`.** CFG-07's two-file merge is a generalisation of this pattern: same atomic-read-then-merge structure, but merging at field level rather than section level.

6. **`validateRepoPaths` in the loader validates path format.** CFG-03 extends this with a pattern check (`repos/<n>`) and a warning (not error) for non-conforming v1 paths. The existing validation returns an error; CFG-03's warning is softer — it should log the warning and continue loading, not fail.

---

## MVP Definition

### For M1 Launch (all phases must be complete)

All 12 CFG requirements (CFG-01 through CFG-12) ship together as M1. There is no partial M1 — any incomplete phase leaves the config loader in an inconsistent state that breaks downstream milestones.

- [x] CFG-01: `[[workspace]]` block + `agentic_frameworks` field (Phase 1)
- [x] CFG-02: `[[repo]]` v2 fields (`track_branch`, `upstream`) (Phase 2)
- [x] CFG-03: `repos/<n>` path enforcement (Phase 3)
- [x] CFG-04: `[[remote]]` + `[[remote.branch_rule]]` (Phase 4)
- [x] CFG-05: `[[sync_pair]]` + cycle detection (Phase 5)
- [x] CFG-06: `[[workstream]]` root block (Phase 6)
- [x] CFG-07: Two-file merge (Phase 7)
- [x] CFG-08: `.gitw-stream` manifest (Phase 8)
- [x] CFG-09: `default_remotes` cascade (Phase 9)
- [x] CFG-10: v1 `[[workgroup]]` detection (Phase 10)
- [x] CFG-12: `UpdatePreservingComments` round-trips (Phase 11)

### Add in Later Milestones

- M2: `EvaluateRule` consuming `[[remote.branch_rule]]` and `[[repo.branch_override]]`
- M3: Fan-out executor consuming `[[sync_pair]]` and `[[remote]]`
- M7: Workspace/workstream lifecycle commands consuming `[[workspace]]` and `.gitw-stream`
- M8: Infra pattern commands consuming `track_branch`, `upstream`, `scope`
- M12: `pkg/migrate` consuming v1 detection results from CFG-10

---

## Feature Prioritization Matrix (M1 Only)

| Feature | User Value | Implementation Cost | Priority | Risk |
|---------|------------|---------------------|----------|------|
| CFG-07: Two-file merge | HIGH (enables private config) | HIGH | P1 | HIGH |
| CFG-12: Comment-preserving round-trips | HIGH (quality-of-life essential) | HIGH | P1 | HIGH |
| CFG-04: `[[remote]]` parsing | HIGH (foundation for M3/M4) | MEDIUM | P1 | MEDIUM |
| CFG-05: `[[sync_pair]]` + cycle detection | HIGH (foundation for M3) | MEDIUM | P1 | MEDIUM |
| CFG-08: `.gitw-stream` parsing | HIGH (foundation for M7+) | MEDIUM | P1 | MEDIUM |
| CFG-09: Cascade resolution | HIGH (enables default_remotes pattern) | MEDIUM | P1 | MEDIUM |
| CFG-01: `[[workspace]]` block | HIGH (foundation for M7+) | LOW | P1 | LOW |
| CFG-02: `[[repo]]` v2 fields | HIGH (foundation for M3/M8) | LOW | P1 | LOW |
| CFG-03: path enforcement | MEDIUM (v1 compat guard) | LOW | P1 | LOW |
| CFG-06: `[[workstream]]` root block | MEDIUM (foundation for M6+) | LOW | P1 | LOW |
| CFG-10: v1 detection | MEDIUM (migration guard) | LOW | P1 | LOW |
| CFG-11: `agentic_frameworks` field | LOW (bundled with CFG-01) | LOW | P1 | LOW |

**Priority key:**
- P1: All M1 features are P1 — there is no optionality within M1.

**Risk key (implementation risk within M1):**
- HIGH risk features need extra investigation time during planning. Specifically: CFG-07 and CFG-12.

---

## Sources

- `.planning/v2/v2-schema.md` — authoritative v2 config schema spec [HIGH confidence — primary source]
- `.planning/v2/v2-milestones.md` — M1 scope definition [HIGH confidence — primary source]
- `.planning/REQUIREMENTS.md` — CFG-01 through CFG-12 requirements [HIGH confidence — primary source]
- `pkg/config/config.go` — existing `WorkspaceConfig` struct and v1 type system [HIGH confidence — direct code inspection]
- `pkg/config/loader.go` — existing `Load`, `Save`, `saveWithCommentPreservation` functions [HIGH confidence — direct code inspection]
- `pkg/toml/preserve.go` — existing `UpdatePreservingComments` implementation [HIGH confidence — direct code inspection]

---
*Feature research for: git-w v2 M1 config schema and loader*
*Researched: 2026-04-02*
