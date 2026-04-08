# Phase 7: Two-file config merge - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-06
**Phase:** 07-two-file-config-merge
**Areas discussed:** Validation sequencing, MergeRepo + MergeWorkspace, Absent .git/.gitw, Load entrypoint shape

---

## Validation sequencing

| Option | Description | Selected |
|--------|-------------|----------|
| Parse both, validate merged | Parse each file individually (TOML only), run full validation on the merged result. Cross-reference checks resolve against the merged config. | ✓ |
| Validate each independently | Run full validation on each file in isolation before merging. Stricter but breaks the intentional split pattern. | |

**User's choice:** Parse both, validate merged
**Notes:** The v2-schema.md annotated example shows workstreams in `.git/.gitw` referencing remotes from `.gitw` — this only works if validation runs on the merged result.

---

## MergeRepo semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Field-level override + new entries | Private file can override fields on existing repos and add entirely new repos. | |
| Override only, no new repos | Private file can only override fields on repos already in .gitw. Net-new repos from .git/.gitw are an error. | ✓ |
| Repos are .gitw-only | Private file cannot touch [[repo]] at all. | |

**User's choice:** Override only, no new repos
**Notes:** Field-level merge confirmed in follow-up — same pattern as `MergeRemote` (non-zero private value wins per field). An unknown repo name in `.git/.gitw` is a load-time error.

---

## MergeWorkspace semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Workspaces are .gitw-only | No MergeWorkspace needed; private file cannot touch [[workspace]]. | |
| Override only, no new workspaces | Private file can override fields on workspaces already in .gitw. | |
| Full override + new entries | Private file can add new workspace blocks or field-override existing ones. | ✓ |

**User's choice:** Full override + new entries
**Notes:** Workspaces are treated more permissively than repos — the private file can introduce new workspaces (e.g. personal groupings) or override shared workspace metadata.

---

## Absent .git/.gitw

| Option | Description | Selected |
|--------|-------------|----------|
| Silent skip | Missing .git/.gitw ignored with no warning. Consistent with mergeLocalConfig behavior today. | ✓ (base behavior) |
| Trace on absence | Emit trace when file is absent. | ✓ (conditional) |
| Error if absent | Require file to exist. | |

**User's choice:** Trace on absence (debug flag only)
**Notes:** Base behavior is silent skip. A trace line emitted to stderr only when `--debug` flag is present on the root command. If `--debug` does not yet exist, the trace is deferred — silent skip is sufficient for Phase 7.

---

## Load entrypoint shape

| Option | Description | Selected |
|--------|-------------|----------|
| Extend Load() transparently | Load() grows a third merge layer internally. All callers unchanged. | ✓ |
| New LoadWithPrivate() entry | New public function; existing Load() unchanged; LoadCWD() updated to call it. | |
| Post-load MergePrivate() | Separate function called after Load(); maximum composability. | |

**User's choice:** Extend Load() transparently
**Notes:** Private config path derived from main config path: `filepath.Join(filepath.Dir(cfgPath), ".git", ".gitw")`. No caller changes needed.

---

## the agent's Discretion

- Error message wording for unknown repo name in `.git/.gitw` (follow existing lowercase/quoted style)
- Ordering of private merge step relative to `.gitw.local` merge step within `Load()`

## Deferred Ideas

None.
