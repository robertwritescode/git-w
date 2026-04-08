# Phase 7: Two-file config merge - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Extend the config loader to read `.git/.gitw` (private, never committed) alongside `.gitw` (shared, committed) and merge the two at field level. Private file wins on all conflicts. All v2 block types participate in the merge. No new CLI commands or output changes — this is purely a loader change.

</domain>

<decisions>
## Implementation Decisions

### Validation sequencing

- **D-01:** Parse each file individually (TOML unmarshal only), then run the full `buildAndValidate` pass on the **merged result** only. Cross-reference checks (e.g. workstream referencing a remote defined in the other file) must resolve against the merged config, not per-file. This matches the intentional split pattern shown in v2-schema.md where `.git/.gitw` workstreams reference remotes from `.gitw`.

### Block-type merge semantics

- **D-02:** `[[remote]]` — merge by `name`; field-level override using the existing `MergeRemote` helper. Private entries with names not in `.gitw` are added as new entries.
- **D-03:** `[[sync_pair]]` — merge by `(from, to)` pair; field-level override using the existing `MergeSyncPair` helper. New pairs from `.git/.gitw` are added.
- **D-04:** `[[workstream]]` — merge by `name`; field-level override using the existing `MergeWorkstream` helper. New workstream entries from `.git/.gitw` are added.
- **D-05:** `[[repo]]` — field-level override using a new `MergeRepo` helper. **No new repos from `.git/.gitw`** — only repos already declared in `.gitw` can be overridden. A repo name in `.git/.gitw` that does not exist in `.gitw` is a load-time error.
- **D-06:** `[[workspace]]` — full field-level merge + new entries allowed. New workspace blocks from `.git/.gitw` are added; existing ones are field-merged using a new `MergeWorkspace` helper. Same pattern as `MergeRemote`.
- **D-07:** `[metarepo]` — field-level override. Non-zero private values win per field.

### Absent `.git/.gitw`

- **D-08:** Missing `.git/.gitw` is silently skipped (no error, no warning). Consistent with how `mergeLocalConfig` handles a missing `.gitw.local` today.
- **D-09:** If a `--debug` flag is present on the root command, emit a single-line trace to stderr when `.git/.gitw` is absent: `debug: .git/.gitw not found, skipping private config`. No trace in normal (non-debug) operation. Note: if `--debug` does not yet exist on the root command, this trace is deferred to when it is added — the silent-skip behavior is sufficient for Phase 7.

### Load entrypoint shape

- **D-10:** Extend `Load()` transparently. It already handles `.gitw` + `.gitw.local`; it now also reads `.git/.gitw` as a third merge layer. No new public entrypoints. All existing callers of `Load()`, `LoadCWD()`, and `LoadConfig()` pick up private merge automatically. The private config path is derived from the main config path: `.git/.gitw` lives at `filepath.Join(filepath.Dir(cfgPath), ".git", ".gitw")`.

### the agent's Discretion

- Error message style for D-05 (unknown repo name in `.git/.gitw`): follow the existing pattern — lowercase, quoted identifier, actionable message.
- Where in `Load()` the private merge step sits relative to `.gitw.local` merge: agent decides (likely after `.gitw.local` since private config is the most specific layer).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Config schema and merge semantics
- `.planning/v2/v2-schema.md` — Two-file merge model, block-type merge keys, `private = true` enforcement, full annotated `.gitw` + `.git/.gitw` config examples, implementation notes listing `MergeRepo` / `MergeWorkspace` as required helpers

### Codebase integration points
- `pkg/config/loader.go` — `Load()`, `loadMainConfig()`, `mergeLocalConfig()`, `buildAndValidate()`, `validateRemotes()` (private enforcement already wired); the new private merge step fits into this file
- `pkg/config/config.go` — `MergeRemote`, `MergeSyncPair`, `MergeWorkstream` already implemented; `MergeRepo` and `MergeWorkspace` are new additions needed in this file

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `MergeRemote(base, override RemoteConfig) RemoteConfig` — pure field-level merge helper; `MergeRepo` and `MergeWorkspace` should follow the identical pattern
- `MergeSyncPair`, `MergeWorkstream` — same pattern; all three live in `pkg/config/config.go`
- `mergeLocalConfig(cfg, localPath)` — existing second-file merge in `loader.go`; the private merge step mirrors this structure
- `buildAndValidate(configPath, cfg)` — single validation integration point; called once on the merged result (D-01)
- `validateRemotes(cfgPath, cfg)` — already checks `private = true` enforcement by testing if cfgPath ends with `.git/.gitw`; this check must be applied to the source file path, not the merged config path

### Established Patterns
- Parse raw bytes into `diskConfig` struct via `toml.Unmarshal`, then build in-memory `WorkspaceConfig`; the private file uses the same `diskConfig` shape
- Array-of-tables blocks (`[[remote]]`, `[[repo]]`, etc.) are merged by their key field; the key fields are: `name` for remotes/repos/workspaces/workstreams, `(from, to)` for sync_pairs
- Silent skip on `os.ErrNotExist` for optional files (see `mergeLocalConfig`)

### Integration Points
- `Load()` in `loader.go:19` — add private merge as a new step between `loadMainConfig` and `mergeLocalConfig`, or after `mergeLocalConfig` (agent decides ordering)
- `validateRemotes` must keep its per-file `cfgPath` check: when validating the shared file, reject `private = true`; when reading the private file's remotes before merge, that check is already satisfied

</code_context>

<specifics>
## Specific Ideas

- The v2-schema.md "Full annotated config example" section (lines 298–438) is the canonical reference for what a real two-file setup looks like. Downstream agents should use it as the primary test fixture reference.
- `private = true` enforcement: already implemented in `validateRemotes` using `strings.HasSuffix(filepath.ToSlash(cfgPath), ".git/.gitw")`. Phase 7 does not change this check — it remains on the shared file validation path.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 07-two-file-config-merge*
*Context gathered: 2026-04-06*
