# Phase 8: Parse `.gitw-stream` manifest - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement a loader for `.gitw-stream` manifest files — a new, self-contained file format entirely separate from the `.gitw` loading path. Delivers `WorkstreamManifest` type with `[[worktree]]` entries, default resolution, and load-time uniqueness validation. No CLI commands. No changes to `Load()` or the `.gitw` merge pipeline.

</domain>

<decisions>
## Implementation Decisions

### Type placement and package

- **D-01:** `WorkstreamManifest`, `WorktreeEntry`, `ShipState`, `StreamContext`, and `WorkstreamStatus` types go in `pkg/config/config.go` alongside existing config types. The loader function lives in a new `pkg/config/stream.go`. All config I/O remains in one package.
- **D-02:** `status` field uses a `WorkstreamStatus` typed string alias with constants `StatusActive`, `StatusShipped`, `StatusArchived` — consistent with the `BranchAction` pattern already in `config.go`.

### Loader API design

- **D-03:** Public entrypoint is `LoadStream(path string) (*WorkstreamManifest, error)` — takes explicit path to a `.gitw-stream` file. No discovery; callers control path resolution. Consistent with `config.Load(path)`.
- **D-04:** Returns `(*WorkstreamManifest, error)`. Pointer return, nil on error.
- **D-05:** Missing file (`os.ErrNotExist`) is returned unwrapped. Callers check with `errors.Is(err, os.ErrNotExist)`. Consistent with how `mergeLocalConfig` handles missing `.gitw.local`.

### Scope of [ship] and [context] blocks

- **D-06:** Define full `ShipState` and `StreamContext` types with all schema-specified fields now (`pr_urls`, `pre_ship_branches`, `shipped_at`, `summary`, `key_decisions`). Parse them in `LoadStream`. Types are ready when M9/M10 phases need them.
- **D-07:** Phase 8 tests cover all defined fields including `[ship]` and `[context]` — full parse round-trip coverage, not just the success criteria fields.

### Default and uniqueness rules

- **D-08:** `name` and `path` defaults are applied inside `LoadStream` as a build step after TOML parsing, before validation. Every `WorktreeEntry` in the returned manifest has both fields populated. Single-occurrence repos get `name` defaulted to `repo`; `path` defaults to `name` when omitted.
- **D-09:** A single `validateStream(manifest *WorkstreamManifest) error` function handles all validation: `name` uniqueness, `path` uniqueness, and the multi-occurrence name-required check. Called from `LoadStream` after defaults are applied.
- **D-10:** Multi-occurrence name-required check: when a repo appears more than once in `[[worktree]]` and any of those entries has an empty `name` field (before defaulting), produce error: `"worktree entry for repo %q requires a name when the repo appears multiple times"`.

### Agent's Discretion

- Exact field names on `ShipState` and `StreamContext` Go structs (TOML tags must match schema field names).
- Internal helper names within `stream.go` (`applyStreamDefaults`, etc.).
- Table layout and test structure within `stream_test.go`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Manifest schema and field semantics
- `.planning/v2/v2-schema.md` — `.gitw-stream` manifest definition (lines 226–279), `[[worktree]]` field semantics (`name` defaulting, `path` defaulting, uniqueness constraints), `[ship]` and `[context]` sub-block field definitions, full annotated manifest example
- `.planning/v2/v2-infra-patterns.md` — Pattern B worktree fields (`name`, `path`, `scope`) in context, multi-worktree per repo semantics

### Codebase integration points
- `pkg/config/config.go` — `BranchAction` typed string pattern (model for `WorkstreamStatus`), existing `Merge*` helpers, `WorkspaceConfig` for type placement reference
- `pkg/config/loader.go` — `mergeLocalConfig` for silent-skip pattern on missing optional files, `buildAndValidate` for validation integration model

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `BranchAction` type in `config.go` — exact pattern to follow for `WorkstreamStatus` typed string with constants
- `mergeLocalConfig` in `loader.go` — shows silent `os.ErrNotExist` skip pattern; `LoadStream` inverts this (returns the error, callers skip)
- `validateRemotes`, `validateSyncPairFields`, `detectSyncCycles` — show the single-function validation pattern; `validateStream` follows the same shape
- `toml.Unmarshal` + raw bytes pattern — established in `loadMainConfig`; `stream.go` uses the same

### Established Patterns
- New block types define a disk-facing struct (for TOML tags) and in-memory type on `WorkspaceConfig`; `WorkstreamManifest` is entirely self-contained so no `diskConfig` split needed
- `buildAndValidate` receives a single merged config; `LoadStream` has its own parse → default → validate pipeline (no shared validation path with `.gitw`)
- Error messages: lowercase, no trailing period, quoted identifiers with `%q`, actionable suggestion

### Integration Points
- `pkg/config/stream.go` — new file; `LoadStream` is the only exported symbol in Phase 8
- `pkg/config/config.go` — add `WorkstreamManifest`, `WorktreeEntry`, `ShipState`, `StreamContext`, `WorkstreamStatus` types here
- No changes to `Load()`, `LoadCWD()`, or `LoadConfig()` — `.gitw-stream` loading is completely separate

</code_context>

<specifics>
## Specific Ideas

- The full annotated manifest example in `v2-schema.md` (lines 239–279) is the primary fixture reference for tests.
- `pre_ship_branches` in `[ship]` is a map: `worktree-name -> branch-name-pre-ship-<timestamp>`. Go type: `map[string]string`.
- `pr_urls` in `[ship]` is a string slice. Go type: `[]string`.
- `key_decisions` in `[context]` is a string slice. Go type: `[]string`.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 08-parse-gitw-stream-manifest*
*Context gathered: 2026-04-06*
