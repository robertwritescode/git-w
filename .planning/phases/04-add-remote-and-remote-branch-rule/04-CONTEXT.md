# Phase 4: Add `[[remote]]` and `[[remote.branch_rule]]` - Context

**Gathered:** 2026-04-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Parse `[[remote]]` and `[[remote.branch_rule]]` blocks from `.gitw`. Users can define remote configurations (URL, kind, push behavior, credentials) with per-remote branch-level push rules. Validation errors on invalid remote/rule configs. Evaluation and cascade resolution of these rules are separate phases.

Delivers: CFG-04 (`[[remote]]` and `[[remote.branch_rule]]` parsing and validation).

Out of scope: rule evaluation (Phase 12/13), cascade resolution (Phase 7/9), `[[sync_pair]]` (Phase 5), any command that consumes remote config beyond loading and validating.

</domain>

<decisions>
## Implementation Decisions

### Struct design and disk/in-memory split

- **D-01:** `[]RemoteConfig` lives directly on `WorkspaceConfig` with TOML key `"remote"` — no map, no `diskConfig` split. Matches the `WorkspaceBlock` pattern established in Phase 1 (array-of-tables, declaration order preserved).
- **D-02:** `MergeRemote(base, override RemoteConfig) RemoteConfig` is defined in Phase 4 alongside the struct as a pure function. Phase 7 (cascade) will call it; defining it now makes the Phase 7 task trivial and keeps the merge contract with the schema.
- **D-03:** `BranchRuleConfig` lives in `pkg/config` (schema layer). A future `pkg/brules` package (Phases 12/13) will import `BranchRuleConfig` from `pkg/config` for evaluation. No circular dependency.

### `[[remote.branch_rule]]` nesting and ordering

- **D-04:** `RemoteConfig` has `BranchRules []BranchRuleConfig \`toml:"branch_rule"\`` — TOML array-of-tables nested inside array-of-tables. `go-toml/v2` handles this natively; declaration order is preserved.
- **D-05:** Optional boolean criteria fields (`Untracked`, `Explicit`) on `BranchRuleConfig` use `*bool` pointer types. `nil` means "not set" (distinct from `false`), which matters for criteria matching semantics in Phase 13.
- **D-06:** `Action` field uses a typed string alias: `type BranchAction string` defined in Phase 4, with named constants `ActionAllow`, `ActionBlock`, `ActionWarn`, `ActionRequireFlag`. Phase 12/13's `EvaluateRule` uses this same type — no redefinition needed.

### `remotes` field on `RepoConfig`

- **D-07:** `Remotes []string \`toml:"remotes,omitempty"\`` is added to `RepoConfig` in Phase 4. The field exists for parsing; nothing consumes it until Phase 9 (cascade) or Phase 16 (sync). `omitempty` ensures existing configs that omit the field are unaffected.

### Validation scope and error granularity

- **D-08:** Phase 4 validates four things in `buildAndValidate`:
  1. Each `[[remote]]` has a non-empty `name`.
  2. `name` values are unique across all `[[remote]]` blocks.
  3. `kind` is one of the valid enum values: `gitea`, `forgejo`, `github`, `generic`.
  4. Each `[[remote.branch_rule]]` `action` is one of: `allow`, `block`, `warn`, `require-flag`.
  Cross-field rules (e.g. `flag` required when `action=require-flag`) are NOT in scope for Phase 4.
- **D-09:** `private=true` enforcement IS in Phase 4. A `[[remote]]` marked `private = true` must live in `.git/.gitw`, not `.gitw`. This check lives in `buildAndValidate` where the config file path is known. Violation returns an error (not a warning).

### Agent's Discretion

- Exact ordering of validation checks within `buildAndValidate` (name presence, uniqueness, kind, action, private enforcement)
- Whether remote validation is a standalone `validateRemotes` function or multiple extracted helpers
- Field ordering on `RemoteConfig` and `BranchRuleConfig` structs
- Test table structure for validation error cases

</decisions>

<specifics>
## Specific Ideas

- `MergeRemote` should be a pure function with no side effects — Phase 7 expects to call it without touching config state.
- The `private` check uses the config file path already available in `buildAndValidate` — check if the path ends in `.git/.gitw`, not `.gitw`.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### `[[remote]]` schema and fields
- `.planning/v2/v2-schema.md` — `[[remote]]` block definition with all field specs (`name`, `url`, `kind`, `token_env`, `repo_prefix`, `repo_suffix`, `push_mode`, `critical`, `private`); `[[remote.branch_rule]]` sub-block fields (`pattern`, `action`, criteria fields); full annotated config example

### Remote management and branch rules
- `.planning/v2/v2-remote-management.md` — Remote management motivation; branch rule evaluation order; `BranchRuleConfig` implementation notes; `MergeRemote` spec; `private` field enforcement rationale

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-04: `[[remote]]` and `[[remote.branch_rule]]` parsing requirement

### Prior phase context (carry-forward decisions)
- `.planning/phases/01-add-workspace-block/1-CONTEXT.md` — `buildAndValidate` is the single validation integration point; array-of-tables pattern for new top-level blocks
- `.planning/phases/02-add-track-branch-and-upstream-fields/02-CONTEXT.md` — `*bool` pointer pattern for optional bools; `RepoConfig` field additions
- `.planning/phases/03-enforce-repos-n-path-convention/03-CONTEXT.md` — `Warnings []string` on `WorkspaceConfig`; warning-vs-error distinction; `buildAndValidate` call chain

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/config/loader.go` `buildAndValidate`: The single validation integration point — all new remote validation wires here. Existing `validateRepoPaths` shows the pattern for a dedicated validation helper called from `buildAndValidate`.
- `pkg/config/config.go` `MetarepoConfig`: Uses `*bool` for optional bool fields (`PushProtection`, `RequireCleanBranch`) — carry this pattern to `BranchRuleConfig.Untracked` and `BranchRuleConfig.Explicit`.
- `pkg/config/config.go` `WorkspaceConfig`: Add `Remotes []RemoteConfig \`toml:"remote"\`` here, alongside `Repos`, `Groups`, `Worktrees`, `Workgroups`.

### Established Patterns
- Array-of-tables on `WorkspaceConfig` (e.g. `Repos []RepoConfig`) — `Remotes []RemoteConfig` follows the same pattern, no `diskConfig` wrapper needed.
- `buildAndValidate` returns `error` only; soft checks use `cfg.Warnings` (Phase 3); hard checks (including the `private` enforcement) return `error`.
- `output.Writef(cmd.ErrOrStderr(), ...)` for stderr output in command handlers — no changes needed here since Phase 4 adds no new commands.

### Integration Points
- `pkg/config/config.go` `WorkspaceConfig` struct — add `Remotes []RemoteConfig` and new `RemoteConfig` / `BranchRuleConfig` / `BranchAction` type definitions.
- `pkg/config/config.go` `RepoConfig` struct — add `Remotes []string \`toml:"remotes,omitempty"\``.
- `pkg/config/loader.go` `buildAndValidate` — wire `validateRemotes(cfg, cfgPath)` (or equivalent) call.

</code_context>

<deferred>
## Deferred Ideas

- Cross-field validation (`flag` required when `action=require-flag`) — Phase 12/13 (rule evaluation)
- `MergeRemote` call sites and cascade resolution — Phase 7
- `RepoConfig.Remotes` consumption for cascade — Phase 9
- Branch rule evaluation (`EvaluateRule`) — Phase 12/13
- `[[sync_pair]]` parsing — Phase 5

</deferred>

---

*Phase: 04-add-remote-and-remote-branch-rule*
*Context gathered: 2026-04-04*
