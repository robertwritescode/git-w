# Phase 5: Add `[[sync_pair]]` parsing - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-04
**Phase:** 05-add-sync-pair-parsing
**Areas discussed:** Cycle detection design, refs field defaults, MergeSyncPair timing, Validation scope

---

## Cycle detection design

| Option | Description | Selected |
|--------|-------------|----------|
| DFS with visited/stack sets | Standard DFS over the directed sync graph — O(V+E), well-understood, easy to test | ✓ |
| Topological sort (Kahn's) | Works but harder to produce the cycle path string for the error message | |
| Simple path enumeration | Iterate paths; acceptable for small graphs but less principled | |

**User's choice:** DFS with visited/stack sets

---

| Option | Description | Selected |
|--------|-------------|----------|
| Full cycle path with arrow notation | e.g. "sync_pair cycle detected: origin → personal → contractor → origin" | ✓ |
| Start and end node only | e.g. "sync_pair cycle: origin and personal form a loop" | |
| Generic cycle error | No specific node names | |

**User's choice:** Full cycle path with arrow notation — repeat the starting node at the end so the cycle is visually obvious.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Report first cycle only | Stop at first cycle, return one error — simpler | ✓ |
| Report all cycles | Walk entire graph and collect all cycles before returning | |

**User's choice:** Report first cycle only

---

## refs field defaults

| Option | Description | Selected |
|--------|-------------|----------|
| omitempty, nil means all-refs | `toml:"refs,omitempty"` — omitted when empty; consumers treat nil/empty as "all refs" | ✓ |
| Default to ["**"] at load time | Populate at load time; serializes back to `refs = ["**"]` — noisier output | |
| Required field, no default | Omitting refs is a validation error | |

**User's choice:** omitempty, nil means all-refs

---

## MergeSyncPair timing

| Option | Description | Selected |
|--------|-------------|----------|
| Define now alongside the struct | Same pattern as MergeRemote in Phase 4; Phase 7 just calls it | ✓ |
| Defer to Phase 7 | Less code in Phase 5; Phase 7 planner derives semantics | |

**User's choice:** Define now alongside the struct

---

| Option | Description | Selected |
|--------|-------------|----------|
| Override file refs wins (full replace) | Private file's refs completely replace base file's refs for same (from,to) pair | ✓ |
| Union of refs slices | Merge the refs slices (deduplicated union) | |
| Non-zero wins (same as MergeRemote) | Private file's refs replaces if non-empty, otherwise base used | |

**User's choice:** Override file refs wins (non-empty override wins; if override is nil/empty, base is used) — clarified as "non-zero wins" consistent with MergeRemote scalar field pattern.

---

## Validation scope

| Option | Description | Selected |
|--------|-------------|----------|
| Structural only: empty, duplicates, cycles | Validate non-empty from/to, no duplicate pairs, cycle detection | ✓ |
| Include name-reference validation | Also check from/to match a defined [[remote]] name | |
| Cycles only | Skip duplicate and empty checks | |

**User's choice:** Structural only (empty, duplicates, cycles). Name-reference validation deferred — referenced remote may live only in .git/.gitw.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Hard error | return error — consistent with duplicate [[remote]] names in Phase 4 | ✓ |
| Warning only | cfg.Warnings — consistent with path convention in Phase 3 | |

**User's choice:** Hard error for duplicate (from, to) pairs

---

| Option | Description | Selected |
|--------|-------------|----------|
| Single validateSyncPairs function | One function called from buildAndValidate | |
| Split: field validation + cycle detection separate | validateSyncPairFields + detectSyncCycles — easier to test each check in isolation | ✓ |

**User's choice:** Split into two functions: `validateSyncPairFields` and `detectSyncCycles`, both called from `buildAndValidate`

---

## Agent's Discretion

- Exact field ordering on `SyncPairConfig` struct
- Internal DFS helper function names and signature
- Whether DFS operates on `[]SyncPairConfig` directly or builds an adjacency map first
- Test table structure

## Deferred Ideas

- Name-reference validation (from/to must match defined [[remote]] names) — after Phase 7 two-file merge
- Fan-out execution — Phase 15
- MergeSyncPair call sites — Phase 7
- refs filtering beyond globs — post-v2.0 (POST-05)
