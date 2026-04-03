# Phase 3: Enforce `repos/<n>` Path Convention - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-03
**Phase:** 3-enforce-repos-n-path-convention
**Areas discussed:** Warning output mechanism, Convention definition, Warning message content, Scope of path check

---

## Warning output mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Return warnings alongside cfg | Load returns (cfg, []string, error) | |
| Pass io.Writer into Load | config.Load accepts optional io.Writer for warnings | |
| Package-level warnings collector | Global collector drained after Load() | |
| Warnings field on WorkspaceConfig | cfg.Warnings []string; Load signature stays (cfg, error) | ✓ |
| config.Load returns (cfg, []string, error) | Simple slice return | |
| Diagnostics type | New type wrapping warnings and errors | |
| Print in LoadConfig helper (one place) | Shared helper prints after load | ✓ |
| Each RunE handler prints warnings | ~30 handlers need updating | |
| Only print in mutating commands | Asymmetric behavior | |
| Extend existing LoadConfig (with cmd arg) | LoadConfig gains warning-print responsibility | ✓ |
| Separate PrintWarnings helper | Separate helper callers must remember to call | |
| Agent's discretion | Let agent pick | |

**User's choices:** `Warnings []string` field on `WorkspaceConfig`; print warnings in the existing `LoadConfig` helper (one place); extend `LoadConfig` to receive `cmd` for `cmd.ErrOrStderr()` access.

---

## Convention definition

| Option | Description | Selected |
|--------|-------------|----------|
| Exactly repos/<single-segment> | Two segments: `repos` + one non-empty name, no / in name | ✓ |
| Starts with repos/ (any depth) | Any depth under repos/ is conforming | |
| Configurable pattern | User-configurable regex | |
| Normalize then check | filepath.Clean before checking | ✓ |
| Check raw path (no normalization) | Check as-is | |
| Agent's discretion | Let agent pick | |

**User's choices:** Exactly `repos/<single-segment>`; normalize with `filepath.Clean` first.

---

## Warning message content

| Option | Description | Selected |
|--------|-------------|----------|
| One warning per repo | One line per non-conforming repo with suggested path | ✓ |
| Single summary with all repos listed | One summary message for all | |
| Agent's discretion | Let agent pick | |
| Include suggested path in warning | Show `repos/<basename>` suggestion in warning text | ✓ |
| Flag only, no suggestion | Just flag the non-conforming path | |
| Agent's discretion | Let agent pick | |

**User's choices:** One warning per repo; include the suggested `repos/<basename>` path in the warning.

---

## Scope of path check

| Option | Description | Selected |
|--------|-------------|----------|
| All repos including aliases | Check all [[repo]] entries regardless of track_branch/upstream | ✓ |
| Non-alias repos only | Skip alias repos | |
| Agent's discretion | Let agent pick | |
| [[repo]] path only, skip bare_path | Phase 3 scope is [[repo]] path fields only | ✓ |
| Include bare_path entries too | Check worktree bare_path too | |
| Agent's discretion | Let agent pick | |

**User's choices:** Check all repos including aliases; skip `bare_path` entries on worktree sets.

---

## Agent's Discretion

- Exact field placement of `Warnings` on `WorkspaceConfig`
- Whether warning logic is a standalone function or inline in `buildAndValidate`
- Test case naming and table structure

## Deferred Ideas

- Path migration / `mv` on disk — Phase 61
- Collision detection — Phase 59
- `git w migrate` command — Phase 62
- Bare repo detection — Phase 59
