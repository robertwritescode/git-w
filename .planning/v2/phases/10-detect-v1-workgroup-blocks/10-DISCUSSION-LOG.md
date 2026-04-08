# Phase 10: Detect v1 `[[workgroup]]` blocks - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 10-detect-v1-workgroup-blocks
**Areas discussed:** Detection approach, Pipeline placement and fail behavior

---

## Detection approach

| Option | Description | Selected |
|--------|-------------|----------|
| Add detection field to diskConfig | Add `[]map[string]any` field tagged `toml:"workgroup"` to diskConfig. If len > 0 after unmarshal, it's a v1 config. Uses the same pipeline as all other detections. | ✓ |
| Targeted raw TOML pass | Unmarshal raw bytes a second time into a minimal struct with only the workgroup key, similar to Phase 6 strict-key workstream check. | |
| String scan on raw bytes | Scan raw bytes for `[[workgroup]]` before parsing. Fast but fragile — false positives from comments or quoted strings. | |

**User's choice:** Add detection field to diskConfig

---

**Type for the detection field:**

| Option | Description | Selected |
|--------|-------------|----------|
| Named struct (v1WorkgroupConfig) | Named Go struct with v1 workgroup fields. Type-safe but requires knowing v1 field names. | |
| Generic `[]map[string]any` | Captures any workgroup entry without needing to know v1 field names. Detection only needs presence, not values. | ✓ |

**User's choice:** `[]map[string]any`

---

**Where the check fires:**

| Option | Description | Selected |
|--------|-------------|----------|
| Inside buildAndValidate | diskConfig data threaded through WorkspaceConfig into buildAndValidate. | ✓ |
| In loadMainConfig before buildAndValidate | Check `len(dc.WorkgroupList) > 0` directly in loadMainConfig after unmarshal. Simple, no refactor needed. | |
| Via WorkspaceConfig field in buildAndValidate | Add V1WorkgroupCount field to WorkspaceConfig, detect inside buildAndValidate. | |

**User's choice:** Inside buildAndValidate (via WorkspaceConfig field)

---

**How to thread the data:**

| Option | Description | Selected |
|--------|-------------|----------|
| Thread count through WorkspaceConfig | Set count on WorkspaceConfig in loadMainConfig; buildAndValidate reads cfg.V1WorkgroupCount. | ✓ |
| Pass diskConfig to buildAndValidate | Pass full diskConfig as additional parameter. More flexible but changes a widely-used function signature. | |

**User's choice:** Thread count through WorkspaceConfig

---

## Pipeline placement and fail behavior

| Option | Description | Selected |
|--------|-------------|----------|
| First check, return immediately | detectV1Workgroups at top of buildAndValidate before all other checks. Return immediately with migration error. | ✓ |
| Last check | Near the end after all other validations. User sees all errors at once but also noisy v1 config errors. | |
| Separate pre-validation pass | Separate detectV1 pass before buildAndValidate. Explicit but adds a separate call site to maintain. | |

**User's choice:** First check, return immediately

---

**Count vs bool for WorkspaceConfig field:**

| Option | Description | Selected |
|--------|-------------|----------|
| bool (HasV1Workgroups) | Simpler type; detection only needs presence. | |
| int (V1WorkgroupCount) | Useful if error message says "found N [[workgroup]] blocks" | ✓ |

**User's choice:** int (V1WorkgroupCount) — allows error message to include count

---

## the agent's Discretion

- Exact field name on diskConfig for the detection list
- Whether detectV1Workgroups is a private function or inline guard
- Test fixture naming and structure

## Deferred Ideas

- `DetectV1` in `pkg/migrate` (Phase 59)
- V1WorkgroupCount in user-facing status output
- Warning/non-fatal mode for v1 workgroup detection
