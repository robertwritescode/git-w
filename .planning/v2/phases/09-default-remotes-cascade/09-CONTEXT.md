# Phase 9: Default remotes cascade - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement a pure cascade resolver in `pkg/config` that returns the effective remote list for a repo. Three-level cascade: `[metarepo] default_remotes` → `[[workstream]] remotes` → `[[repo]] remotes` (innermost wins, full replacement not merge). No CLI commands. No changes to the loading pipeline beyond fixing merge guards (see D-05 below).

Delivers: CFG-09

</domain>

<decisions>
## Implementation Decisions

### Zero-config fallback

- **D-01:** When nothing is configured at any level (no `default_remotes`, no workstream remotes, no repo remotes), the resolver returns an empty slice (`nil` / `[]string{}`).
- **D-02:** There is no implicit fallback to `["origin"]`. Empty result means "only git-native origin; no git-w secondary remotes."
- **D-03:** Document this in code comments: empty result is intentional and means the repo has no git-w secondary remotes — consistent with the spec statement "A repo with no remotes field and no `[metarepo] default_remotes` gets no secondary remotes."

### Function signature

- **D-04:** Two separate methods on `WorkspaceConfig`, consistent with existing `RepoByName` / `WorkstreamByName` methods:
  - `ResolveRepoRemotes(repoName string) ([]string, string)` — for callers without workstream context; walks repo → metarepo only.
  - `ResolveWorkstreamRemotes(repoName, workstreamName string) ([]string, string)` — for callers with workstream context; walks repo → workstream → metarepo.
- **D-05:** Both return `([]string, string)` — the resolved remote list AND a source-level string (`"repo"`, `"workstream"`, `"metarepo"`, `"none"`) indicating which level contributed the result. The source string is intended for dry-run output in later phases.

### Empty-slice semantics

- **D-06:** `nil` (field absent / not set) means "not configured at this level — fall through to the next outer level."
- **D-07:** `[]string{}` (explicit `remotes = []`) means "explicitly no remotes at this level — stop cascade, return empty." Consistent with Phase 6 D-06.
- **D-08:** The `nil` vs `[]string{}` distinction must be preserved through the cascade walk. The resolver checks `remotes != nil` (not `len > 0`) at each level to decide whether to stop or fall through.
- **D-09:** Fix `MergeWorkstream` and `MergeRepo` in `pkg/config/config.go` in this phase: change the `len(override.Remotes) > 0` guards to `override.Remotes != nil`. This is a prerequisite for D-08 to work correctly end-to-end when `.git/.gitw` overrides are involved.
- **D-10:** `MetarepoConfig.DefaultRemotes` follows the same semantics: `nil` = not set (return empty, source = `"none"`); `[]string{}` = explicitly empty (return empty, source = `"metarepo"`). The source distinction at the metarepo level is not practically significant today but is kept consistent.

### the agent's Discretion

- Exact private helper names used to implement the cascade walk.
- Whether `ResolveRepoRemotes` is implemented by delegating to `ResolveWorkstreamRemotes` with an empty workstream name, or as a standalone function.
- Exact wording of code comments documenting the nil vs empty distinction.

</decisions>

<specifics>
## Specific Ideas

- The source-level string returned alongside the remote list (`"repo"`, `"workstream"`, `"metarepo"`, `"none"`) is forward-looking: it exists to support dry-run output in later phases without requiring callers to re-derive the cascade. Keep it simple and consistent even if unused in Phase 9 itself.
- `MergeWorkstream` and `MergeRepo` guard fixes (D-09) are small and contained — they should be included in this phase rather than deferred, because the cascade resolver depends on them to behave correctly end-to-end.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Cascade resolution spec
- `.planning/v2/v2-schema.md` — `[metarepo] default_remotes` block definition and cascade semantics; `[[repo]] remotes` field; `[[workstream]] remotes` field
- `.planning/v2/v2-remote-management.md` — effective remote list behavior, what "innermost wins" means, empty-list explicit override intent

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-09 requirement definition and phase mapping

### Prior context decisions to carry forward
- `.planning/phases/06-add-workstream-root-config-block/06-CONTEXT.md` — D-06: `remotes = []` at workstream level = explicit none override (not fallback). Phase 9 extends this to all three levels.
- `.planning/phases/04-add-remote-and-remote-branch-rule/04-CONTEXT.md` — Remote naming and validation patterns established in Phase 4; resolver returns remote names (strings), not `RemoteConfig` structs.
- `.planning/phases/08-parse-gitw-stream-manifest/08-CONTEXT.md` — Most recent prior phase context; confirms loader and type patterns in use.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/config/config.go` `WorkspaceConfig.RepoByName` / `WorkstreamByName` — existing accessor pattern for the two new resolver methods to follow.
- `pkg/config/config.go` `ResolveDefaultBranch` — three-level cascade for `default_branch`; closest structural analog to the remote cascade. Study its nil/empty handling and fallback order.
- `pkg/config/config.go` `MergeWorkstream` (line 246) / `MergeRepo` (line 263) — need guard fixes per D-09.

### Established Patterns
- Methods on `WorkspaceConfig` use value receiver for read-only accessors (e.g. `AutoGitignoreEnabled`, `ResolveDefaultBranch`); new resolver methods should follow this.
- Existing cascade methods (e.g. `ResolveDefaultBranch`) use early-return guard clauses — the remote cascade should follow the same guard pattern.
- Source location: new methods belong in `pkg/config/config.go` alongside existing `WorkspaceConfig` accessors.

### Integration Points
- `MergeWorkstream` and `MergeRepo` guard changes are the only loader-touching changes in Phase 9; no new loader wiring is needed.
- No CLI command changes. The resolver is a pure config-layer function consumed by downstream phases.

</code_context>

<deferred>
## Deferred Ideas

- Resolver usage in CLI commands (push, fetch, sync) — downstream phases in M6/M7.
- Dry-run output that prints the source level — the `string` return value is wired now but used in a later phase.
- Workstream-scoped remote resolution for `.gitw-stream` manifests — out of scope for Phase 9; manifests do not carry a `remotes` field today.

</deferred>

---

*Phase: 09-default-remotes-cascade*
*Context gathered: 2026-04-07*
