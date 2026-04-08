# Phase 2 Research: Add `track_branch` and `upstream` Fields

**Phase:** 02 — add-track-branch-and-upstream-fields
**Researched:** 2026-04-03
**Requirements:** CFG-02

---

## Summary

Phase 2 bundles three cohesive changes into one breaking migration pass:

1. **`[[repo]]` array-of-tables migration** — `map[string]RepoConfig` (TOML: `[repos.<n>]`) → `[]RepoConfig` (TOML: `[[repo]]`, required `name` field).
2. **`URL` → `CloneURL` rename** — TOML key `url` → `clone_url` on `RepoConfig`.
3. **New fields** — `TrackBranch string` (`toml:"track_branch"`) and `Upstream string` (`toml:"upstream"`) on `RepoConfig`.

All three are intentionally a single breaking change (precedent from Phase 1's `[workspace]` → `[metarepo]` rename).

---

## Key Findings

### 1. Migration Scope: `cfg.Repos` Map Accesses

The in-memory `WorkspaceConfig.Repos` field (`map[string]RepoConfig`) is accessed in **11 non-test source files** across 4 packages. These must all be updated to use `RepoByName(name string) (RepoConfig, bool)` or the slice iteration equivalent after migration.

**Files with direct `cfg.Repos[name]` map lookups (non-test):**

| File | Access Pattern |
|------|---------------|
| `pkg/config/config.go` | `c.Repos[repoName]` (ResolveDefaultBranch), `c.Repos[name]` (RepoName) |
| `pkg/config/loader.go` | `cfg.Repos[repoName]` (synthesize conflict check + assign), `cfg.Repos` range |
| `pkg/repo/repo.go` | `cfg.Repos[name]` (FromNames), `config.SortedStringKeys(cfg.Repos)` (FromConfig) |
| `pkg/repo/filter.go` | `cfg.Repos[member]` (group membership check) |
| `pkg/repo/add.go` | `cfg.Repos[name] = ...` (write) |
| `pkg/repo/clone.go` | `cfg.Repos[name] = ...` (write) |
| `pkg/repo/list.go` | `cfg.Repos[name]` (lookup), `SortedStringKeys(cfg.Repos)` (range) |
| `pkg/repo/rename.go` | `cfg.Repos[oldName]`, `delete(cfg.Repos, oldName)`, `cfg.Repos[newName] = ...` |
| `pkg/repo/unlink.go` | `cfg.Repos[name]` (exists check), `delete(cfg.Repos, name)` |
| `pkg/repo/restore.go` | `SortedStringKeys(cfg.Repos)`, `cfg.Repos[name]` |
| `pkg/workspace/group.go` | `cfg.Repos[name]` (membership validation) |
| `pkg/workgroup/drop.go` | `cfg.Repos[repoName]` (path lookup) |

**Key insight:** The internal map representation (`map[string]RepoConfig`) can be **preserved in memory** as a computed index even though the TOML source is now `[]RepoConfig`. This is the same approach Phase 1 used for `[[workspace]]`: parse into a slice, build any needed in-memory indexes at load time.

**Recommended approach:** Keep `WorkspaceConfig.Repos` as `map[string]RepoConfig` in memory (built from the `[]RepoConfig` slice at load time), add `RepoByName` as a convenience accessor. This minimizes the caller-side change — only the TOML serialization layer changes, not the in-memory API. The `diskConfig` struct uses `[]RepoConfig` for serialization; `WorkspaceConfig.Repos` stays as a map.

This avoids a massive cascade update across all 11 files.

### 2. Test Fixture Cascade: `[repos.<n>]` → `[[repo]]`

**23 occurrences** of the `[repos.<name>]` TOML pattern exist across **10 test files** plus **2 testutil helper files**. All must be updated to `[[repo]]` format with a `name` field.

Affected test files:
- `pkg/config/loader_test.go` (6 occurrences — inline TOML fixtures)
- `pkg/branch/checkout_test.go`, `create_test.go`, `default_test.go` (inline + dynamic builders)
- `pkg/workgroup/helpers_test.go` (dynamic builder)
- `pkg/repo/add_test.go`, `restore_test.go` (inline TOML)
- `pkg/git/info_test.go`, `commands_test.go`, `sync_test.go` (direct struct assignment)

Testutil helper files:
- `pkg/testutil/cmd.go` — `appendRepoTOML` and `makeWorkspaceWithRepoNames` use `[repos.%s]` format
- `pkg/testutil/helpers.go` — `setupWorkspaceDir` likely uses `[repos.%s]` format

**Key insight:** The `pkg/testutil/cmd.go` helpers (`appendRepoTOML`, `makeWorkspaceWithRepoNames`) are the highest-leverage fix — updating them cascades to fix all command-integration tests that use them. However, many test files also write raw TOML strings inline. Both testutil helpers AND all inline fixtures must be updated.

Note: `pkg/toml/preserve_test.go` uses its own local struct types (not `config.RepoConfig`) and `[repos.<n>]` as a generic TOML shape — it must NOT be updated (same decision made in Phase 1 Plan 01).

### 3. `URL` → `CloneURL` Rename: Affected Callsites

`RepoConfig.URL` (`toml:"url"`) is accessed in these non-test files:

| File | Usage |
|------|-------|
| `pkg/config/config.go` | Field definition `URL string toml:"url,omitempty"` |
| `pkg/config/loader.go` | `cfg.Repos[repoName] = RepoConfig{Path: ..., URL: setCfg.URL}` (worktree synthesis) |
| `pkg/repo/restore.go` | `rc.URL` (clone URL for restore, lines 187, 191) |
| `pkg/repo/add.go` | `config.RepoConfig{Path: relPath, URL: gitutil.RemoteURL(...)}` |
| `pkg/repo/clone.go` | `config.RepoConfig{Path: relPath, URL: url}` |

After rename: field becomes `CloneURL string toml:"clone_url,omitempty"` and all `.URL` references become `.CloneURL`.

**Test files also referencing `.URL`:**
- `pkg/repo/add_test.go` — `cfg.Repos[name].URL`
- `pkg/config/loader_test.go` — `cfg.Repos["frontend"].URL`
- `pkg/repo/restore_test.go` — `url = %q` in TOML fixtures (TOML key `url` → `clone_url`)
- `pkg/git/info_test.go` — `config.RepoConfig{Path: relPath}` (no URL used, unaffected)

### 4. `name` Required Field Validation

When TOML is `[[repo]]` array-of-tables, each entry must have a `name` field. Missing `name` must produce a load-time error. Implementation in `buildAndValidate`:

```go
func validateRepoNames(cfg *WorkspaceConfig) error {
    seen := make(map[string]struct{}, len(cfg.RepoList))
    for i, rc := range cfg.RepoList {
        if rc.Name == "" {
            return fmt.Errorf("[[repo]] entry %d missing required name field", i+1)
        }
        if _, dup := seen[rc.Name]; dup {
            return fmt.Errorf("duplicate [[repo]] name %q", rc.Name)
        }
        seen[rc.Name] = struct{}{}
    }
    return nil
}
```

The name-uniqueness check prevents `RepoByName` from having ambiguous results.

### 5. `track_branch`/`upstream` Co-presence Validation (D-01, D-02)

Rules from locked decisions:
- **D-01:** `track_branch` without `upstream` (or vice versa) is a load-time error.
- **D-02:** Within an `upstream` group, `track_branch` values must be unique.

```go
func validateAliasFields(cfg *WorkspaceConfig) error {
    // D-01: co-presence check
    for _, rc := range cfg.RepoList {
        hasTrack := rc.TrackBranch != ""
        hasUp := rc.Upstream != ""
        if hasTrack != hasUp {
            return fmt.Errorf("repo %q: track_branch and upstream must both be set or both be absent", rc.Name)
        }
    }

    // D-02: uniqueness per upstream group
    seen := make(map[string]map[string]string) // upstream -> track_branch -> repo name
    for _, rc := range cfg.RepoList {
        if rc.Upstream == "" {
            continue
        }
        if seen[rc.Upstream] == nil {
            seen[rc.Upstream] = make(map[string]string)
        }
        if prior, dup := seen[rc.Upstream][rc.TrackBranch]; dup {
            return fmt.Errorf("repo %q: track_branch %q already used by %q in upstream group %q",
                rc.Name, rc.TrackBranch, prior, rc.Upstream)
        }
        seen[rc.Upstream][rc.TrackBranch] = rc.Name
    }
    return nil
}
```

### 6. `RepoByName` and In-Memory Index

**D-08** requires a `RepoByName(name string) (RepoConfig, bool)` helper. Since the internal `Repos map[string]RepoConfig` is preserved (see Finding 1), this is a thin wrapper:

```go
// RepoByName returns the RepoConfig for the given name and whether it was found.
func (c *WorkspaceConfig) RepoByName(name string) (RepoConfig, bool) {
    rc, ok := c.Repos[name]
    return rc, ok
}
```

Existing call sites (`cfg.Repos["name"]`) continue to work unchanged during this phase. `RepoByName` is available for future use and for code that needs the idiomatic accessor pattern.

### 7. Disk vs. In-Memory Representation Split

The clean approach (same as Phase 1's `WorkspaceBlock` pattern):

**In memory:** `WorkspaceConfig.Repos map[string]RepoConfig` — keyed by name, built from the slice at load time.

**On disk (new `diskConfig`):** `RepoList []RepoConfig toml:"repo,omitempty"` — array-of-tables `[[repo]]`.

**Load path:**
1. TOML unmarshal into a `diskConfig` struct (has `RepoList []RepoConfig`)
2. `buildIndex` converts `RepoList` → `Repos map[string]RepoConfig` after name validation
3. Existing `buildAndValidate` runs against the populated `Repos` map

**Save path:**
1. `prepareDiskConfig` converts `Repos map[string]RepoConfig` → `RepoList []RepoConfig` (sorted by name for determinism)
2. Worktree-synthesized repos are excluded (same as current `withoutSynthesizedRepos`)

This preserves full backward compatibility for all internal callers — they continue using `cfg.Repos["name"]` unchanged.

### 8. Separation of Concerns: Two Plans

The work separates cleanly into two sequential plans:

**Plan 01 (Structs + TOML migration, no new fields yet):**
- `RepoConfig`: add `Name` field, rename `URL` → `CloneURL`, update TOML tags
- `diskConfig`: add `RepoList []RepoConfig toml:"repo,omitempty"`
- `WorkspaceConfig`: keep `Repos map[string]RepoConfig` (in-memory index)
- `loadMainConfig`: unmarshal via diskConfig, call `buildReposIndex` to populate `Repos`
- `buildAndValidate`: call `validateRepoNames` 
- `prepareDiskConfig`: convert `Repos` → `RepoList` (sorted, no synthesized repos)
- `RepoByName` accessor on `WorkspaceConfig`
- Update all testutil helpers and test fixtures from `[repos.<n>]` + `url` to `[[repo]]` + `clone_url`
- All existing test suites must pass

**Plan 02 (New fields + alias validation):**
- Add `TrackBranch` and `Upstream` fields to `RepoConfig`
- `buildAndValidate`: call `validateAliasFields`
- New tests: TOML round-trip for alias fields, co-presence error cases (D-01), uniqueness error cases (D-02)

### 9. `IsAlias()` Method (D-05, at discretion)

At agent's discretion. The field `TrackBranch != ""` (which implies `Upstream != ""` after D-01 validation) is the natural alias check. An `IsAlias() bool` method is clean:

```go
// IsAlias reports whether this repo is an env alias (has track_branch set).
func (r RepoConfig) IsAlias() bool {
    return r.TrackBranch != ""
}
```

Recommended: add it. Aids readability in future phases (Phase 17, 42, 43). Zero cost.

### 10. Validation Architecture (Nyquist)

The validation integration point is `buildAndValidate` in `pkg/config/loader.go`. The Phase 2 additions extend this function:

```go
func buildAndValidate(configPath string, cfg *WorkspaceConfig) error {
    // existing
    if err := validateWorktreePaths(configPath, cfg); err != nil { return err }
    if err := synthesizeWorktreeTargets(cfg); err != nil { return err }
    if err := validateRepoPaths(configPath, cfg); err != nil { return err }
    if err := validateAgenticFrameworks(cfg); err != nil { return err }
    // new in Phase 2
    if err := validateRepoNames(cfg); err != nil { return err }
    if err := validateAliasFields(cfg); err != nil { return err }
    return nil
}
```

Note: `validateRepoNames` and building the `Repos` map index from `RepoList` must happen before `validateRepoPaths` runs (since `validateRepoPaths` ranges over `cfg.Repos`). The load sequence must be: unmarshal → `buildReposIndex` → `ensureWorkspaceMaps` → `buildAndValidate`.

---

## Standard Stack

- **TOML parsing:** `github.com/BurntSushi/toml` (or `go-toml/v2` — existing `pkg/toml` wrapper)
- **Test suite:** `testify/suite`, `testutil.CmdSuite` pattern
- **No new dependencies needed**

---

## Architecture Patterns (from codebase)

- `WorkspaceBlock` (Phase 1) is the direct precedent for `[]RepoConfig` array-of-tables — use the same load-time index-building approach
- `buildAndValidate` is the canonical validation hook — all new validation goes here
- `diskConfig` vs `WorkspaceConfig` split already established — extend the same pattern
- `withoutSynthesizedRepos` pattern for save: exclude computed entries from disk

---

## Common Pitfalls

1. **Forgetting `pkg/toml/preserve_test.go`** — must NOT be updated (uses local struct types, not config schema)
2. **Validation order in `buildAndValidate`** — `validateRepoNames` + index build must precede `validateRepoPaths`
3. **`clone_url` in testutil helpers** — `appendRepoTOML` in `pkg/testutil/cmd.go` does not include URL; it only writes `path`. The `url`→`clone_url` rename only impacts fixtures that explicitly write the URL field (restore tests, add tests, specific loader tests)
4. **Sorted output in `prepareDiskConfig`** — `[]RepoConfig` in disk format should be written in sorted `name` order for deterministic diffs and test assertions
5. **Worktree-synthesized repos excluded from disk** — `prepareDiskConfig` must still call `withoutSynthesizedRepos` logic when building `RepoList`

---

## Plan Split Recommendation

| Plan | Focus | Wave | Files |
|------|-------|------|-------|
| 02-01 | `[[repo]]` struct migration + TOML key updates + test cascade | 1 | `pkg/config/config.go`, `pkg/config/loader.go`, `pkg/testutil/cmd.go`, 10 test files |
| 02-02 | `track_branch` + `upstream` fields + alias validation | 2 (depends on 02-01) | `pkg/config/config.go`, `pkg/config/loader.go`, `pkg/config/config_test.go`, `pkg/config/loader_test.go` |

---

## RESEARCH COMPLETE

Phase 2 is a well-scoped config migration. The approach is clear:
- Keep in-memory `Repos map[string]RepoConfig` (no cascade to 11 caller files)
- Add `diskConfig.RepoList []RepoConfig` for TOML serialization
- Build the map index at load time (Phase 1's `WorkspaceBlock` precedent)
- Two plans: migration first, new fields second
- Biggest risk is test fixture cascade (~23 occurrences) — testutil helper updates cascade to most command tests
