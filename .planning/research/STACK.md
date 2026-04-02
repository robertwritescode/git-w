# Stack Research

**Domain:** Go CLI tool — multi-repo git orchestration (v2 new dependencies + M1 config schema)
**Researched:** 2026-04-02
**Confidence:** HIGH

> Scope: NEW dependencies and algorithmic patterns. The general v2 stack (Gitea SDK,
> go-github, doublestar, errgroup) is documented in the prior section and preserved here.
> The section below **M1 Config Schema Additions** is the new research for this milestone.

---

## M1 Config Schema Additions

### The Core Question: go-toml/v2 Comment Preservation

**Answer: Stay with go-toml/v2. Do NOT add a new library. Extend the existing `pkg/toml.UpdatePreservingComments` instead.**

#### What go-toml/v2 provides today (v2.2.4, pinned in go.mod)

1. **`Marshal` / `Unmarshal`** — stable API, already used throughout `pkg/config/` and `pkg/toml/`.

2. **`comment` / `commented` struct tags** — when *marshaling*, these emit `# comment text` above fields
   and `# field = value` commented-out fields. This is for *generating* annotated config files, NOT for
   preserving user-written comments during round-trips. Irrelevant to CFG-12.

3. **`unstable.Parser` with `KeepComments: true`** — the AST-level parser exposes `Kind.Comment` nodes
   with their exact raw bytes and byte-range positions. This is how the library represents user-written
   comments internally. It is the correct foundation for a comment-preserving round-trip implementation.
   **Confirmed in go-toml/v2 unstable package docs (v2.3.0).** The unstable API explicitly says it does not
   follow backward compatibility guarantees yet, but the `KeepComments` flag has been stable behavior
   since v2.0.0-beta.4.

