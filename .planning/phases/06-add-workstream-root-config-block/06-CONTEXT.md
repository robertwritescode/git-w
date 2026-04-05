# Phase 6: Add `[[workstream]]` root config block - Context

**Gathered:** 2026-04-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Add root-level `[[workstream]]` config entries for lightweight per-workstream remote overrides used by downstream cascade resolution.

Delivers: CFG-06 (`[[workstream]]` root config blocks for remote overrides).

Out of scope: two-file merge behavior implementation (Phase 7), cascade resolver implementation (Phase 9/16), workstream lifecycle commands (M7), and any policy-mode feature for shared-vs-private workstream placement.

</domain>

<decisions>
## Implementation Decisions

### Placement and file policy

- **D-01:** `[[workstream]]` blocks are allowed in both `.gitw` and `.git/.gitw` in Phase 6.
- **D-02:** Presence of `[[workstream]]` in `.gitw` is accepted with no warning and no error.
- **D-03:** Phase 6 treats `.gitw` usage as intentional, and docs/error text should explicitly state that this is valid behavior (not misconfiguration).

### Block shape and parsing contract

- **D-04:** Phase 6 parses `[[workstream]]` with exactly two fields: `name` and `remotes`.
- **D-05:** `remotes` key is required to be present, but `remotes = []` is valid.
- **D-06:** Semantic meaning of `remotes = []` is explicit none override at workstream level (not fallback to outer defaults).
- **D-07:** Schema is strict for this block: unknown extra keys in `[[workstream]]` are load errors.

### Validation behavior in Phase 6

- **D-08:** `name` is required and must be non-empty for each `[[workstream]]` entry.
- **D-09:** Duplicate `[[workstream]]` names in the same loaded file are hard errors.
- **D-10:** `[[workstream]].remotes` values are validated immediately against declared `[[remote]]` names in Phase 6.
- **D-11:** Unknown remote names in a workstream remote list are hard errors, failing load on first unknown with actionable naming.
- **D-12:** Duplicate names within one `remotes` list (for example `"origin", "origin"`) are validation errors.

### Ordering and normalization (intentional Phase 6 exception)

- **D-13:** In-memory `[[workstream]]` entries are normalized/sorted by `name` instead of preserving declaration order.
- **D-14:** `remotes` list values are normalized/sorted rather than preserving user-entered order.
- **D-15:** This differs from prior array-of-table declaration-order patterns and is intentional for Phase 6.

### the agent's Discretion

- Exact helper/function naming for `[[workstream]]` validation and normalization steps.
- Exact error message wording while preserving decision-level semantics above.
- Internal representation details used to support strict-key validation in `go-toml/v2` loader flow.

</decisions>

<specifics>
## Specific Ideas

- Shared-team workflows are a valid reason to keep some `[[workstream]]` blocks in committed `.gitw`; this phase should not force private-only placement.
- `remotes = []` is intentional signal, not omission.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 6 roadmap and requirement anchors
- `.planning/ROADMAP.md` — Phase 6 goal, success criteria, and canonical refs for this phase
- `.planning/REQUIREMENTS.md` — CFG-06 requirement definition and phase mapping

### v2 schema and remote/cascade behavior
- `.planning/v2/v2-schema.md` — `[[workstream]]` block schema and cascade model (`[metarepo]` -> `[[workstream]]` -> `[[repo]]`)
- `.planning/v2/v2-remote-management.md` — effective remote list behavior and workstream remote override intent

### Prior context decisions to carry forward
- `.planning/phases/01-add-workspace-block/1-CONTEXT.md` — top-level array-of-table pattern and `buildAndValidate` as validation integration point
- `.planning/phases/04-add-remote-and-remote-branch-rule/04-CONTEXT.md` — remote naming, validation patterns, and hard-error policy shape
- `.planning/phases/05-add-sync-pair-parsing/05-CONTEXT.md` — load-time validation split patterns and deferred-reference-check precedent (not followed for workstream refs in this phase)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/config/config.go` — established top-level schema structs (`RemoteConfig`, `SyncPairConfig`) and merge helpers provide the direct model for adding `WorkstreamConfig`.
- `pkg/config/loader.go` `buildAndValidate` — single validation orchestration point for all load-time schema checks.
- `pkg/config/loader.go` `diskConfig` list fields (`RemoteList`, `SyncPairList`, `RepoList`) — existing deserialization pattern for array-of-table blocks.

### Established Patterns
- New schema blocks are represented as slices on `WorkspaceConfig` and loaded via dedicated disk struct fields.
- Validation uses hard errors for structural violations (missing required keys, duplicates, invalid enum/reference values).
- Load-time checks and config warnings are intentionally separated; warnings are reserved for non-fatal policy signals.

### Integration Points
- Add `WorkstreamConfig` type and `Workstreams` collection in `pkg/config/config.go`.
- Extend loader deserialization and `buildAndValidate` chain in `pkg/config/loader.go` to parse/validate/normalize `[[workstream]]`.
- Ensure behavior aligns with later Phase 7/9 consumers without implementing merge/cascade execution in this phase.

</code_context>

<deferred>
## Deferred Ideas

- Configurable placement mode (policy switch controlling whether committed `.gitw` may contain `[[workstream]]`) is a separate capability and should be scheduled as its own future phase.

</deferred>

---

*Phase: 06-add-workstream-root-config-block*
*Context gathered: 2026-04-05*
