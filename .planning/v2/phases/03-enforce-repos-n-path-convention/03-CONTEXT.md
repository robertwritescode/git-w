# Phase 3: Enforce `repos/<n>` Path Convention - Context

**Gathered:** 2026-04-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Tool warns at load time when repos use v1-style paths and suggests running `git w migrate`. Non-conforming paths produce per-repo warnings to stderr but do not prevent config loading.

Delivers: CFG-03 (`repos/<n>` path convention enforcement with v1 warning).

Out of scope: actual path migration (Phase 59-63), `git w migrate` command, moving directories on disk, any command-level changes beyond surfacing warnings.

</domain>

<decisions>
## Implementation Decisions

### Warning output mechanism

- **D-01:** `WorkspaceConfig` gains a `Warnings []string` field. `buildAndValidate` appends path-convention warnings into it during load. `config.Load` signature stays `(*WorkspaceConfig, error)` — callers access `cfg.Warnings` after load.
- **D-02:** `config.LoadConfig` (the cobra-layer wrapper used by all domain command RunE handlers) gains responsibility for printing warnings. It prints each warning to `cmd.ErrOrStderr()` via `output.Writef`. No changes required to individual RunE handlers.
- **D-03:** `config.LoadConfig` receives `cmd *cobra.Command` (already does in most domain packages — confirm signature and update where needed).

### Convention definition

- **D-04:** A conforming path is exactly two clean path segments where the first is `repos` and the second is a single non-empty name with no `/` in it. `repos/my-repo` = conforming. `repos/org/my-repo` = non-conforming. Any depth beyond one segment fails.
- **D-05:** Path is normalized with `filepath.Clean` before checking (strips `./`, trailing slashes, collapses `//`). `./repos/x` and `repos/x/` both normalize to `repos/x` and are treated as conforming.

### Warning message content

- **D-06:** One warning string per non-conforming repo. Format:
  `warning: repo "<name>" path "<path>" does not follow repos/<n> convention; suggested: "repos/<basename>"; run 'git w migrate' to update`
  where `<basename>` is `filepath.Base(path)`.
- **D-07:** Suggested path is always `repos/<basename>` — the basename of the existing path. This matches what `git w migrate` would produce.

### Scope of path check

- **D-08:** Check applies to all `[[repo]]` entries (all repos, including aliases with `track_branch`/`upstream` set). No special-casing for alias repos.
- **D-09:** `bare_path` entries on worktree sets (`WorktreeConfig`) are NOT checked. Phase 3 scope is `[[repo]] path` fields only. Worktree bare paths are a separate concern already validated by `validateWorktreePaths`.

### Agent's Discretion

- Exact field placement of `Warnings` on `WorkspaceConfig` (before or after existing fields)
- Whether `warnNonConformingRepoPaths` is a standalone function or folded inline into `buildAndValidate`
- Test case naming and table structure for the new warning logic

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v2 config schema
- `.planning/v2/v2-schema.md` — `[[repo]]` block definition; `path = "repos/<n>"` convention; annotated full config example showing correct paths throughout

### v2 migration
- `.planning/v2/v2-migration.md` — Detection triggers (non-`repos/<n>` paths); migration plan cases; `repos/<basename>` as the canonical suggested target; confirms warning-not-error semantics

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-03: enforce `repos/<n>` path convention and warn on v1 paths with migration suggestion

### Prior phase context
- `.planning/phases/01-add-workspace-block/1-CONTEXT.md` — `buildAndValidate` is the validation integration point; all validation wires through here
- `.planning/phases/02-add-track-branch-and-upstream-fields/02-CONTEXT.md` — `[[repo]]` array-of-tables migration; `RepoConfig` now has `Path` field on each entry; `buildAndValidate` call chain established

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/config/loader.go` `buildAndValidate`: Integration point — path-convention warning logic wires in here, after `validateRepoPaths` (which already validates non-empty + relative). Warning populates `cfg.Warnings`, not returns an error.
- `pkg/config/loader.go` `validateRepoPaths` + `validateRepoPath`: Already validates empty and absolute paths (errors). Phase 3 adds a softer check on top: path is valid but does not match `repos/<n>` convention → warning.
- `pkg/config/loader.go` `ResolveRepoPath`: Uses `filepath.Clean` + `filepath.IsAbs` internally. The same normalization (`filepath.Clean`) should be applied before the convention check.
- `config.LoadConfig` in domain packages: The existing cobra-layer wrapper that all RunE handlers call. Needs to print `cfg.Warnings` to `cmd.ErrOrStderr()` after load.

### Established Patterns
- `buildAndValidate` returns `error` only. Phase 3 changes the flow slightly: path-convention check appends to `cfg.Warnings` (no error return) while all existing checks continue to return errors.
- `output.Writef(cmd.ErrOrStderr(), ...)` is the established pattern for warnings/stderr output in command handlers.
- `filepath.Clean` is already used in config path handling — use it consistently for normalization.

### Integration Points
- `WorkspaceConfig.Warnings []string` — new field; populated during `buildAndValidate`, consumed by `config.LoadConfig` in the cobra layer.
- All domain command packages use `config.LoadConfig(cmd)` (or equivalent) — warning print happens there, no RunE handler changes needed.
- `pkg/config/config.go` `WorkspaceConfig` struct — add `Warnings []string` field.

</code_context>

<specifics>
## Specific Ideas

- Warning fires at load time, not at save time. `validateRepoPaths` (called during `Save`) does NOT get the convention check — save-time path validation is a separate pass. Only `buildAndValidate` (load path) gets the warning.
- The suggested path in the warning (`repos/<basename>`) is intentionally simple. If `basename` would collide with another repo name, that is migrate's problem to detect and report, not the load-time warning's.

</specifics>

<deferred>
## Deferred Ideas

- Actual path migration / `mv` on disk — Phase 61 (`ApplyPlan`)
- Collision detection between suggested paths — Phase 59 (`DetectV1`)
- `git w migrate` command — Phase 62
- Bare repo detection — Phase 59

</deferred>

---

*Phase: 03-enforce-repos-n-path-convention*
*Context gathered: 2026-04-03*
