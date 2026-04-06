# Phase 8: Parse `.gitw-stream` manifest - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-06
**Phase:** 08-parse-gitw-stream-manifest
**Areas discussed:** Type placement and package, Loader API design, Scope of [ship]/[context] blocks, Default and uniqueness rules

---

## Type placement and package

| Option | Description | Selected |
|--------|-------------|----------|
| New types in pkg/config (stream.go) | Types in config.go alongside WorkspaceConfig; loader in new stream.go. All config I/O in one package | ✓ |
| Separate pkg/stream package | New package for a single file format — clean but creates package for one concern used by later phases | |
| Types in config.go, loader in stream.go | Same as recommended but making the split explicit | |

**User's choice:** New types in pkg/config (stream.go) — types alongside existing config types, loader in stream.go

---

| Option | Description | Selected |
|--------|-------------|----------|
| StreamManifest + WorktreeEntry | Standard naming convention | |
| WorkstreamManifest + WorktreeEntry | Matches workstream terminology used throughout the rest of the codebase | ✓ |
| Agent's discretion on names | Defer to agent | |

**User's choice:** WorkstreamManifest — matches workstream terminology

---

| Option | Description | Selected |
|--------|-------------|----------|
| Plain string for status | Simple, matches TOML | |
| WorkstreamStatus type with constants | Type-safe, discoverable, consistent with BranchAction pattern | ✓ |

**User's choice:** WorkstreamStatus typed string with constants Active/Shipped/Archived

---

## Loader API design

| Option | Description | Selected |
|--------|-------------|----------|
| LoadStream(path) — explicit path | Caller controls discovery; consistent with config.Load(path) | ✓ |
| LoadStream(dir) — directory-based discovery | Convenience but hides filesystem walk | |
| Both explicit and directory variants | Two functions; more surface area | |

**User's choice:** LoadStream(path string) — explicit path

---

| Option | Description | Selected |
|--------|-------------|----------|
| (*WorkstreamManifest, error) | Pointer return, nil on error | ✓ |
| (WorkstreamManifest, error) | Value type, forces zero-value handling | |

**User's choice:** (*WorkstreamManifest, error)

---

| Option | Description | Selected |
|--------|-------------|----------|
| Return os.ErrNotExist, caller handles | Consistent with mergeLocalConfig silent-skip pattern | ✓ |
| Wrap with domain error | More context but wrapping loses errors.Is compatibility | |
| Always error on missing file | Too strict for optional manifest | |

**User's choice:** Return os.ErrNotExist unwrapped; callers use errors.Is

---

## Scope of [ship] and [context] blocks

| Option | Description | Selected |
|--------|-------------|----------|
| Define full types now, parse them | ShipState and StreamContext with all fields; types ready for M9/M10 | ✓ |
| Skip [ship] and [context] for now | Add types when M9/M10 need them | |
| Zero-value stub structs only | Placeholder to avoid unknown-key errors | |

**User's choice:** Define full ShipState and StreamContext types with all schema fields

---

| Option | Description | Selected |
|--------|-------------|----------|
| Test success criteria only | [ship]/[context] round-trip coverage deferred to Phase 11 | |
| Test all fields including [ship]/[context] | Full coverage since types are being defined now | ✓ |

**User's choice:** Test all defined fields in Phase 8

---

## Default and uniqueness rules

| Option | Description | Selected |
|--------|-------------|----------|
| Defaults applied in LoadStream (build step) | Applied after parse, before validation; callers get normalized struct | ✓ |
| Defaults via accessor methods | Raw struct preserved; computed on access | |
| Defaults applied during validation | Applied in-place during validate step | |

**User's choice:** Apply defaults in LoadStream build step; every returned WorktreeEntry has name and path populated

---

| Option | Description | Selected |
|--------|-------------|----------|
| Single validateStream function | One function covers name uniqueness, path uniqueness, multi-occurrence check | ✓ |
| Two separate validation functions | validateWorktreeFields + validateWorktreeUniqueness | |

**User's choice:** Single validateStream(manifest) function

---

| Option | Description | Selected |
|--------|-------------|----------|
| Error on empty name for multi-occurrence repo | Explicit check before uniqueness; clear error message | ✓ |
| Rely on uniqueness check to catch multi-occurrence | Implicit; defaults produce colliding empty names | |

**User's choice:** Explicit error: "worktree entry for repo %q requires a name when the repo appears multiple times"

---

## Agent's Discretion

- Exact field names on ShipState and StreamContext Go structs (TOML tags must match schema)
- Internal helper names within stream.go
- Table layout and test structure within stream_test.go

## Deferred Ideas

None.
