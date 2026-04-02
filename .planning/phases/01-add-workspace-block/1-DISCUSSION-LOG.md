# Phase 1: Add `[[workspace]]` Block - Discussion Log

**Date:** 2026-04-02
**Phase:** 01-add-workspace-block
**Mode:** Interactive (no --auto, no advisor)

---

## Gray Areas Identified

Three gray areas were identified during codebase scouting:

1. **Config struct evolution strategy** — The existing `WorkspaceConfig.Workspace WorkspaceMeta toml:"workspace"` (v1 single-table) conflicts directly with the new `[[workspace]]` array-of-tables on the same TOML key.
2. **`pkg/agents` bootstrapping scope** — `agentic_frameworks` validation requires `agents.FrameworkFor()`, but `pkg/agents` does not exist yet.
3. **`[metarepo]` struct completeness** — `agentic_frameworks` lives in `[metarepo]`; question was whether to add all fields now or just the one needed.

---

## Round 1: Gray Area Selections

**Q: Config struct evolution strategy**
Options presented: (a) New V2Config type, (b) Extend in-place, (c) Dual-schema loader.

**Answer:** The existing `[workspace]` gets renamed to `[metarepo]` in this milestone. We can do that now, updating its struct names along the way.

*Resolution: Rename the existing v1 `[workspace]` to `[metarepo]` and update Go struct/field names to match. This eliminates the TOML key conflict entirely, enabling `[[workspace]]` to be introduced cleanly.*

---

**Q: `pkg/agents` bootstrapping scope**
Options presented: (a) Minimal `pkg/agents` now, (b) Inline in `pkg/config`.

**Answer:** Minimal `pkg/agents` now (Recommended).

*Resolution: Create `pkg/agents` with just the registry map and `FrameworkFor`/`FrameworksFor` functions. No interface or implementation yet.*

---

**Q: `[metarepo]` struct completeness**
Options presented: (a) Full `MetarepoConfig` now, (b) `agentic_frameworks` field only.

**Answer:** Full MetarepoConfig now (Recommended).

*Resolution: `MetarepoConfig` gets all known fields (`Name`, `DefaultRemotes`, `AgenticFrameworks`) to avoid struct churn in Phases 6 and 9.*

---

## Round 2: Rename Follow-on Questions

The config struct rename answer opened follow-on questions about execution details.

**Q: Rename + new struct in one step or two?**

**Answer:** Rename + new struct in Phase 1 (Recommended) — handle both in one coherent change.

---

**Q: Rename Go struct names to match?**

**Answer:** Rename structs to match (Recommended) — `WorkspaceMeta` -> `MetarepoConfig`, `Workspace` field -> `Metarepo`.

---

**Q: Migrate existing `.gitw` files?**

**Answer:** No migration in Phase 1 (Recommended) — existing files using `[workspace]` will break; migration is a separate concern.

---

## Summary of Decisions

| ID | Decision |
|----|----------|
| D-01 | Existing `[workspace]` TOML block renamed to `[metarepo]` |
| D-02 | Go types renamed: `WorkspaceMeta` -> `MetarepoConfig`, field `Workspace` -> `Metarepo` |
| D-03 | No `.gitw` file migration in Phase 1 |
| D-04 | Rename + new `[[workspace]]` struct introduced in one Phase 1 change |
| D-05 | `[[workspace]]` parses into `[]WorkspaceBlock` with `toml:"workspace"` |
| D-06 | Missing `agentic_frameworks` defaults to `["gsd"]` at validation time |
| D-07 | `MetarepoConfig` gets all fields: `Name`, `DefaultRemotes`, `AgenticFrameworks` |
| D-08 | Minimal `pkg/agents` created: registry map + `FrameworkFor`/`FrameworksFor` only |
| D-09 | Validation calls `agents.FrameworksFor(names)`, returns named `ErrUnknownFramework` |
