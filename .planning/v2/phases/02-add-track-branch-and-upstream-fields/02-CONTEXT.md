# Phase 2: Add `track_branch` and `upstream` Fields - Context

**Gathered:** 2026-04-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can annotate `[[repo]]` blocks with `track_branch` and `upstream` string fields for branch-per-env infrastructure patterns. This phase also migrates `RepoConfig` from the v1 `map[string]RepoConfig` (TOML: `[repos.<n>]`) to the v2 `[]RepoConfig` array-of-tables (`[[repo]]` with a required `name` field), and renames `URL` to `CloneURL` (TOML: `clone_url`) as part of the same struct alignment.

Delivers: CFG-02 (`track_branch` and `upstream` fields on `[[repo]]`).

Out of scope: sync behavior using `track_branch` (Phase 17), env-group expansion (Phase 43), `--upstream` filter commands (Phase 44), clone URL resolution for aliases (Phase 42), any commands consuming these fields.

</domain>

<decisions>
## Implementation Decisions

### Alias recognition at load time

- **D-01:** Load-time validation enforces that `upstream` is not empty when `track_branch` is set (and vice versa). Both fields must appear together or not at all — either alone is a load-time error.
- **D-02:** Within a given `upstream` group (all repos sharing the same `upstream` value), `track_branch` values must be unique. Duplicate `track_branch` within an upstream group is a load-time error.
- **D-03:** `upstream` is a free-form grouping label (e.g., `"infra"` derived from the URL basename). It does NOT need to match any `[[repo]]` name — there is no such repo; it is a logical concept, not a pointer to a config entry.
- **D-04:** Clone URL handling for alias repos (how an alias knows its remote URL) is deferred to Phase 42 (`git w repo add --branch-map`).

### `IsAlias()` helper

- **D-05:** Whether to add an `IsAlias() bool` method on `RepoConfig` is at agent discretion. The validation logic above implicitly identifies aliases; a helper method is allowed if it aids readability.

### `[[repo]]` array-of-tables migration

- **D-06:** Phase 2 migrates `RepoConfig` from `map[string]RepoConfig` (TOML key `repos`) to `[]RepoConfig` (TOML key `repo`, array-of-tables `[[repo]]`). This is a breaking TOML change and is intentional — v2 targets `[[repo]]` throughout.
- **D-07:** `name` is a required field on every `[[repo]]` entry. The map key is eliminated; repo names come from the `name` field only. Missing `name` produces a load-time error.
- **D-08:** In-memory access to repos (currently `cfg.Repos["name"]`) will need to be updated to index on the `Name` field. A helper method (e.g., `RepoByName(name string) (RepoConfig, bool)`) should be added to `WorkspaceConfig` to replace direct map lookups.

### Field naming alignment

- **D-09:** `URL string toml:"url"` on `RepoConfig` is renamed to `CloneURL string toml:"clone_url"` in this same phase. The rename is part of the same struct migration pass — one cohesive breaking change rather than two separate ones.

### the agent's Discretion

- Whether `IsAlias()` is added as a method on `RepoConfig`
- Exact error message wording for validation failures
- Internal helper design for `RepoByName` (method, free function, or map rebuild)
- Field ordering within the updated `RepoConfig` struct

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v2 config schema
- `.planning/v2/v2-schema.md` — Full `[[repo]]` block definition: `name`, `path`, `clone_url`, `track_branch`, `upstream`, `remotes` fields; array-of-tables structure; annotated example with Pattern A aliases

### Infra patterns
- `.planning/v2/v2-infra-patterns.md` — Pattern A (branch-per-env) explanation: how `track_branch` and `upstream` are used, what env-group means, sync behavior for aliases (for context only — sync is out of scope for Phase 2)

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-02: `track_branch` and `upstream` fields on `[[repo]]` blocks for env aliases

### Prior phase context
- `.planning/phases/01-add-workspace-block/1-CONTEXT.md` — Phase 1 decisions: `buildAndValidate` is the validation integration point; struct patterns established; `pkg/agents` bootstrapping

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/config/loader.go` `buildAndValidate`: All new validation (`track_branch`/`upstream` rules, `track_branch` uniqueness per upstream group) must wire in here alongside existing validation.
- `WorkspaceConfig.ResolveDefaultBranch` and `WorkspaceConfig.BranchSyncSourceEnabled`: Existing boolean accessor pattern on `WorkspaceConfig` — `IsAlias()` on `RepoConfig` (if added) should follow the same nil-guard pattern used throughout.
- `pkg/config/config_test.go`: Existing config tests exercise `WorkspaceConfig` accessors — new tests for the array-of-tables `[[repo]]` parsing and alias validation should follow the same table-driven pattern.

### Established Patterns
- `map[string]RepoConfig` (current): Used throughout `pkg/repo/filter.go`, `pkg/repo/repo.go`, and domain command packages via `cfg.Repos["name"]` lookups. All these call sites must be updated to use the new `RepoByName` helper after migration.
- `buildAndValidate` validation hook: The single entry point for all load-time validation. Both `track_branch`/`upstream` co-presence check and `track_branch` uniqueness per group must go here.
- Phase 1 established `WorkspaceBlock` as an array-of-tables struct with a required `name` field — `RepoConfig` follows the same pattern.

### Integration Points
- All callers of `cfg.Repos` map throughout domain packages (`pkg/repo/`, `pkg/git/`, `pkg/worktree/`, `pkg/workgroup/`, `pkg/branch/`) must be updated from map access to `cfg.RepoByName(name)` or equivalent after the `[[repo]]` migration.
- `pkg/config/loader.go` TOML deserialization: The `diskConfig` struct (if it exists as a separate deserialization target) needs updating; otherwise the change is directly on `WorkspaceConfig`.
- Existing `.gitw` test fixtures using `[repos.<n>]` format must be updated to `[[repo]]` format throughout the test suite.

</code_context>

<specifics>
## Specific Ideas

- The `[[repo]]` array-of-tables migration is intentionally bundled with the field additions — one cohesive breaking change is better than two. The `[workspace]` → `[metarepo]` rename in Phase 1 established the precedent.
- `upstream` is a free-form label derived from URL basename by convention (e.g., `github.com/work-org/infra` → `"infra"`), but git-w does not enforce this derivation at Phase 2. That is Phase 42's concern.

</specifics>

<deferred>
## Deferred Ideas

- Sync behavior using `track_branch` as pull target — Phase 17
- `ResolveEnvGroup` / `--env-group` expansion — Phase 43
- `git w repo list --upstream` and `git w status --repo <upstream>` — Phase 44
- `git w repo add --branch` / `--branch-map` for creating aliases with URL — Phase 42
- Clone URL resolution for alias repos — Phase 42

</deferred>

---

*Phase: 02-add-track-branch-and-upstream-fields*
*Context gathered: 2026-04-03*
