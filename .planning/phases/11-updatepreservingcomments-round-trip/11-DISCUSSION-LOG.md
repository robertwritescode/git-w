# Phase 11: `UpdatePreservingComments` round-trip - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 11-updatepreservingcomments-round-trip
**Areas discussed:** Test scope and coverage targets, Tech debt fixes, Test file location and structure

---

## Test scope and coverage targets

**Which block types to cover:**

| Option | Description | Selected |
|--------|-------------|----------|
| All v2 blocks in diskConfig | Cover all block types: `metarepo`, `[[workspace]]`, `[[repo]]` (with `[[repo.branch_override]]`), `[[remote]]` (with `[[remote.branch_rule]]`), `[[sync_pair]]`, `[[workstream]]`, `[groups]`, `[worktrees]`. Full serialization surface. | ✓ |
| Only new blocks added in Phases 1-10 | Focus on blocks not previously tested. Smaller scope but leaves gaps on existing types. | |
| Representative sample | Cover 2-3 block types as smoke test. Fast but low confidence for CFG-12. | |

**User's choice:** All v2 blocks in diskConfig

---

**How to exercise the code:**

| Option | Description | Selected |
|--------|-------------|----------|
| Via `config.Save()` + `config.Load()` | Production path — exercises the full pipeline including `saveWithCommentPreservation` and the TOML round-trip. | ✓ |
| Call `toml.UpdatePreservingComments` directly | Unit-level, more targeted but skips the production wiring. | |

**User's choice:** Via `config.Save()` + `config.Load()` on a real temp `.gitw` file

---

**Test structure:**

| Option | Description | Selected |
|--------|-------------|----------|
| Table-driven, one case per block type | Single test function with cases; easy to scan, easy to add new cases, consistent with codebase patterns. | ✓ |
| Separate `TestXxx` per block | More isolation but duplicates setup; violates DRY. | |

**User's choice:** Table-driven with one case per block type

---

**Assertion level:**

| Option | Description | Selected |
|--------|-------------|----------|
| Comment appears at correct position | Assert comment appears in output AND before its anchor key/section header. Catches displacement. | ✓ |
| Comment presence only (`Contains`) | Simpler but does not detect comment displacement. | |

**User's choice:** Position check — comment line index must be less than anchor line index in the output

---

## Tech debt fixes

**Fix `interface{}` → `any`:**

| Option | Description | Selected |
|--------|-------------|----------|
| Fix in Phase 11 alongside tests | Both concerns are in `pkg/toml/preserve.go`; fix them together while tests are being written. | ✓ |
| Defer to a dedicated cleanup phase | Separate concern, lower risk. But trivially safe and already documented as tech debt. | |

**User's choice:** Fix in Phase 11

---

**Fix silent error swallow in `applySmartUpdate`:**

| Option | Description | Selected |
|--------|-------------|----------|
| Propagate the error | `applySmartUpdate` already returns `([]byte, error)` — return the error from `smartUpdate` instead of silently returning `newBytes, nil`. | ✓ |
| Log a warning and fall back | Surface to user without hard failure; comments lost silently today, warned tomorrow. | |
| Defer to a dedicated phase | Already documented in CONCERNS.md; Phase 11 is the natural time to fix it. | |

**User's choice:** Propagate the error

---

## Test file location and structure

**Where to put the new tests:**

| Option | Description | Selected |
|--------|-------------|----------|
| `pkg/config/round_trip_test.go` | Dedicated file, clearly named, easy to find. Keeps round-trip tests separate from loader and stream tests. | ✓ |
| Add to `pkg/config/loader_test.go` | Collocated with `Save`/`Load` tests. Less separation of concerns. | |
| `pkg/toml/preserve_test.go` | TOML package tests, but tests would use `config.Save`/`Load` — wrong package. | |

**User's choice:** `pkg/config/round_trip_test.go`

---

**Suite vs plain test:**

| Option | Description | Selected |
|--------|-------------|----------|
| Testify suite (CmdSuite-style) | Tests call `config.Save` + `config.Load` using real temp `.gitw` files; suite provides clean `SetupTest`/`TeardownTest` lifecycle. | ✓ |
| Plain `func TestXxx(t *testing.T)` | Simpler, but no shared lifecycle; per-test setup would be duplicated. | |

**User's choice:** Testify suite

---

**Per-subtest isolation:**

| Option | Description | Selected |
|--------|-------------|----------|
| `s.T().TempDir()` inside each `s.Run` closure | Full isolation; each subtest gets its own temp dir automatically cleaned up. | ✓ |
| Single `TempDir` shared across table cases | Simpler but subtests can interfere if filenames collide. | |

**User's choice:** `s.T().TempDir()` per subtest

---

## Agent's Discretion

- Exact comment placement in each test fixture
- Whether to add a single comprehensive "all blocks" integration case in addition to per-block table cases
- Field naming and struct layout within test helper functions
- Whether `saveWithCommentPreservation` in `loader.go` needs updating alongside `preserve.go` changes

## Deferred Ideas

- None — discussion stayed within phase scope