4. **No built-in `UpdatePreservingComments`** — go-toml/v2 has explicitly removed document editing from
   scope (maintainer stated in discussion #506, October 2021). There is no `toml.Document` type or
   comment-preserving write API in the stable package. The maintainer noted "the parser has been built in
   a way that should be solid foundation to support Document Editing — should just need to make some
   functions public and expose comments in the AST." This confirms that `unstable.Parser{KeepComments: true}`
   is the right hook but that the round-trip assembly logic must be written in application code.

5. **Latest version: v2.3.0** (published 2026-03-24, confirmed on pkg.go.dev). Current pinned version:
   v2.2.4. The v2.3.0 delta adds `unstable.RawMessage` type. No breaking changes.

#### Why the current `UpdatePreservingComments` needs extending for v2 schema

The existing implementation in `pkg/toml/preserve.go` works by:
1. Marshaling old/new structs to bytes, diffing at section level via regex (`[section]` / `[section.sub]`)
2. Re-injecting comments anchored to key names via string matching

**This works for the current flat schema** (single-level tables: `[workspace]`, `[repos]`, `[groups]`).

**It will NOT work as-is for the v2 array-of-tables schema** because:
- `[[remote]]`, `[[repo]]`, `[[workspace]]`, `[[sync_pair]]` are *array tables* (`[[double bracket]]`),
  not regular tables (`[single bracket]`)
- The current `findSectionBounds` regex only matches `^\[section\]` patterns, not `^\[\[section\]\]`
- Multiple `[[remote]]` blocks with comments between them cannot be identified by section name alone;
  they must be matched by array index or key field value (e.g., `name = "personal"`)
- Field-order preservation within an array table block requires knowing which array entry corresponds
  to which identity key

**The fix is surgical**: extend `pkg/toml/preserve.go` to handle `[[array table]]` patterns.
No new external library is needed.

#### Upgrade path for go-toml/v2

Bumping to v2.3.0 is safe and should be done:

```bash
go get github.com/pelletier/go-toml/v2@v2.3.0
```

No API changes affect existing code. The `unstable.RawMessage` addition in v2.3.0 is purely additive.
This gives access to `unstable.RawMessage` if needed for deferred raw TOML decoding in the validator.

### Structured Validation Pattern

**Use: hand-written `validate()` functions with named errors. No external validation library.**

The v2 schema requires several load-time validations (CFG-03, CFG-10, CFG-11):
- `repos/<n>` path convention check with v1 warning
- v1 `[[workgroup]]` detection with actionable error
- `agentic_frameworks` registry check

**Recommended pattern:**

```go
// Named sentinel errors for typed checking in tests
var ErrV1Workgroup = errors.New("v1 config detected: [[workgroup]] blocks are not supported in v2")
var ErrUnknownFramework = errors.New("unknown agentic framework")

// Validator function per concern, pure (no side effects)
func validateAgenticFrameworks(frameworks []string) error {
    for _, f := range frameworks {
        if agents.FrameworkFor(f) == nil {
            return fmt.Errorf("%w: %q (valid: %s)", ErrUnknownFramework, f, agents.KnownFrameworks())
        }
    }
    return nil
}
```

**Why not a validation library (e.g., `go-playground/validator`)?**
- The schema uses semantic validation (registry lookups, graph properties, file-level constraints),
  not structural validation (required fields, string lengths, regex patterns). Struct tags handle
  the structural side; semantic checks need custom code regardless.
- Adding validator adds ~15 transitive dependencies for zero benefit.
- The existing codebase has no precedent for it; hand-written validators follow the established pattern.

### Cycle Detection Algorithm

**Use: iterative DFS with three-color marking. No external graph library.**

The `[[sync_pair]]` blocks form a directed graph where `from` and `to` are node identities (remote names).
Cycle detection at load time (CFG-05) is a standard graph algorithm.

**Recommended algorithm:**

```go
// Three-color DFS: white=unvisited, gray=in-stack, black=done
type color int
const (
    white color = iota
    gray
    black
)

// DetectSyncPairCycle returns the cycle path if found, nil if the graph is acyclic.
func DetectSyncPairCycle(pairs []SyncPair) []string {
    // Build adjacency list: from -> []to
    adj := make(map[string][]string)
    for _, p := range pairs {
        adj[p.From] = append(adj[p.From], p.To)
    }

    visited := make(map[string]color)
    var path []string

    var dfs func(node string) bool
    dfs = func(node string) bool {
        visited[node] = gray
        path = append(path, node)
        for _, neighbor := range adj[node] {
            switch visited[neighbor] {
            case gray:
                path = append(path, neighbor) // close the cycle for error message
                return true
            case white:
                if dfs(neighbor) {
                    return true
                }
            }
        }
        path = path[:len(path)-1]
        visited[node] = black
        return false
    }

    for node := range adj {
        if visited[node] == white {
            if dfs(node) {
                return path
            }
        }
    }
    return nil
}
```

**Why not `gonum/graph`?**
- The `[[sync_pair]]` graph is tiny (typically 2-5 nodes, at most ~20 remotes in any real config).
  gonum/graph is a heavyweight scientific computing library with 50+ transitive dependencies.
- Standard DFS on a `map[string][]string` adjacency list is ~20 lines. No external dependency justified.
- This is the same pattern used by Go's module dependency cycle detection (no library).

**Error message format** (aligns with spec requirement for actionable errors):

```
sync_pair cycle detected: origin -> personal -> backup -> origin
```

Return the full cycle path so the user knows which pairs to remove.

### Two-File Merge Pattern

**Use: `MergeRemote`, `MergeRepo`, etc. functions in `pkg/config`. No external library.**

Field-level merge of `.gitw` + `.git/.gitw` follows a well-defined rule: private file wins on conflicts,
base file fills in zero-value fields. The schema spec (v2-schema.md) documents `MergeRemote(base, override Remote)`.

**Recommended pattern per merge function:**

```go
// MergeRemote merges base and override Remote; override wins on non-zero fields.
func MergeRemote(base, override Remote) Remote {
    result := base
    if override.Kind != "" {
        result.Kind = override.Kind
    }
    if override.URL != "" {
        result.URL = override.URL
    }
    // ... all exported fields
    if len(override.BranchRules) > 0 {
        result.BranchRules = override.BranchRules
    }
    return result
}
```

This is purely algorithmic; no library provides value here.

### `agentic_frameworks` Registry Pattern

**Use: a `map[string]SpecFramework` in `pkg/agents`. No external registry library.**

```go
var registry = map[string]SpecFramework{
    "gsd": &GSDFramework{},
}

// FrameworkFor returns the framework implementation for name, or nil if unknown.
func FrameworkFor(name string) SpecFramework {
    return registry[name]
}

// KnownFrameworks returns a sorted list of valid framework names for error messages.
func KnownFrameworks() []string {
    names := make([]string, 0, len(registry))
    for k := range registry {
        names = append(names, k)
    }
    sort.Strings(names)
    return names
}
```

Default behavior when `agentic_frameworks` is absent: return `["gsd"]`. This is set at load time
in the config loader, not in the registry itself.

---

## Recommended Stack (Full, Including Prior Research)

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `github.com/pelletier/go-toml/v2` | v2.3.0 | TOML parsing, marshaling, comment-preserving round-trips | Bump from v2.2.4 to v2.3.0 for `unstable.RawMessage` and latest fixes. Stay with this library; no alternative needed. Document editing is intentionally application-layer responsibility in this library's design. |
| `code.gitea.io/sdk/gitea` | v0.24.1 | Gitea API provider (`internal/provider/`) | First-party SDK, typed methods, no hand-rolled HTTP. |
| `github.com/google/go-github/v84` | v84.0.0 | GitHub API provider (`internal/provider/`) | Google-maintained, tracks GitHub REST API. `v84` targets Go 1.26. |
| `github.com/bmatcuk/doublestar/v4` | v4.10.0 | Glob pattern matching for branch rules | `**` crosses `/`; `*` does not. Exact semantics the spec requires. Zero deps. |
| `golang.org/x/sync/errgroup` | v0.20.0 | Bounded parallel fan-out (sync executor) | `SetLimit(n)` replaces manual semaphore patterns. Official Go sub-repo. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/pelletier/go-toml/v2/unstable` | v2.3.0 (same module) | AST-level TOML parsing with `KeepComments: true` | Only in `pkg/toml/preserve.go` for extending `UpdatePreservingComments` to handle `[[array tables]]`. Import the unstable sub-package directly; its API is practically stable despite the name. |
| `encoding/json` (stdlib) | Go 1.26 | JSON output for `--json` flags | All `--json` output. Sufficient for flat output structs. |
| `net/http` (stdlib) | Go 1.26 | HTTP client for API providers | Pass configured `*http.Client` to SDK constructors. |
| `context` (stdlib) | Go 1.26 | Cancellation propagation | `errgroup.WithContext`, API provider calls. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `mage` (existing) | Build/test orchestration | No change. |
| `golangci-lint` (existing) | Linting | No change. Covers new packages. |
| `go test -race` (via mage) | Race detection for errgroup concurrency | Critical for new fan-out. `mage test` already runs with `-race`. |

## Installation

```bash
# Upgrade go-toml/v2 to latest (patch bump, no API changes)
go get github.com/pelletier/go-toml/v2@v2.3.0

# New runtime dependencies for later milestones (M3+)
go get code.gitea.io/sdk/gitea@v0.24.1
go get github.com/google/go-github/v84@v84.0.0
go get github.com/bmatcuk/doublestar/v4@v4.10.0
go get golang.org/x/sync@v0.20.0
```

**M1 only needs the go-toml/v2 bump.** The other packages are for M3+ (sync, remote management).

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Stay with `go-toml/v2` + extend `UpdatePreservingComments` | Switch to `BurntSushi/toml` | Never. BurntSushi/toml has no comment-preservation API either. Switching rewrites all config code for zero gain. |
| Stay with `go-toml/v2` + extend `UpdatePreservingComments` | `github.com/naoina/toml` | Never. Unmaintained (last commit 2021), no `[[array table]]` guarantee, not worth the migration risk. |
| Stay with `go-toml/v2` + extend `UpdatePreservingComments` | `tomtom` / custom AST-preserving TOML writer | Would require a complete replacement of all config I/O. The `unstable.Parser{KeepComments: true}` already provides the raw bytes + position information needed to implement comment preservation without switching libraries. |
| Hand-written DFS for cycle detection | `gonum/graph` | Never for this scale. gonum adds 50+ transitive deps for a problem solvable in 20 lines of stdlib code. |
| Hand-written `validate()` functions | `go-playground/validator` | Never. Semantic validation (registry lookup, graph properties) cannot be expressed in struct tags. |
| Hand-written merge functions | `mergo` or similar | Never. The schema merge rules are field-specific (some fields have special zero-value semantics, array fields replace rather than append). A generic library cannot express these. |
| `go-github/v84` | `go-gitlab` | Add later if GitLab support is required. Same Provider interface; swap implementation. |
| `doublestar/v4` | `filepath.Match` (stdlib) | Never for branch rules. `filepath.Match` has OS-specific path sep behavior and no `**`. |
| `errgroup` | Manual semaphore + WaitGroup | Keep `parallel.RunFanOut` for fire-and-forget. Use errgroup for first-error-cancels-all sync. Both coexist. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Any new TOML library (`BurntSushi/toml`, `naoina/toml`) | None of them provide comment-preserving round-trips either. The problem must be solved at the application layer regardless of which library you use. Switching creates migration cost for zero capability gain. | Extend `pkg/toml/preserve.go` using `go-toml/v2/unstable.Parser{KeepComments: true}` |
| `go-playground/validator` | Struct-tag-based validators cannot express semantic rules (registry lookup, graph cycle, file-location enforcement). Adds ~15 transitive deps for features that don't apply here. | Hand-written `validate()` functions per concern in `pkg/config/validator.go` |
| `gonum/graph` | 50+ transitive dependencies for a 3-5 node graph. Completely disproportionate. | Standard iterative DFS on `map[string][]string` in `pkg/config/cycles.go` (~20 lines) |
| `mergo` / `dario.cat/mergo` | Generic struct merging cannot express per-field semantics (e.g., `remotes []string` replaces rather than appends; `private bool` has a false-zero that means "not private" vs. unset). | Explicit per-type `MergeX(base, override X) X` functions in `pkg/config/merge.go` |
| `bubbletea` / `lipgloss` | Project constraint: no TUI. | `text/tabwriter` + `output.Writef` |
| `go-git` | Creates two incompatible git execution paths. Contradicts existing `os/exec` architecture. | `os/exec` via `pkg/gitutil/` for `git remote add` / `git remote set-url` |
| `viper` | Heavyweight config framework adds 10+ transitive deps for env-var/YAML features not needed. | Existing `go-toml/v2` + `pkg/config` with new merge functions |
| `oauth2` / `golang.org/x/oauth2` | API providers use static PATs from config, not OAuth flows. | Read token from config, pass to SDK constructors directly. |

## Stack Patterns by Variant

**For `UpdatePreservingComments` with `[[array tables]]`:**
- Use `unstable.Parser{KeepComments: true}` to walk the original document AST
- Associate each `Comment` node (by byte offset) with the nearest following non-comment `KeyValue` or `ArrayTable` node
- When re-emitting, marshal the new data with `toml.Marshal`, then re-inject comment bytes at the
  corresponding offsets relative to each array table entry's identity field (e.g., `name = "personal"`)
- Fall back to `toml.Marshal` output (no comments) if anchor matching fails — never corrupt the data

**For v1 `[[workgroup]]` detection:**
- Parse with `Decoder.DisallowUnknownFields()` disabled (permissive decode)
- After successful unmarshal, check if the raw bytes contain `[[workgroup]]` via a single
  `bytes.Contains` call — faster than a second parse
- Return a named error wrapping `ErrV1Config` with an actionable message: `"v1 [[workgroup]] blocks detected; run: git w migrate"`

**For `private = true` enforcement (`.gitw` vs `.git/.gitw` file check):**
- After loading `.gitw`, iterate `cfg.Remotes` and return a named error for any `Remote.Private == true`
- Error message: `"remote %q has private=true but is in .gitw; move it to .git/.gitw"`
- This is a load-time semantic check, not a schema validation — lives in `pkg/config/loader.go`

**If adding a new API provider (e.g., GitLab):**
- Implement the `Provider` interface in `internal/provider/gitlab/`
- Add `go get github.com/xanzy/go-gitlab` (or successor)
- Register in the provider factory. No architectural changes needed.

**If sync fan-out needs fire-and-forget semantics:**
- Do NOT use errgroup. Use the existing `parallel.RunFanOut` pattern.
- Use errgroup only for first-error-cancels-all semantics.

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `go-toml/v2` v2.3.0 | Go 1.21+ (module min); Go 1.26 (this project) | Patch bump from v2.2.4. Drop-in replacement. `unstable` sub-package available as a sub-path import. |
| `go-toml/v2/unstable` v2.3.0 | Same module, same version | Sub-package of go-toml/v2. No separate `go get` needed; import path `github.com/pelletier/go-toml/v2/unstable`. |
| `go-github/v84` v84.0.0 | Go 1.26+ | v84 module path includes major version. Uses generics (Go 1.18+). |
| `gitea-sdk` v0.24.1 | Go 1.21+ | No generics dependency. Compatible with Go 1.26. |
| `doublestar/v4` v4.10.0 | Go 1.18+ | Minimal Go version. Zero external dependencies. |
| `golang.org/x/sync` v0.20.0 | Go 1.22+ | `errgroup.SetLimit` available since x/sync v0.7.0. v0.20.0 is latest. |

## Sources

- `pkg.go.dev/github.com/pelletier/go-toml/v2` — confirmed v2.3.0 published 2026-03-24. `unstable.Parser.KeepComments` confirmed present in v2.3.0 AST docs. HIGH confidence.
- `pkg.go.dev/github.com/pelletier/go-toml/v2/unstable` — confirmed `Kind.Comment`, `Parser.KeepComments` field, full AST example with comment nodes. HIGH confidence.
- `github.com/pelletier/go-toml/discussions/506` — maintainer explicitly removed document editing from scope October 2021. Confirms no built-in comment round-trip will be added. HIGH confidence.
- `pkg/toml/preserve.go` (codebase) — read directly; confirmed current regex approach handles `[section]` but not `[[array table]]`. HIGH confidence.
- `pkg.go.dev/code.gitea.io/sdk/gitea` — confirmed v0.24.1 published 2026-03-24. HIGH confidence.
- `github.com/google/go-github` releases — confirmed v84.0.0 published 2026-02-27. HIGH confidence.
- `pkg.go.dev/github.com/bmatcuk/doublestar/v4` — confirmed v4.10.0 published 2026-01-25. HIGH confidence.
- `pkg.go.dev/golang.org/x/sync/errgroup` — confirmed v0.20.0 published 2026-02-23. HIGH confidence.
- `go.mod` (codebase) — confirmed current dependency versions. HIGH confidence.
- `.planning/v2/v2-schema.md` — spec for merge semantics, sync_pair cycle detection, comment round-trips. HIGH confidence.
- `.planning/REQUIREMENTS.md` — CFG-01 through CFG-12 requirements. HIGH confidence.

---
*Stack research for: git-w v2 M1 config schema + loader*
*Researched: 2026-04-02*
