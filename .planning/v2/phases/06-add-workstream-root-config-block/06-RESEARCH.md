# Phase 6: Add `[[workstream]]` root config block - Research

**Researched:** 2026-04-05
**Domain:** git-w config schema + loader validation
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** `[[workstream]]` blocks are allowed in both `.gitw` and `.git/.gitw` in Phase 6.
- **D-02:** Presence of `[[workstream]]` in `.gitw` is accepted with no warning and no error.
- **D-03:** Phase 6 treats `.gitw` usage as intentional, and docs/error text should explicitly state that this is valid behavior (not misconfiguration).
- **D-04:** Phase 6 parses `[[workstream]]` with exactly two fields: `name` and `remotes`.
- **D-05:** `remotes` key is required to be present, but `remotes = []` is valid.
- **D-06:** Semantic meaning of `remotes = []` is explicit none override at workstream level (not fallback to outer defaults).
- **D-07:** Schema is strict for this block: unknown extra keys in `[[workstream]]` are load errors.
- **D-08:** `name` is required and must be non-empty for each `[[workstream]]` entry.
- **D-09:** Duplicate `[[workstream]]` names in the same loaded file are hard errors.
- **D-10:** `[[workstream]].remotes` values are validated immediately against declared `[[remote]]` names in Phase 6.
- **D-11:** Unknown remote names in a workstream remote list are hard errors, failing load on first unknown with actionable naming.
- **D-12:** Duplicate names within one `remotes` list (for example `"origin", "origin"`) are validation errors.
- **D-13:** In-memory `[[workstream]]` entries are normalized/sorted by `name` instead of preserving declaration order.
- **D-14:** `remotes` list values are normalized/sorted rather than preserving user-entered order.
- **D-15:** This differs from prior array-of-table declaration-order patterns and is intentional for Phase 6.

### the agent's Discretion
- Exact helper/function naming for `[[workstream]]` validation and normalization steps.
- Exact error message wording while preserving decision-level semantics.
- Internal representation details used to support strict-key validation in `go-toml/v2` loader flow.

### Deferred Ideas (OUT OF SCOPE)
- Configurable placement mode controlling shared-vs-private `[[workstream]]` placement.
</user_constraints>

<research_summary>
## Summary

Phase 6 cleanly fits the existing config architecture: add a new `WorkstreamConfig` schema type, wire `[[workstream]]` in `diskConfig`, and run a dedicated validator from `buildAndValidate` after remote validation. The current code already uses this pattern for `[[remote]]` and `[[sync_pair]]`, so no new subsystem is required.

The only non-routine piece is **strict key validation** for `[[workstream]]` (D-07) while preserving current loader behavior for other blocks. Recommended approach: keep normal typed unmarshal for runtime structures, and add a targeted key-check pass for `[[workstream]]` entries by parsing raw TOML into a map-like representation and rejecting keys outside `{name, remotes}` before returning from load.

**Primary recommendation:** Implement `WorkstreamConfig` + `validateWorkstreams` in the loader, with strict `[[workstream]]` key checks and sorted in-memory normalization by `name` and `remotes`.
</research_summary>

