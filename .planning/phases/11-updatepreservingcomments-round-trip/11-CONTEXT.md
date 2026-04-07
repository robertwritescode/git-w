# Phase 11: `UpdatePreservingComments` round-trip - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Add round-trip tests verifying that `UpdatePreservingComments` (via `config.Save` + `config.Load`) preserves comments at correct positions for all v2 block types present in `diskConfig`. Also fix two documented tech debt items in `pkg/toml/preserve.go`: replace `interface{}` with `any` throughout, and surface the silent error from `applySmartUpdate` instead of swallowing it.

No new config logic, no new block types. This is a test coverage + targeted cleanup phase.

Delivers: CFG-12

</domain>

<decisions>
## Implementation Decisions

### Test scope and coverage targets

- **D-01:** Tests cover all block types in `diskConfig`: `metarepo`, `[[workspace]]`, `[[repo]]` (with `[[repo.branch_override]]`), `[[remote]]` (with `[[remote.branch_rule]]`), `[[sync_pair]]`, `[[workstream]]`, `[groups]`, `[worktrees]`. Full coverage of the serialization surface.
- **D-02:** Tests exercise the production path — `config.Save()` then `config.Load()` on a real temp `.gitw` file — not `toml.UpdatePreservingComments` directly.
- **D-03:** Table-driven test structure: one test function with cases, one case per block type. Each case sets up a config with comments at/around the block under test.
- **D-04:** Assertion level: comments at correct positions. Assert that each comment appears in the output AND appears before its anchor key/section header (not just a `Contains` check). This catches comment displacement, not just comment loss.

### Tech debt fixes

- **D-05:** Replace all `interface{}` with `any` in `pkg/toml/preserve.go`. Affects: `Marshal`, `Unmarshal`, `UpdatePreservingComments`, `marshalBoth`, `parseBothToMaps`, `applySmartUpdate`, `mapsEqual`, `extractSectionContent`, `appendSection`, `saveWithCommentPreservation` in loader.go.
- **D-06:** Fix silent error swallow in `applySmartUpdate`: return the error from `smartUpdate` instead of silently falling back to `newBytes, nil`. The function signature already returns `([]byte, error)` — propagate the error.

### Test file location and structure

- **D-07:** New file `pkg/config/round_trip_test.go` — keeps round-trip tests separate from loader and stream tests, easy to find.
- **D-08:** Use a testify suite (`CmdSuite`-style) since the tests use `config.Save` + `config.Load` which require real temp `.gitw` files.
- **D-09:** Each table-driven subtest creates its own `t.TempDir()` for isolation. No shared workspace state in `SetupTest`.

### Agent's Discretion

- Exact comment placement in each test fixture (which lines carry comments, how many comments per case)
- Whether to add a single comprehensive "all blocks" integration case in addition to the per-block table cases
- Field naming and struct layout within test helper functions
- Whether `saveWithCommentPreservation` in `loader.go` needs updating alongside the `preserve.go` changes

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Round-trip fidelity spec
- `.planning/v2/v2-schema.md` — all v2 block types and their field shapes; defines the serialization surface to test
- `.planning/REQUIREMENTS.md` — CFG-12 requirement definition

### Codebase: TOML package
- `pkg/toml/preserve.go` — `UpdatePreservingComments`, `applySmartUpdate`, all functions using `interface{}` (tech debt fix target)
- `pkg/toml/preserve_test.go` — existing tests; shows current assertion patterns and what's already covered (must NOT regress)

### Codebase: config package
- `pkg/config/loader.go` — `diskConfig` struct (lines ~648-657), `prepareDiskConfig` (lines ~659-670), `saveWithCommentPreservation` (lines ~756-782), `Save` function (lines ~634-646)
- `pkg/config/config.go` — all v2 block type definitions (`RemoteConfig`, `SyncPairConfig`, `WorkstreamConfig`, `WorkspaceBlock`, `WorktreeEntry`, etc.)

### Tech debt reference
- `.planning/codebase/CONCERNS.md` — "Silent Error Swallowing in TOML Comment Preservation" and "`interface{}` Instead of `any`" entries document both fixes

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/toml/preserve_test.go` — existing test patterns; table-driven expansion should follow the same `assert.Contains` + position-check approach the new tests will use
- `pkg/testutil/suite.go` — `CmdSuite` base type; round-trip suite embeds this for `SetupTest`/`TeardownTest` lifecycle
- `pkg/config/loader.go` `Save` + `Load` — the integration path the tests will exercise; no mocking needed

### Established Patterns
- Test files in `pkg/config/` use `package config_test` (black-box); `round_trip_test.go` follows the same convention
- Table-driven test cases: `[]struct{ name string, ... }` with `s.Run(tc.name, func() { ... })`
- Per-subtest isolation: `s.T().TempDir()` inside the `s.Run` closure
- Comment assertions in existing tests use `assert.Contains(t, resultStr, "# comment text")`; new tests add position check (comment line index < anchor line index in the output)

### Integration Points
- `diskConfig` in `loader.go` is the complete serialization struct — all fields that can be round-tripped are declared there
- `saveWithCommentPreservation` in `loader.go` calls `toml.UpdatePreservingComments` and uses `interface{}` — needs updating as part of D-05
- `pkg/config/loader_test.go` and `stream_test.go` show existing test suite structure for the config package

</code_context>

<deferred>
## Deferred Ideas

- None — discussion stayed within phase scope

</deferred>

---

*Phase: 11-updatepreservingcomments-round-trip*
*Context gathered: 2026-04-07*
