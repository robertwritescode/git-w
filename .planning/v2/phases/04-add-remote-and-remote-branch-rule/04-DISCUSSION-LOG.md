# Phase 4: Add `[[remote]]` and `[[remote.branch_rule]]` - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-04
**Phase:** 04-add-remote-and-remote-branch-rule
**Areas discussed:** Struct design and disk/in-memory split, `[[remote.branch_rule]]` nesting and ordering, `remotes` field on `RepoConfig`, Validation scope and error granularity

---

## Struct design and disk/in-memory split

| Option | Description | Selected |
|--------|-------------|----------|
| `[]RemoteConfig` directly on `WorkspaceConfig` | No map, no diskConfig split; TOML key `"remote"` | ✓ |
| Map keyed by name (like v1 repos) | `map[string]RemoteConfig`; would need diskConfig split | |
| `diskConfig` split with migration | Separate on-disk and in-memory types | |
| Define `MergeRemote` in Phase 4 | Pure function alongside struct; Phase 7 calls it | ✓ |
| Defer `MergeRemote` to Phase 7 | Define only when first used | |
| `BranchRuleConfig` in `pkg/config` | Schema layer; `pkg/brules` imports from here | ✓ |
| `BranchRuleConfig` in `pkg/brules` | Evaluation layer owns the type | |

**User's choices:** `[]RemoteConfig` directly on `WorkspaceConfig`; define `MergeRemote` in Phase 4; `BranchRuleConfig` lives in `pkg/config`.

---

## `[[remote.branch_rule]]` nesting and ordering

| Option | Description | Selected |
|--------|-------------|----------|
| `BranchRules []BranchRuleConfig` on `RemoteConfig` | TOML array-of-tables nested inside array-of-tables | ✓ |
| Flat `[[branch_rule]]` at workspace level | All rules at root; referenced by remote name | |
| Agent's discretion | Let agent pick | |
| `*bool` for optional criteria fields | `nil` = not set (distinct from false) | ✓ |
| `bool` with separate "set" flag | More verbose; two fields per criterion | |
| Agent's discretion | Let agent pick | |
| `type BranchAction string` with named constants | Typed alias; constants defined in Phase 4 | ✓ |
| Plain `string` for action field | No type safety; Phase 12/13 defines type | |
| Agent's discretion | Let agent pick | |

**User's choices:** `BranchRules []BranchRuleConfig` on `RemoteConfig`; `*bool` for optional criteria fields; `type BranchAction string` with constants in Phase 4.

---

## `remotes` field on `RepoConfig`

| Option | Description | Selected |
|--------|-------------|----------|
| Add `Remotes []string` in Phase 4 | Field exists for parsing; consumed in later phases | ✓ |
| Add only when first consumed (Phase 9/16) | Defer field addition | |
| Agent's discretion | Let agent pick | |

**User's choices:** Add `Remotes []string \`toml:"remotes,omitempty"\`` to `RepoConfig` in Phase 4.

---

## Validation scope and error granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Validate name presence + uniqueness + kind + action enums | Four checks in Phase 4 | ✓ |
| Validate only name and uniqueness | Defer enum checks | |
| Include cross-field rules (`flag` required for `require-flag`) | More complete validation | |
| Defer cross-field rules to Phase 12/13 | Only structural validation in Phase 4 | ✓ |
| Agent's discretion | Let agent pick | |
| `private=true` enforcement in Phase 4 | Check file path in `buildAndValidate` | ✓ |
| Defer `private` check to Phase 7/9 | Validate when private remotes are consumed | |
| Hard error for `private` in `.gitw` | Violation returns error, not warning | ✓ |
| Warning for `private` in `.gitw` | Softer signal | |

**User's choices:** Four validation checks (name presence, uniqueness, kind enum, action enum); cross-field rules deferred to Phase 12/13; `private=true` enforcement in Phase 4 as a hard error.

---

## Agent's Discretion

- Ordering of validation checks within `buildAndValidate`
- Whether remote validation uses a standalone `validateRemotes` function or multiple extracted helpers
- Field ordering on `RemoteConfig` and `BranchRuleConfig` structs
- Test table structure for validation error cases

## Deferred Ideas

- Cross-field validation (`flag` required when `action=require-flag`) — Phase 12/13
- `MergeRemote` call sites and cascade resolution — Phase 7
- `RepoConfig.Remotes` consumption — Phase 9 (cascade) / Phase 16 (sync)
- Branch rule evaluation (`EvaluateRule`) — Phase 12/13
- `[[sync_pair]]` parsing — Phase 5
