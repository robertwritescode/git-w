# Phase 5: Add `[[sync_pair]]` parsing - Context

**Gathered:** 2026-04-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Parse `[[sync_pair]]` blocks from `.gitw` and `.git/.gitw`. Users can define explicit sync routing between remotes (from, to, optional refs). Cycle detection at load time prevents infinite sync loops. Define `MergeSyncPair` for Phase 7. No execution (fan-out executor is Phase 15).

Delivers: CFG-05 (`[[sync_pair]]` parsing and cycle detection).

Out of scope: sync fan-out execution (Phase 15), cascade resolution (Phase 7/9), ref filtering beyond parsing (post-v2.0), name-reference validation (from/to matching defined remotes).

</domain>

<decisions>
## Implementation Decisions

### SyncPairConfig struct

- **D-01:** `SyncPairs []SyncPairConfig` lives directly on `WorkspaceConfig` with TOML key `"sync_pair"` — no `diskConfig` split. Same array-of-tables pattern as `Remotes []RemoteConfig` established in Phase 4.
- **D-02:** `Refs []string` field uses `toml:"refs,omitempty"` — omitted from output when empty. Consumers treat `len(Refs) == 0` as "all refs". Keeps TOML files clean for the common case (no `refs = ["**"]` noise in serialized output).

### MergeSyncPair

- **D-03:** `MergeSyncPair(base, override SyncPairConfig) SyncPairConfig` is defined in Phase 5 alongside the struct — same pattern as `MergeRemote` in Phase 4. Phase 7 (two-file merge) calls it without re-deriving merge semantics.
- **D-04:** For the `Refs` field, the override file's value wins if non-empty (full replace, not union). If override `Refs` is nil/empty, base `Refs` is used. This is the non-zero-wins pattern consistent with `MergeRemote` scalar fields.

### Validation split

- **D-05:** Validation is split into two functions both called from `buildAndValidate`:
  1. `validateSyncPairFields(cfg)` — checks (a) each pair has non-empty `from` and `to`, (b) no duplicate `(from, to)` pairs. Both are hard errors (return `error`), consistent with `validateRemotes` duplicate-name behavior.
  2. `detectSyncCycles(cfg)` — DFS cycle detection on the directed sync graph. Hard error on first cycle found. Separate function makes each check independently testable.
- **D-06:** Name-reference validation (checking that `from`/`to` match a defined `[[remote]]` name) is NOT in Phase 5. Rationale: the referenced remote may live only in `.git/.gitw` while the pair is in `.gitw`; single-file validation would produce spurious errors before the two-file merge.

### Cycle detection design

- **D-07:** DFS with visited/in-stack sets — standard O(V+E) algorithm. Nodes are remote names; directed edges are `(from, to)` pairs from `[[sync_pair]]` blocks.
- **D-08:** Report first cycle only — stop at the first cycle found, return one error. No need to enumerate all cycles.
- **D-09:** Error format: `"sync_pair cycle detected: origin → personal → contractor → origin"` — full path with arrow notation, naming every node in the cycle. The path ends by repeating the node where the cycle closes.

### Agent's Discretion

- Exact field ordering on `SyncPairConfig` struct
- Internal DFS helper function names and signature
- Whether the DFS operates on `[]SyncPairConfig` directly or builds an adjacency map first
- Test table structure for cycle detection and field validation cases

</decisions>

<specifics>
## Specific Ideas

- `MergeSyncPair` should be a pure function with no side effects — same contract as `MergeRemote`.
- The cycle error path should repeat the starting node at the end so the cycle is visually obvious: `origin → personal → origin`, not `origin → personal`.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### `[[sync_pair]]` schema and fields
- `.planning/v2/v2-schema.md` — `[[sync_pair]]` block definition (`from`, `to`, `refs` fields); merge semantics by `(from, to)` key; full annotated config example showing sync_pair usage in `.git/.gitw`

### Remote management and sync
- `.planning/v2/v2-remote-management.md` — Sync fan-out motivation; `[[sync_pair]]` routing design; resolved design decisions table (ref routing, sync execution order)

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-05: `[[sync_pair]]` parsing and cycle detection requirement

### Prior phase context (carry-forward decisions)
- `.planning/phases/01-add-workspace-block/1-CONTEXT.md` — `buildAndValidate` is the single validation integration point; array-of-tables pattern for new top-level blocks
- `.planning/phases/04-add-remote-and-remote-branch-rule/04-CONTEXT.md` — `MergeRemote` pure function pattern; `WorkspaceConfig` direct array-of-tables; hard-error vs. warning distinction in `buildAndValidate`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/config/loader.go` `buildAndValidate`: Wire `validateSyncPairFields(cfg)` and `detectSyncCycles(cfg)` here — same pattern as `validateRemotes(cfgPath, cfg)` added in Phase 4.
- `pkg/config/config.go` `MergeRemote`: Pure function pattern for `MergeSyncPair` — field-by-field non-zero-wins merge.
- `pkg/config/loader.go` `validateRemotes`: Shows the error message and return pattern for structural validation; `validateSyncPairFields` follows the same shape.

### Established Patterns
- Array-of-tables directly on `WorkspaceConfig` (e.g. `Remotes []RemoteConfig \`toml:"remote"\``) — `SyncPairs []SyncPairConfig \`toml:"sync_pair"\`` follows the same pattern.
- Hard errors in `buildAndValidate` for structural violations (duplicate names, invalid enum values) — duplicate `(from, to)` pairs use the same pattern.
- `omitempty` on optional fields (e.g. `RepoConfig.Remotes`) — `SyncPairConfig.Refs` follows the same pattern.

### Integration Points
- `pkg/config/config.go` `WorkspaceConfig` struct — add `SyncPairs []SyncPairConfig` and `SyncPairConfig` type definition.
- `pkg/config/config.go` — add `MergeSyncPair(base, override SyncPairConfig) SyncPairConfig` pure function.
- `pkg/config/loader.go` `buildAndValidate` — wire `validateSyncPairFields(cfg)` and `detectSyncCycles(cfg)` calls.

</code_context>

<deferred>
## Deferred Ideas

- Name-reference validation (from/to must match defined [[remote]] names) — deferred; valid after Phase 7 two-file merge when both config files are merged
- Fan-out execution using sync_pair routes — Phase 15
- `MergeSyncPair` call sites and cascade resolution — Phase 7
- `refs` filtering beyond globs (by age, exclude tags) — post-v2.0 (POST-05)

</deferred>

---

*Phase: 05-add-sync-pair-parsing*
*Context gathered: 2026-04-04*
