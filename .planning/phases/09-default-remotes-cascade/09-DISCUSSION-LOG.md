# Phase 9: Default remotes cascade - Discussion Log

**Date:** 2026-04-07
**Phase:** 09-default-remotes-cascade
**Status:** All gray areas resolved — ready for planning

---

## Context

Phase 9 implements the three-level cascade resolver in `pkg/config`. No CLI commands. No loader pipeline changes beyond fixing two merge guards that block correct empty-slice behavior.

Three gray areas were identified and discussed.

---

## Gray Area 1: Zero-config fallback

**Question:** When no remotes are configured at any level (no `default_remotes`, no workstream remotes, no repo remotes), should the resolver return `["origin"]` implicitly, or an empty slice?

**Options presented:**
- Return `["origin"]` implicitly — familiar git default; callers always get something.
- Return empty slice — explicit, honest; callers must handle empty themselves.

**Decision:** Return empty slice. Empty means "no git-w secondary remotes." The spec is explicit: "A repo with no remotes field and no `[metarepo] default_remotes` gets no secondary remotes." No implicit `origin` fallback.

---

## Gray Area 2: Function signature

**Question:** One function or two? What return type?

**Options presented:**
1. Single function `ResolveRemotes(repoName string, workstreamName ...string) []string` — variadic for optional workstream.
2. Two methods: `ResolveRepoRemotes(repoName string)` and `ResolveWorkstreamRemotes(repoName, workstreamName string)` — explicit split matching existing `RepoByName`/`WorkstreamByName` pattern.
3. Return `[]string` only vs. return `([]string, string)` with source level.

**Decision:** Two separate methods on `WorkspaceConfig`, both returning `([]string, string)`:
- `ResolveRepoRemotes(repoName string) ([]string, string)`
- `ResolveWorkstreamRemotes(repoName, workstreamName string) ([]string, string)`

The second return value is a source-level string (`"repo"`, `"workstream"`, `"metarepo"`, `"none"`) for dry-run output in later phases.

---

## Gray Area 3: Empty-slice semantics

**Sub-question A:** Should `remotes = []` (empty slice) at an inner level stop the cascade and return empty, or fall through to the next outer level?

**Decision:** Stop cascade, return empty. Empty slice = explicit "no remotes" override. This is consistent with Phase 6 D-06 which established that `remotes = []` at the workstream level means explicit none, not fallback. The resolver checks `remotes != nil` (not `len > 0`) at each level.

**Sub-question B:** `MergeWorkstream` and `MergeRepo` currently use `len(override.Remotes) > 0` as the merge guard, which means a `remotes = []` override in `.git/.gitw` is silently ignored. Should this be fixed in Phase 9?

**Decision:** Fix in Phase 9. Change both guards to `override.Remotes != nil`. Small, contained change. Required for the cascade resolver to work correctly end-to-end when `.git/.gitw` carries an empty-remotes override.

---

## Summary of Decisions

| # | Area | Decision |
|---|------|----------|
| D-01 | Zero-config fallback | Return empty slice (nil/`[]string{}`) |
| D-02 | Zero-config fallback | No implicit `["origin"]` fallback |
| D-03 | Zero-config fallback | Document in code: empty = no git-w secondary remotes |
| D-04 | Function signature | Two methods: `ResolveRepoRemotes` and `ResolveWorkstreamRemotes` |
| D-05 | Function signature | Both return `([]string, string)` — list + source level |
| D-06 | Empty-slice semantics | `nil` = not set, fall through |
| D-07 | Empty-slice semantics | `[]string{}` = explicit none, stop cascade |
| D-08 | Empty-slice semantics | Resolver checks `!= nil`, not `len > 0` |
| D-09 | Empty-slice semantics | Fix `MergeWorkstream`/`MergeRepo` guards to `!= nil` in Phase 9 |
| D-10 | Empty-slice semantics | Same nil vs empty semantics apply at metarepo level |

---

*Discussion log: 09-default-remotes-cascade*
*Recorded: 2026-04-07*
