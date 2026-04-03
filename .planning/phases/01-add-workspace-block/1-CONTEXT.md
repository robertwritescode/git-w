# Phase 1: Add `[[workspace]]` Block - Context

**Gathered:** 2026-04-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can define `[[workspace]]` blocks in `.gitw` and the tool validates `agentic_frameworks` against the framework registry. Delivers CFG-01 (`[[workspace]]` parsing) + CFG-11 (`agentic_frameworks` validation).

Out of scope: workspace commands, worktree commands, any downstream consumers of the workspace data. This phase is purely config schema + validation.

</domain>

<decisions>
## Implementation Decisions

### Config struct rename ([workspace] -> [metarepo])

- **D-01:** The existing v1 `[workspace]` TOML block is renamed to `[metarepo]` in this phase. This unblocks the new `[[workspace]]` array-of-tables on the same key without any TOML conflict.
- **D-02:** Go struct names and field names are renamed to match: `WorkspaceMeta` -> `MetarepoConfig`, and the field on `WorkspaceConfig` is renamed `Workspace` -> `Metarepo` (with `toml:"metarepo"`).
- **D-03:** No migration of existing `.gitw` files or test fixtures in Phase 1. Existing files using `[workspace]` will break silently; migration is a separate concern for a later phase.

### New [[workspace]] array-of-tables

- **D-04:** Phase 1 handles both the `[workspace]` -> `[metarepo]` rename AND introduces the new `[[workspace]]` array-of-tables struct in one coherent change.
- **D-05:** `[[workspace]]` parses into a `[]WorkspaceBlock` field on `WorkspaceConfig` with `toml:"workspace"`. Each `WorkspaceBlock` has at minimum: `Name string`, `AgenticFrameworks []string`.
- **D-06:** Missing `agentic_frameworks` field defaults to `["gsd"]` during config loading/validation (not at TOML parse time).

### [metarepo] struct completeness

- **D-07:** `MetarepoConfig` is introduced with ALL known fields now: `Name string`, `DefaultRemotes []string`, `AgenticFrameworks []string`. This avoids struct churn in Phases 6 and 9 which need those fields.

### pkg/agents bootstrapping

- **D-08:** Create a minimal `pkg/agents` package in Phase 1 containing only: a registry map of valid framework names and `FrameworkFor(name string) (Framework, bool)` / `FrameworksFor(names []string) error` functions for validation. No full interface or implementation — those are deferred to Phase 48.
- **D-09:** `agentic_frameworks` validation in `pkg/config` calls `agents.FrameworksFor(names)` and returns a named error (e.g., `ErrUnknownFramework`) identifying the invalid value.

### the agent's Discretion

- Exact Go type for `Framework` in `pkg/agents` (string alias, struct, or iota — pick what fits the minimal scope)
- Whether `FrameworkFor` and `FrameworksFor` are separate functions or one entry point
- Error message wording for unknown framework names
- Exact field ordering within new structs

</decisions>

<specifics>
## Specific Ideas

- The `[workspace]` -> `[metarepo]` rename is an intentional v2 schema migration, not incidental refactoring. Treat the rename as the primary structural change that enables the new `[[workspace]]` array-of-tables.
- `pkg/agents` must be bootstrapped now because `agentic_frameworks` validation is a Phase 1 success criterion — it cannot be deferred.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v2 config schema
- `.planning/v2/v2-schema.md` — Full v2 config schema: `[[workspace]]` block structure, `[metarepo]` block fields, `agentic_frameworks` field spec and valid values

### v2 milestone scope
- `.planning/v2/v2-milestones.md` — M1 scope and implementation notes; confirms Phase 1 boundaries and what is deferred

### agentic_frameworks validation
- `.planning/v2/v2-agent-interop.md` — `agentic_frameworks` validation logic, `pkg/agents` interface design, framework registry spec

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-01 (`[[workspace]]` parsing) and CFG-11 (`agentic_frameworks` validation) — the two requirements delivered by this phase

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/config/loader.go` `buildAndValidate`: Existing validation hook function — `agentic_frameworks` validation should be wired in here alongside existing validation logic.
- `pkg/toml` `UpdatePreservingComments`: Already exists; not directly used in Phase 1 but relevant context for how config writes are handled elsewhere.

### Established Patterns
- `WorkspaceConfig` in `pkg/config/config.go`: The struct being renamed and extended. `Workspace WorkspaceMeta toml:"workspace"` becomes `Metarepo MetarepoConfig toml:"metarepo"` plus a new `Workspaces []WorkspaceBlock toml:"workspace"` field.
- `loadMainConfig` -> `buildAndValidate` flow in `pkg/config/loader.go`: All validation must go through `buildAndValidate`. No validation logic outside this path.
- Domain package convention: `pkg/agents` must export only `Register(*cobra.Command)` in `register.go` if it has commands. Since Phase 1's `pkg/agents` is registry-only (no commands), it may not need `register.go` at all — just exported registry functions.

### Integration Points
- `pkg/config` calls `pkg/agents` for framework validation — unidirectional dependency, no cycle.
- All callers of `WorkspaceConfig.Workspace` (type `WorkspaceMeta`) throughout the codebase must be updated to `WorkspaceConfig.Metarepo` (type `MetarepoConfig`).

</code_context>

<deferred>
## Deferred Ideas

- Migration of existing `.gitw` files that use `[workspace]` — separate concern, later phase.
- Full `pkg/agents` interface and implementation (agent discovery, execution) — Phase 48.
- `default_remotes` cascade behavior — Phase 9.
- Workspace commands (`git w workspace list`, etc.) — later phases in M1.

</deferred>

---

*Phase: 01-add-workspace-block*
*Context gathered: 2026-04-02*
