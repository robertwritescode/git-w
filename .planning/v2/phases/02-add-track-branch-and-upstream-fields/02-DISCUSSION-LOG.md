# Phase 2: Add `track_branch` and `upstream` Fields - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-03
**Phase:** 02-add-track-branch-and-upstream-fields
**Areas discussed:** Alias recognition, Validation rules, RepoConfig vs array-of-tables, Field naming alignment

---

## Alias recognition

| Option | Description | Selected |
|--------|-------------|----------|
| Helper method (IsAlias) | Add `IsAlias() bool` method on `RepoConfig` (returns true when `track_branch != ""`). Callers decide what to do. | |
| Deferred to callers | No recognition at load time — callers check `TrackBranch != ""` directly. Recognition happens when the fields are consumed (Phases 17, 43, 44). | |
| Load-time validation only | Validate at load time that `upstream` is not empty when `track_branch` is set, and that upstream names are internally consistent. | ✓ |

**User's choice:** Load-time validation where `upstream` is validated not-empty when `track_branch` is set and upstream names are internally consistent is important. `IsAlias` may or may not be needed — agent's discretion.

---

## Validation rules

### upstream without track_branch

| Option | Description | Selected |
|--------|-------------|----------|
| Error: upstream requires track_branch | `upstream` without `track_branch` is never valid — aliases must declare their target branch. | ✓ |
| Warn: suspicious but allowed | `upstream` without `track_branch` is suspicious but not an error — let it through with a warning. | |
| Allow silently | Treat `upstream` without `track_branch` as pure display metadata even without a tracked branch. | |

**User's choice:** Error: upstream requires track_branch.

### Duplicate track_branch per upstream group

| Option | Description | Selected |
|--------|-------------|----------|
| Error: duplicate track_branch per upstream | Reject configs where two repos share an upstream but use the same `track_branch` value. | ✓ |
| Warn: duplicates suspicious | Flag duplicate `track_branch` values per upstream group but proceed. | |
| Allow: no uniqueness check now | No uniqueness validation within an upstream group at this phase. | |

**User's choice:** Error: duplicate track_branch per upstream.

### upstream as reference vs. label

| Option | Description | Selected |
|--------|-------------|----------|
| Error: upstream name must match a repo | Error if `upstream = "infra"` but no `[[repo]]` with name "infra" exists. | — |
| No: it's a free label (not a repo reference) | `upstream` is a grouping label. It doesn't need to match any `[[repo]]` name. | ✓ |
| Warn: suggest it should match | Warn if upstream doesn't match a repo name, but proceed. | |

**User's choice:** Initially chose "Error: must match a repo" but upon clarification that `upstream = "infra"` does NOT correspond to a `[[repo]]` entry (it's derived from URL basename), confirmed: upstream is a free-form label. No reference validation needed.

### Clone URL for alias repos

| Option | Description | Selected |
|--------|-------------|----------|
| URL on each alias repo independently | Each alias independently specifies the URL. `upstream` is display only. | |
| Infer from upstream group | Find canonical URL by looking at all repos sharing the same upstream. | |
| Defer to Phase 42 | Phase 2 only adds fields; URL handling is Phase 42 (repo add). | ✓ |

**User's choice:** Defer to Phase 42.

---

## RepoConfig vs `[[repo]]` array-of-tables

| Option | Description | Selected |
|--------|-------------|----------|
| Keep map[string]RepoConfig now | Smallest change; the map is functional and tested. | |
| Migrate to []RepoConfig array-of-tables now | One cohesive structural change alongside adding the new fields. | ✓ |
| I'm not sure | Asked follow-up question. | — |

**Follow-up: when should migration happen?**

| Option | Description | Selected |
|--------|-------------|----------|
| Migrate now: Phase 2 adds fields + migrates structure | `track_branch` and `upstream` require the `name` field; doing it later means two breaking changes. | ✓ |
| Defer: keep map, add fields only | Keep `map[string]RepoConfig`, add two new string fields, defer struct migration. | |
| You decide | Agent's discretion. | |

**User's choice:** Migrate now — bundle both changes.

**Follow-up: name field handling**

| Option | Description | Selected |
|--------|-------------|----------|
| Required `name` field on every `[[repo]]` | Map key eliminated; `name` field is the primary key. | ✓ |
| Parse as array, store as map internally | Parse `[[repo]]`, then build `map[string]RepoConfig` in loader. | |
| Dual-format support | Support both old map + new array for compatibility. | |

**User's choice:** Required `name` field on every `[[repo]]`.

---

## Field naming alignment

| Option | Description | Selected |
|--------|-------------|----------|
| Rename URL → clone_url in Phase 2 | Rename alongside struct migration — one cohesive v2 alignment. | ✓ |
| Defer: keep URL field as-is | Rename in a later phase. | |
| Agent's discretion | Pick whichever keeps the code cleaner. | |

**User's choice:** Rename `URL` → `CloneURL` (TOML: `clone_url`) in Phase 2.

---

## the agent's Discretion

- Whether `IsAlias()` helper method is added to `RepoConfig`
- Exact error message wording for validation failures
- Internal helper design for `RepoByName` (method, free function, or map rebuild)
- Field ordering within the updated `RepoConfig` struct

## Deferred Ideas

- Sync behavior using `track_branch` as pull target — Phase 17
- `ResolveEnvGroup` / `--env-group` expansion — Phase 43
- `git w repo list --upstream` and `git w status --repo <upstream>` — Phase 44
- `git w repo add --branch` / `--branch-map` for creating aliases — Phase 42
- Clone URL resolution for alias repos — Phase 42
