# Phase 10: Detect v1 `[[workgroup]]` blocks - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Detect v1 `[[workgroup]]` blocks in `.gitw` at load time and return a hard error directing the user to `git w migrate`. Detection only — no migration logic runs. This is a load-time **error** (not a warning); loading fails when v1 workgroup blocks are present.

Delivers: CFG-10

</domain>

<decisions>
## Implementation Decisions

### Detection mechanism

- **D-01:** Add a detection field to `diskConfig` tagged `toml:"workgroup,omitempty"` with type `[]map[string]any`. This captures any `[[workgroup]]` entries without needing to know v1 field names. Detection only needs presence, not values.
- **D-02:** After `toml.Unmarshal` populates `dc`, set `cfg.V1WorkgroupCount = len(dc.WorkgroupList)` in `loadMainConfig` before calling `buildAndValidate`. The `WorkspaceConfig` carries the count into validation.
- **D-03:** `V1WorkgroupCount int` field on `WorkspaceConfig` — an int rather than bool, so the error message can say "found N `[[workgroup]]` blocks".

### Pipeline placement

- **D-04:** `detectV1Workgroups` is the **first check** in `buildAndValidate`, before `validateRepoNames` and all other validators. If `cfg.V1WorkgroupCount > 0`, return immediately with the migration error. A v1 config should not produce noise from other validations.
- **D-05:** Fail-fast: return immediately on first detection. No accumulation of additional errors.

### Error message

- **D-06:** Error message format (lowercase, no trailing period, actionable):
  `"v1 config detected: found %d [[workgroup]] block(s) — run 'git w migrate' to upgrade"`
- **D-07:** The error is returned from `buildAndValidate`, which propagates through `loadMainConfig` → `Load`. Callers see a standard `error` return; no special error type needed for Phase 10.

### the agent's Discretion

- Exact field name on `diskConfig` for the detection list (e.g. `WorkgroupList`).
- Whether `detectV1Workgroups` is a private function called from `buildAndValidate` or inline guard clause.
- Test fixture naming and structure.

</decisions>

<specifics>
## Specific Ideas

- The `V1WorkgroupCount` field on `WorkspaceConfig` is detection-only state — not saved, not serialized, not visible in `prepareDiskConfig`. It belongs alongside `Warnings` as an in-memory-only field.
- Phase 6 used a targeted raw TOML pass for strict-key workstream checks. Phase 10 takes a simpler path by adding a field to `diskConfig` instead — consistent with how all other v2 block types are handled.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1 detection spec
- `.planning/v2/v2-schema.md` — load-time detection of v1 `[[workgroup]]` blocks (Implementation notes section)
- `.planning/v2/v2-migration.md` — `[[workgroup]]` detection triggers, `DetectV1` scope context; confirms detection-only intent (no migration logic in config loader)

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-10 requirement definition and phase mapping

### Codebase integration
- `pkg/config/loader.go` — `diskConfig` struct (lines ~620-629), `loadMainConfig` function (lines ~37-72), `buildAndValidate` function (lines ~80-119); all three are modified in this phase
- `pkg/config/config.go` — `WorkspaceConfig` struct (lines ~12-24); `V1WorkgroupCount int` field added alongside `Warnings []string`

### Prior context decisions to carry forward
- `.planning/phases/03-enforce-repos-n-path-convention/03-CONTEXT.md` — non-conforming paths produce warnings (load succeeds); Phase 10 produces a hard error (load fails) — contrast is intentional per spec
- `.planning/phases/06-add-workstream-root-config-block/06-CONTEXT.md` — strict-key workstream check used raw TOML pass; Phase 10 uses diskConfig field instead (simpler approach chosen)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `diskConfig` struct in `loader.go` — all v2 block types follow the `[]TypeConfig` + `toml:"key,omitempty"` pattern; Phase 10 adds `[]map[string]any` with the same pattern
- `buildAndValidate` in `loader.go` — ordered list of validation calls; `detectV1Workgroups` becomes the first entry
- `Warnings []string` on `WorkspaceConfig` — precedent for in-memory-only fields; `V1WorkgroupCount int` follows the same pattern

### Established Patterns
- All validators are private functions called from `buildAndValidate`; `detectV1Workgroups` follows the same shape
- Error messages: lowercase, no trailing period, `%q` or `%d` for values, actionable suggestion at the end
- Data flow: `diskConfig` → `WorkspaceConfig` fields set in `loadMainConfig` → read in `buildAndValidate`

### Integration Points
- `loadMainConfig`: set `cfg.V1WorkgroupCount = len(dc.WorkgroupList)` after `ensureWorkspaceMaps(cfg)`
- `buildAndValidate`: add `detectV1Workgroups` as the first call before `validateRepoNames`
- `WorkspaceConfig`: add `V1WorkgroupCount int` field; keep comment `// in-memory only`
- `diskConfig`: add `WorkgroupList []map[string]any` field tagged `toml:"workgroup,omitempty"`
- No changes to `prepareDiskConfig` — this field is detection-only and not written back

</code_context>

<deferred>
## Deferred Ideas

- `DetectV1` function in `pkg/migrate` that also scans workgroup blocks — that is Phase 59 (M12), not this phase
- Surfacing V1WorkgroupCount in any user-facing status output — no need; the error is the only output
- Warning (non-fatal) mode for v1 workgroup detection — spec is clear this is a hard error; not a candidate for soft degradation

</deferred>

---

*Phase: 10-detect-v1-workgroup-blocks*
*Context gathered: 2026-04-07*