<standard_stack>
## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | 1.26 | Validation logic, sorting, error formatting | Already used across loader and validators |
| `github.com/pelletier/go-toml/v2` via `pkg/toml` | existing | Typed TOML marshal/unmarshal | Existing project standard for config parsing |
| `testify/suite` | existing | Table-driven suite tests in `loader_test.go` | Existing test convention in config package |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pkg/output` | existing | Warning output path (if ever needed) | Not needed for Phase 6 since decisions require hard errors/no warning |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Targeted `[[workstream]]` strict-key validation | Global unknown-field rejection | Global mode risks breaking existing tolerant parsing in unrelated blocks |
| Loader-level sorting of workstreams/remotes | Preserve declaration order | Violates locked D-13 and D-14 |
</standard_stack>

<architecture_patterns>
## Architecture Patterns

### Pattern 1: Schema-on-config + diskConfig wiring
**What:** Keep domain schema structs in `pkg/config/config.go`, wire TOML array-of-tables through `diskConfig` in `pkg/config/loader.go`, then copy to `WorkspaceConfig`.
**When to use:** Every new top-level TOML block (`[[remote]]`, `[[sync_pair]]`, now `[[workstream]]`).

### Pattern 2: Centralized validation chain
**What:** All load-time validation runs in `buildAndValidate(configPath, cfg)` with ordered helper calls.
**When to use:** Any structural, referential, or semantic checks needed at load.

### Pattern 3: Hard errors for structural violations
**What:** Missing required fields, duplicate keys, invalid references return immediate errors.
**When to use:** Config shape and correctness constraints (applies directly to D-08..D-12).

### Anti-Patterns to Avoid
- Adding warning-only behavior for `.gitw` `[[workstream]]` usage (conflicts with D-02/D-03).
- Deferring remote reference checks to later phases (conflicts with D-10/D-11).
- Keeping declaration order for workstreams/remotes in memory (conflicts with D-13/D-14).
</architecture_patterns>

<dont_hand_roll>
## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Config parsing pipeline | Custom parser for full `.gitw` format | Existing `pkg/toml` + typed structs | Existing parser already works for all current blocks |
| Validation orchestration | Parallel/distributed validation paths | `buildAndValidate` sequence | Single source of truth avoids missed checks |
| Sorting/normalization | Custom ordering semantics per consumer | Normalize once in loader | Keeps downstream behavior deterministic |
</dont_hand_roll>

<common_pitfalls>
## Common Pitfalls

### Pitfall 1: `remotes` treated as optional
**What goes wrong:** Missing `remotes` is silently accepted.
**Why it happens:** Zero-value slice makes omission look like empty list.
**How to avoid:** Track field presence explicitly during parse/validation and reject missing key (D-05).

### Pitfall 2: Unknown key acceptance in `[[workstream]]`
**What goes wrong:** Typo keys pass through (`remote` instead of `remotes`).
**Why it happens:** Default unmarshal may ignore unknown fields.
**How to avoid:** Add strict key check for each `[[workstream]]` table (D-07).

### Pitfall 3: Non-deterministic ordering
**What goes wrong:** Workstream list/remotes order varies by source.
**Why it happens:** No normalization step.
**How to avoid:** Sort `cfg.Workstreams` by `name` and each `Remotes` slice before returning loaded config (D-13/D-14).
</common_pitfalls>

## Validation Architecture

Nyquist-relevant verification points for this phase:

1. **Schema parse tests**: `[[workstream]]` blocks load correctly in `.gitw` and `.git/.gitw`.
2. **Validation tests**: required keys, duplicate names, unknown remotes, duplicate remotes, and unknown keys fail with actionable errors.
3. **Normalization tests**: loaded `Workstreams` and each `Remotes` slice are sorted in-memory.
4. **Round-trip tests**: save/load preserves valid data and supports multiple `[[workstream]]` blocks.

Primary command budget:
- Fast loop: `mage testfast`
- Final gate before done: `mage test`

<sources>
## Sources

### Primary (HIGH confidence)
- `.planning/phases/06-add-workstream-root-config-block/06-CONTEXT.md` — locked decisions D-01..D-15
- `.planning/v2/v2-schema.md` — `[[workstream]]` schema and cascade semantics
- `.planning/v2/v2-remote-management.md` — workstream remote override intent in cascade
- `pkg/config/config.go` and `pkg/config/loader.go` — existing schema/loader patterns
- `pkg/config/loader_test.go` and `pkg/config/config_test.go` — established test patterns

### Secondary (MEDIUM confidence)
- `.planning/phases/04-add-remote-and-remote-branch-rule/*-SUMMARY.md` — remote validation/wiring patterns
- `.planning/phases/05-add-sync-pair-parsing/*-SUMMARY.md` — validator split and cycle-check integration pattern
</sources>

<metadata>
## Metadata

**Research scope:**
- Core technology: config schema + loader
- Patterns: array-of-tables wiring, validator chaining, deterministic normalization
- Risks: strict key enforcement and required-key presence detection for `remotes`

**Confidence breakdown:**
- Standard stack: HIGH
- Architecture: HIGH
- Pitfalls: HIGH
- Verification strategy: HIGH

**Research date:** 2026-04-05
**Valid until:** 2026-05-05
</metadata>

---

*Phase: 06-add-workstream-root-config-block*
*Research completed: 2026-04-05*
*Ready for planning: yes*
