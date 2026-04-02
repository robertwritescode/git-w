# Pitfalls Research

**Domain:** v2 config schema and loader — adding new TOML block types to an existing Go CLI
**Researched:** 2026-04-02
**Confidence:** HIGH

Sources: direct reading of `pkg/toml/preserve.go`, `pkg/config/loader.go`, `pkg/config/config.go`,
`.planning/v2/v2-schema.md`, `.planning/v2/v2-migration.md`, `.planning/codebase/CONCERNS.md`,
`.planning/REQUIREMENTS.md` (CFG-01 through CFG-12).

---

## Critical Pitfalls

### Pitfall 1: `UpdatePreservingComments` Silently Swallows Failures on New Block Types

**What goes wrong:**
`applySmartUpdate` in `pkg/toml/preserve.go:77-83` already has a known silent-failure path: when
`smartUpdate` returns an error, the function returns `newBytes, nil` — discarding comments with no
user-visible warning. This was acceptable when only `[workspace]`, `[repos]`, `[groups]` existed.
v2 adds `[[workspace]]`, `[[remote]]`, `[[remote.branch_rule]]`, `[[sync_pair]]`, `[[workstream]]`
— array-of-tables (AoT) types the existing string-matching algorithm has never seen. Every one of
these will cause `smartUpdate` to fail silently, stripping all user comments from the config file on
the first write after M1.

**Why it happens:**
The current `findSectionBounds` uses a regex that matches `[section]` headers (`^\[section\]\s*$`).
TOML array-of-tables uses `[[section]]` headers. These are distinct patterns; the existing regex
does not match AoT headers. When any of the new block types are written and then `Save` is called,
the `smartUpdate` path fails to locate the section, falls through to the silent-return branch, and
the caller gets `newBytes` — correctly structured TOML, but all user comments gone.

**How to avoid:**
Before writing a single line of the new schema structs, add AoT handling to `findSectionBounds` and
test it explicitly. The regex `^\[\[` + `regexp.QuoteMeta(section)` + `\]\]` must be tried
alongside the existing `^\[` variant. Also fix the silent error: when `smartUpdate` fails, log at
minimum a debug-level warning rather than silently discarding the failure. The fix is two lines
(add `fmt.Fprintf(os.Stderr, "warning: comment preservation failed for %s\n", ...)`) but must be
done before any AoT sections are introduced.

Write a golden-file test: load a fixture `.gitw` that contains comments above `[[remote]]` and
`[[workspace]]` blocks, run `UpdatePreservingComments`, and assert the output matches the expected
file byte-for-byte. Fail loudly on mismatch rather than `assert.Contains`.

**Warning signs:**
- Any `[[double-bracket]]` block in the new schema without a corresponding `TestUpdatePreservingComments_AoT*` test
- The phrase `return newBytes, nil` in `applySmartUpdate` without a preceding warning/log
- Tests for `UpdatePreservingComments` that only use `assert.Contains` (not golden-file comparison)

**Phase to address:**
Phase 11 (CFG-12: `UpdatePreservingComments`) — but the AoT regex fix must land before or alongside
Phase 4 (CFG-04: `[[remote]]`) because that is the first AoT block to be written. If CFG-12 is
Phase 11 and CFG-04 is Phase 4, the executor must either fix the regex in Phase 4 or document that
comment preservation is broken until Phase 11.

---

### Pitfall 2: `[[remote.branch_rule]]` Comment Anchors Collide When Multiple Remotes Exist

**What goes wrong:**
`extractCommentAnchors` builds an identity for each comment from `anchorIdentity`, which for
subsection headers returns the subsection name and for key-value lines returns
`currentSubsection + "." + key`. When two `[[remote]]` blocks both have `[[remote.branch_rule]]`
entries with the same field names (e.g., `pattern`, `action`), all anchor identities for the second
remote's `branch_rule` entries collide with the first remote's. Comments from `origin`'s
`[[remote.branch_rule]]` get injected into `personal`'s `[[remote.branch_rule]]` output, or
disappear entirely after the second one consumes the anchor from the lookup table.

The `buildAnchorLookup` function uses a flat `map[string][]string` keyed by anchor identity.
`delete(lookup, id)` is called after the first injection. So the second block with the same key
gets no comment injected. For `[[remote.branch_rule]]` entries, the anchor identity is something
like `personal.pattern` — which is the same string regardless of which `[[remote]]` block the rule
belongs to.

**Why it happens:**
The algorithm was designed around a single instance of each section. TOML arrays-of-tables
(repeated `[[block]]` headers) are structurally different: the same key name appears multiple times
in different instances of the same block type. The flat anchor identity map has no mechanism to
distinguish "the `pattern` key in the first `[[remote]]`" from "the `pattern` key in the second
`[[remote]]`."

**How to avoid:**
The anchor algorithm needs to namespace identities by AoT index. Instead of
`currentSubsection + "." + key`, use `currentSubsection + "[" + index + "]" + "." + key` where
index is the ordinal position of the enclosing `[[block]]` header. This is a non-trivial change to
`extractCommentAnchors` and `injectSectionComments` that requires tracking `[[block]]` headers as
context-switching markers, not just ordinary subsection headers.

Alternatively (simpler): accept that comment preservation for repeated AoT blocks is best-effort,
document this limitation explicitly, and only preserve comments on single-instance blocks
(`[metarepo]`, top-level `[[workspace]]`) and on entire AoT sections as a unit (preserve the
region between `[[remote]]` headers, not individual keys within).

Write a test: two `[[remote]]` blocks each with a comment on `pattern`, call
`UpdatePreservingComments`, assert both comments survive.

**Warning signs:**
- `anchorIdentity` returning the same string for keys in different AoT instances
- Comment preservation tests that only have a single `[[remote]]` block
- `delete(lookup, id)` consuming an anchor before the second AoT instance processes it

**Phase to address:**
Phase 11 (CFG-12) — but must be evaluated early. Recommend: define the limitation in Phase 4 tests,
implement the fix (or explicit limitation documentation) in Phase 11.

---

### Pitfall 3: Two-File Merge Treats Absent Fields as Zero-Value Overrides

**What goes wrong:**
The spec says `.git/.gitw` wins on field-level conflicts (innermost wins). The natural implementation
is `MergeRemote(base, override Remote) Remote` where for each exported field, if the override value
is non-zero, use it; otherwise use base. This is correct for strings and booleans, but wrong for
pointer types, slices, and deliberately-zero values.

Example: `critical = false` is a deliberate choice in `.git/.gitw`. But `false` is the Go zero
value for bool. The merge function can't distinguish "user explicitly set critical=false in private
config" from "critical was not set in private config." The result: the field is always taken from
the base `.gitw`, ignoring the override. The user's private config value is silently discarded.

Same problem for `direction = ""` (empty string) in `.git/.gitw` — a deliberately reset field
vs. an absent field. And for `refs = []` in `[[sync_pair]]` — an empty slice could mean "sync
nothing" (intentional override) or "field not set" (defer to base).

**Why it happens:**
Go's zero-value conflation is the classic config-merge trap. It is easy to write
`if override.Critical { result.Critical = override.Critical }` but this is wrong: it treats
`false` as "not set." The correct check is "is this field present in the source file?" which
requires either a TOML decoder with presence tracking, or pointer types.

**How to avoid:**
Use pointer types (`*bool`, `*string`) for all fields where the zero value is semantically valid in
`.git/.gitw`. When a pointer is `nil`, the field was absent in that file and the base value wins.
When a pointer is non-nil (even pointing to a zero value), the override file's value wins. This is
the pattern already used in `WorkspaceMeta` (`AutoGitignore *bool`, `SyncPush *bool`) — apply it
consistently to all mergeable fields in `Remote`, `Repo`, `SyncPair`, `Workstream`.

Slice fields (`refs`, `remotes`, `branch_rules`) are trickier: a `nil` slice vs. an empty slice
must be distinguishable. Use `*[]string` and treat `nil` as "not set."

Write table-driven tests for every merge case: base-only, override-only, both-present, both-absent,
and especially "override explicitly sets zero value" (`false`, `""`, `[]`).

**Warning signs:**
- `MergeRemote` (or equivalent) using `if override.Field != zero { result.Field = override.Field }`
- Config structs for mergeable types using `string` instead of `*string` for optional fields
- Tests for two-file merge that never test "override sets a field to false/empty"

**Phase to address:**
Phase 7 (CFG-07: two-file merge) — but the struct design decision (pointer vs. value types) must be
made in Phase 1 (CFG-01, `[[workspace]]` struct) and applied consistently throughout M1. Changing
from `string` to `*string` after several phases are implemented requires touching all test
fixtures.

---

### Pitfall 4: Cycle Detection Misses Indirect Cycles in `[[sync_pair]]` Graph

**What goes wrong:**
A `[[sync_pair]]` graph with only direct cycles is easy to detect: if `from="A" to="B"` and
`from="B" to="A"` both exist, that is a direct 2-cycle. But the spec says cycle detection runs at
load time on the full graph, and users can construct indirect cycles:

```toml
[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "personal"
to   = "backup"

[[sync_pair]]
from = "backup"
to   = "origin"    # indirect cycle: origin → personal → backup → origin
```

A naive implementation that checks only direct `(from, to)` pairs misses the 3-node cycle. Worse,
sync fans out in parallel: if the cycle is detected only at runtime (not load time), a sync
operation triggers `origin → personal`, `personal → backup`, and `backup → origin` simultaneously
— potentially creating a mirroring loop until storage is exhausted.

**Why it happens:**
Developers test cycle detection with the simple 2-node case and declare it done. The indirect case
requires graph traversal (DFS or topological sort), which is non-obvious to write for a config
loader where cycles are typically impossible.

**How to avoid:**
Implement load-time cycle detection using DFS with a visited-and-on-stack set (standard directed
graph cycle detection). The graph nodes are remote names; edges are `(from, to)` pairs from
`[[sync_pair]]` blocks. If DFS visits a node already on the current path stack, a cycle exists.
Include the full cycle path in the error message:

```
config error: [[sync_pair]] creates a sync cycle: origin → personal → backup → origin
Remove the pair that closes the loop.
```

Edge cases to test: single-node self-loop (`from="A" to="A"`), 2-node direct, 3-node indirect,
a graph with no cycle but multiple paths (DAG), and a graph with a cycle where one remote is never
a `from` (it's only a destination — valid).

**Warning signs:**
- Cycle detection implemented as `seenPairs[from+to]` without graph traversal
- Tests with only 2-node cycles
- Missing test case: `A→B→C→A` indirect cycle with 3 pairs

**Phase to address:**
Phase 5 (CFG-05: `[[sync_pair]]` with cycle detection). The algorithm must be graph-based from
the start, not patchable from a simple pair-check later.

---

### Pitfall 5: v1 `[[workgroup]]` Detection Fires on Partially-Migrated Configs

**What goes wrong:**
The v1 detection check fires when `[[workgroup]]` blocks exist. The spec says the error must be
"actionable" — it directs users to `git w migrate`. But users who are mid-migration (have run
`git w migrate --apply` partially) may have a config that is in transition: some workgroups
converted to workstreams, some still in `[[workgroup]]` format. The error fires on every `git w`
invocation, making the tool unusable during migration.

A second failure mode: the TOML key for v1 workgroups is `[workgroup]` (as a map, matching the
existing `WorkgroupConfig` struct and the `localDiskConfig.Workgroups` field) or `[[workgroup]]`
(as AoT). If the detection check uses `toml.Unmarshal` into a struct with a `Workgroups` field, it
silently succeeds for `.gitw.local` configs (which legitimately have `[workgroup.*]` keys in v1).
The detection logic must check the shared `.gitw` file only, not the local override.

**Why it happens:**
v1 detection is written as a simple "does this field exist" check without considering which file
is being checked, or what partial-migration state looks like. The error is designed for a clean
pre-migration state; it becomes an obstacle post-migration.

**How to avoid:**
The detection check must:
1. Run only on the primary `.gitw` file, not `.git/.gitw` or `.gitw.local`.
2. Distinguish between `[[workgroup]]` (v1 AoT syntax for workgroups in shared config) and
   `[workgroup.name]` (v1 map-of-tables syntax). Both should trigger the error.
3. Include the specific workgroup names in the error message so the user knows exactly what to
   migrate: `"v1 config detected: [[workgroup]] blocks found: auth-refactor, data-pipeline — run
   'git w migrate' to upgrade."`
4. During migration itself (inside `ApplyPlan`), the detection check must be bypassed or the tool
   cannot write the config mid-migration.

Write tests: v1 workgroup in `.gitw` (should error), v1 workgroup only in `.gitw.local` (should NOT
error — .local is separate concern), partially-migrated (1 workgroup removed of 2, should still
error naming only the remaining one).

**Warning signs:**
- Detection using `cfg.Workgroups != nil` rather than scanning the raw TOML for the key
- No test for detection running on `.gitw.local` without error
- Error message that says "v1 config detected" without naming which workgroups remain

**Phase to address:**
Phase 10 (CFG-10: v1 detection). Detection logic must be strictly scoped to the primary config
file. M12 (`git w migrate`) depends on this detection being accurate without side effects.

---

### Pitfall 6: `repos/<n>` Path Convention Enforcement Breaks Existing Fixture Configs

**What goes wrong:**
CFG-03 enforces the `repos/<n>` path convention: repos not at `repos/<n>` produce a warning.
The existing codebase's test fixtures — both in `pkg/config/loader_test.go` and scattered across
command-level tests — use paths like `apps/frontend`, `services/backend`, `./myrepo`, `./repo1`,
`infra/dev`, `infra/test`. If CFG-03 enforcement turns the warning into a hard error, or if the
warning fires as output noise during other tests, every existing test that uses non-`repos/<n>`
paths breaks.

Even as a warning-only check: if the warning is written to `cmd.OutOrStdout()` inside `Load`,
tests that assert exact stdout output will fail. If the warning goes to stderr, tests using
`s.ExecuteCmd` that don't capture stderr will silently pass while hiding the warning.

**Why it happens:**
Path convention enforcement feels like a simple string-prefix check, so it gets added to `Load`
as a validate step. But `Load` is called by every command in every test, and the existing test
fixtures predate `repos/<n>`. The validator doesn't distinguish "old v1 path we need to warn
about" from "v2 test fixture that happened to use an arbitrary path."

**How to avoid:**
Separate concerns:
- `validateRepoPaths` (existing, in `Load`) rejects clearly wrong paths (absolute, escaping root).
  Do not change this.
- `warnV1Paths` is a new, separate function called only by the `Load` caller when displaying
  output to users — not from inside the loader itself. Commands can call `config.WarnV1Paths(cfg,
  cfgPath, cmd.ErrOrStderr())` after loading.
- Test fixtures that intentionally use non-`repos/<n>` paths should get a comment: `// v1-style
  path; convention warning expected`. New fixtures must use `repos/<n>`.

This keeps `Load` side-effect-free (no warnings, just errors) and makes the warning testable in
isolation.

**Warning signs:**
- `validateRepoPaths` or `Load` calling `fmt.Fprintf` / `output.Writef` to print a warning
- Existing loader tests failing after CFG-03 with "unexpected output" assertion failures
- A single function doing both "validate correctness" and "emit convention warnings"

**Phase to address:**
Phase 3 (CFG-03: `repos/<n>` enforcement). Must not couple the warning to `Load` internals.

---

### Pitfall 7: `[metarepo] default_remotes` Cascade Returns Wrong Winner When All Three Levels Are Absent

**What goes wrong:**
The cascade is `[metarepo] default_remotes` → `[[workstream]] remotes` → `[[repo]] remotes`
(innermost wins). The typical implementation computes "effective remotes" as:
```go
if repo.Remotes != nil { return repo.Remotes }
if workstream.Remotes != nil { return workstream.Remotes }
return metarepo.DefaultRemotes
```
This looks correct but fails when none of the three levels are set. The spec says: "A repo with no
remotes field and no `[metarepo] default_remotes` gets no secondary remotes — only its own
git-configured origin." The implementation must return an empty list, not `nil`, and callers must
handle an empty list as "use git's own remote config" rather than "no remotes (skip sync)."

Second bug: `[metarepo] default_remotes` field in `.git/.gitw` should override (add to) the base.
If `.gitw` has `default_remotes = ["origin"]` and `.git/.gitw` has `default_remotes = ["origin",
"personal"]`, the private config wins and the effective list is `["origin", "personal"]`. But if
the merging logic does field-level replace (private wins on non-zero), then specifying `["origin",
"personal"]` in the private file means the user must repeat "origin" even though it was already in
the base. If the merging does field-level append, then repeating "origin" causes duplicate remotes
in the effective list.

**Why it happens:**
The cascade spec does not explicitly define "what does absent mean at each level" or "replace vs
append semantics for list fields." Implementors fill in the gap with an assumption that turns out
wrong for at least one user scenario.

**How to avoid:**
Define semantics explicitly before writing code:
- `[[repo]] remotes` replaces cascade defaults entirely for that repo (spec confirms: "override
  `[metarepo] default_remotes`").
- `[[workstream]] remotes` replaces cascade defaults entirely for that workstream's repos.
- `[metarepo] default_remotes` in `.git/.gitw` replaces the base `.gitw` value entirely (standard
  field-level merge: private file wins).
- Absent at all levels returns `[]string{}` (empty, not nil). Callers treat empty as "no
  git-w-managed remotes; git's own origin is the only remote."
- Write `ResolveEffectiveRemotes(metarepo, workstream, repo)` as a pure function with a
  table-driven test covering all 8 combinations of present/absent at each level, plus the
  deduplication case.

**Warning signs:**
- `ResolveEffectiveRemotes` returning `nil` instead of `[]string{}`
- No test for the "all three levels absent" case
- No test for "repo specifies remotes; workstream also specifies — repo wins"

**Phase to address:**
Phase 9 (CFG-09: `[metarepo]` cascade). The pure function must have exhaustive tests covering all
8 combinations before any command code uses it.

---

### Pitfall 8: `.gitw-stream` Loading Does Not Validate `name`/`path` Uniqueness Constraints

**What goes wrong:**
The spec defines two uniqueness constraints for `.gitw-stream` manifests:
- `name` must be unique within the workstream (it is the primary key).
- `path` must be unique within the workstream (it is the on-disk key).
- When the same repo appears more than once, `name` is required on all entries for that repo.

A naive `toml.Unmarshal` into a `StreamManifest` struct with a `[]WorktreeEntry` field loads the
data but does nothing to enforce uniqueness. Two `[[worktree]]` entries with the same `name` parse
successfully. The duplicate-name case only surfaces at runtime when a command tries to look up a
worktree by name and gets the wrong one — difficult to debug.

Also: the spec says `name` defaults to `repo` when omitted and the repo appears once. But if the
same repo appears twice and both entries omit `name`, the defaulting logic silently assigns the
same name to both entries — violating uniqueness without an error.

**Why it happens:**
Struct-based TOML unmarshaling validates TOML syntax, not application-level constraints. The
uniqueness check is business logic that must be added explicitly, but it is easy to write a
"working" parser that accepts duplicate names and only notice the bug when lookup returns wrong
data.

**How to avoid:**
`LoadStream(path string) (*StreamManifest, error)` must run a post-parse validation step:
1. Build a `map[string]int` of name occurrences. Any name appearing more than once is an error.
2. Build a `map[string]int` of path occurrences. Any path appearing more than once is an error.
3. For repos appearing more than once: verify all their entries have explicit `name` set.
4. Apply the default: for repos appearing exactly once with `name` omitted, set `name = repo`.
5. After defaulting, re-check uniqueness (the default could create a collision if another entry
   explicitly used the repo name as its `name`).

Table-driven tests: single worktree (valid), two different repos (valid), two entries for same repo
with names (valid), two entries for same repo without names (error), duplicate name across different
repos (error), duplicate path (error), name-default collides with explicit entry (error).

**Warning signs:**
- `LoadStream` that only calls `toml.Unmarshal` without a post-parse validation function
- No test with two `[[worktree]]` entries referencing the same `repo`
- `name` defaulting logic in display code (not in the loader)

**Phase to address:**
Phase 8 (CFG-08: `.gitw-stream` manifest). All constraints must be enforced at load time, not at
use time.

---

### Pitfall 9: `private = true` Enforcement Checks the Wrong Config-File Path

**What goes wrong:**
The spec says: remotes with `private = true` must live in `.git/.gitw`, never in `.gitw`. The
enforcement logic must check which file a remote was loaded from. The natural implementation
checks the field at the final merged config level — but the merged config doesn't remember which
file each `[[remote]]` block came from. The check fires on the merged result, where all remotes
are present regardless of source. The loader cannot tell "this remote came from `.gitw`" vs.
"this remote came from `.git/.gitw`."

Two failure modes:
1. Check always passes (no error) because it runs on the merged config where `private = true`
   remotes are already present and the loader can't distinguish source.
2. Check fires incorrectly because a `private = true` remote defined in `.git/.gitw` gets flagged
   when it should not.

**Why it happens:**
The merge-then-validate pattern loses provenance. The validator needs per-source data, but the
merged struct has thrown that away.

**How to avoid:**
Run the `private = true` check before merging, on the raw parsed content of each file separately.
The sequence is:
1. Parse `.gitw` into `publicCfg`. Check: any `[[remote]]` with `private = true` in `publicCfg`
   is an error. Name the remote: `"remote 'personal' has private=true but is defined in .gitw —
   move it to .git/.gitw"`.
2. Parse `.git/.gitw` into `privateCfg`. No `private` check here.
3. Merge `publicCfg` and `privateCfg` into final config.

The key discipline: **validate each file before merging, not after.** This is a general principle
for all per-file constraints (not just `private`).

**Warning signs:**
- `validatePrivateRemotes` called on the merged `*Config` rather than on the raw parsed
  `publicConfig`
- No test that places `private = true` in `.gitw` (should error) vs. `.git/.gitw` (should not)
- The merge function receiving only one merged config (no per-file types)

**Phase to address:**
Phase 7 (CFG-07: two-file merge). The merge sequence must preserve per-file validation checkpoints.

---

### Pitfall 10: `[[sync_pair]]` Merged by `(from, to)` Pair — Undefined Behavior When Only One Field Matches

**What goes wrong:**
The spec says `[[sync_pair]]` blocks are merged by `(from, to)` pair — both fields together form
the merge key. If `.gitw` has a pair `from="origin" to="personal"` with `refs = ["main"]` and
`.git/.gitw` has a pair `from="origin" to="personal"` with `refs = ["**"]`, the private config
wins for `refs`. That is the intended behavior.

But what if `.git/.gitw` has `from="origin" to="archive"` (a pair not in `.gitw`)? The spec is
silent on this case. Two interpretations:
- Private file can add new pairs not in the shared config (additive).
- Private file can only override existing pairs (non-additive).

If additive, a user on a shared machine can add private sync routes that route all refs to a
personal archive remote — this may be desirable (privacy backup) or undesirable (security concern
for shared workspaces).

A second ambiguity: what if only `from` matches but `to` does not? The pair is treated as a new
entry (no match) under the `(from, to)` composite key. This is correct, but the implementation
must use composite key equality, not partial matching.

**Why it happens:**
Composite merge keys are unusual in config systems. Implementors often implement merge by looping
through the override's items and replacing matching items, but use single-field matching by
accident. A sync pair `from="origin" to="personal"` in `.git/.gitw` would then wrongly match and
replace any pair where `from="origin"`, even if `to` is different.

**How to avoid:**
Use a `syncPairKey` struct `{ From, To string }` as the map key in the merge function. Never
use a single field as the lookup key. The merge function:
```go
func mergeSyncPairs(base, override []SyncPair) []SyncPair {
    index := make(map[syncPairKey]SyncPair, len(base))
    for _, p := range base { index[syncPairKey{p.From, p.To}] = p }
    for _, p := range override { index[syncPairKey{p.From, p.To}] = p } // override wins
    // rebuild ordered slice from index...
}
```
Write tests: same `(from, to)` pair in both files (override wins), different `to` same `from`
(both survive, independent), pair only in private file (additive, included in result).

**Warning signs:**
- Merge loop using `pair.From` as the only lookup key
- No test where two pairs share `from` but differ in `to`
- Spec ambiguity on additive-vs-non-additive not resolved before Phase 7

**Phase to address:**
Phase 7 (CFG-07) for the merge function; Phase 5 (CFG-05) must define the struct with a
`compositeKey()` method so Phase 7 can use it without retrofitting.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip AoT handling in `UpdatePreservingComments` until CFG-12 | Defer preserve.go changes | Comments silently lost from Phase 4 onwards; user confusion; hard to debug | Acceptable only if documented as a known limitation and the silent-swallow error path is fixed to emit a warning |
| Use concrete `bool` instead of `*bool` in mergeable fields | Simpler struct | Zero-value false is indistinguishable from absent; two-file merge silently discards explicit `false` overrides | Never — use `*bool` for all fields where false is a valid override |
| Validate `private = true` on merged config instead of per-file | One validation pass | `private` remotes in `.gitw` are never caught; credentials committed to version control | Never |
| Cycle detection as `seenPairs[from+to]` string check | One-liner | Misses all indirect cycles (3+ nodes); sync loop runs until storage exhausted | Never — use DFS |
| Run `warnV1Paths` inside `Load` | Simpler call site | Warning fires in all tests; breaks tests with exact-output assertions; couples loader to output | Never — warn at call site, not inside loader |
| Skip `(from, to)` composite key for sync pair merge | Simpler map | Single-field match replaces wrong pairs; sync routes silently misconfigured | Never — use composite key from day one |
| Default `agentic_frameworks` silently to `["gsd"]` without surfacing | Fewer required fields | Users don't know the framework is active; surprises on `context rebuild` | Acceptable — but document it in config comments when generating a new `.gitw` |

---

## Integration Gotchas

The "existing system" gotchas specific to adding v2 to this codebase:

| Integration Point | Common Mistake | Correct Approach |
|-------------------|----------------|------------------|
| `pkg/toml/preserve.go` — existing AoT blind spot | Writing new `[[block]]` types and assuming `UpdatePreservingComments` handles them | Verify AoT regex support before adding any `[[block]]`; add a dedicated AoT test in `pkg/toml/` first |
| `pkg/config/loader.go` — `loadMainConfig` | Adding v2 fields to `WorkspaceConfig` and expecting they survive the `prepareDiskConfig` round-trip | `diskConfig` struct controls what gets written; new v2 fields must be added to `diskConfig` AND `WorkspaceConfig` or they are silently dropped on `Save` |
| `pkg/config/config.go` — `ensureWorkspaceMaps` | Forgetting to initialize new v2 slice/map fields to empty non-nil values | Every new `[]Slice` or `map[K]V` field in the merged config must be initialized in `ensureWorkspaceMaps` or nil-pointer panics happen downstream |
| `mergeLocalConfig` — existing local merge | The v2 two-file merge uses `.git/.gitw`, not `.gitw.local`. These are different files with different semantics | `.gitw.local` is for workgroups/context (v1). `.git/.gitw` is for private remotes/sync pairs (v2). Do not reuse `mergeLocalConfig` for the new private-file merge |
| `validateRepoPaths` — called on every `Save` | Adding `[[repo]]` (AoT) to config structs while `validateRepoPaths` iterates `cfg.Repos` (a map) | New AoT `[[repo]]` entries must populate the same `cfg.Repos` map after parsing, or `validateRepoPaths` silently skips them |
| `go-toml/v2` marshal output — key ordering | `UpdatePreservingComments` assumes a stable marshal key order (uses it for section bounds detection) | Pin `go-toml/v2` version in `go.mod`; add a golden-file test for marshal output of the v2 schema structs so version bumps are caught |
| `agentic_frameworks` registry — `pkg/agents` package | Validating `agentic_frameworks` in `Load` before `pkg/agents` is implemented | Phase 1 (CFG-01 + CFG-11) must stub `agents.FrameworkFor` before `Load` calls it, or create a circular dependency or load-time panic |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| `validateRepoPaths` running on every `Save` with 20+ `[[repo]]` blocks | `Save` noticeably slow; each write does 1 `os.Stat` per repo | Acceptable for typical workspaces; do not add additional I/O to the validate path | >50 repos with slow filesystem (network mounts) |
| Cycle detection running full DFS on every `Load` | Load latency increases with many `[[sync_pair]]` blocks | DFS is O(nodes + edges); fine for <100 pairs. No mitigation needed. | Pathological: >500 sync pairs (never realistic) |
| `mapsEqual` in `detectChanges` marshaling every value to compare | `UpdatePreservingComments` is slow on configs with many `[[repo]]` entries | `mapsEqual` calls `Marshal` twice per comparison; for 50 repos this is 100 marshal calls per save | >50 repos, frequent saves (e.g., `repo add` loop) |
| Loading `.gitw-stream` manifest on every command invocation | Startup latency when workstream has many `[[worktree]]` entries | Only load `.gitw-stream` when the command is workstream-scoped | Manifest with >20 worktrees and non-workstream commands |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Accepting `private = true` in `.gitw` (committed file) | API tokens or private remote URLs committed to version control | Error at load time if `private = true` remote is found in `.gitw`; name the remote and tell user which file it must live in |
| Storing `token_env` value directly instead of env var name | If user accidentally copies a token value instead of var name, it is written to `.git/.gitw` which may be logged or displayed | Validate that `token_env` values do not look like token values (no `ghp_`, no 40-char hex strings); warn but do not reject |
| `.git/.gitw` readable by other users on shared machines | Private remotes (tokens, personal remote URLs) exposed to other system users | `Save` for the private config file should write with `0600` permissions; warn if file found with looser permissions at load time |

---

## "Looks Done But Isn't" Checklist

- [ ] **CFG-07 (two-file merge):** Often missing zero-value override tests — verify that `critical = false` in `.git/.gitw` overrides `critical = true` in `.gitw`
- [ ] **CFG-05 (cycle detection):** Often missing indirect cycle test — verify that `A→B→C→A` (3-pair cycle) is detected at load time with a useful error naming the cycle path
- [ ] **CFG-08 (`.gitw-stream`):** Often missing duplicate-name validation — verify that two `[[worktree]]` entries with same `name` produce a load error, not silent data corruption
- [ ] **CFG-09 (cascade):** Often missing "all levels absent" test — verify that a repo with no `remotes`, in a workstream with no `remotes`, and a metarepo with no `default_remotes`, returns `[]string{}` (empty, not nil, not panic)
- [ ] **CFG-10 (v1 detection):** Often missing "detection on `.gitw.local`" test — verify that `[[workgroup]]` in `.gitw.local` does NOT trigger the v1 error (that file is separate from the v2 private config)
- [ ] **CFG-12 (`UpdatePreservingComments`):** Often missing AoT test — verify that comments above `[[remote]]` and `[[workspace]]` blocks survive a round-trip through `Save`
- [ ] **CFG-03 (path convention):** Often breaks existing tests — verify that existing fixtures using `apps/frontend`-style paths do not produce errors (only warnings, at the command layer, not in `Load`)
- [ ] **`private` enforcement:** Often runs on merged config — verify that a remote with `private = true` in `.gitw` produces an error even when the same remote is also in `.git/.gitw`
- [ ] **`diskConfig` coverage:** Often misses new fields — verify that every new v2 field on the config struct appears in `prepareDiskConfig` output and round-trips through `Save`/`Load`
- [ ] **`ensureWorkspaceMaps`:** Often missed for new fields — verify that loading an empty `.gitw` (just `[metarepo]`) produces non-nil slices/maps for all new v2 collection fields

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Comments lost from TOML after AoT write | LOW | `git checkout -- .gitw` if file is committed; `.gitw.bak` if backup strategy implemented |
| Two-file merge discards explicit `false` override | MEDIUM | Use pointer types (`*bool`) in struct definition; migrate existing field values; update all test fixtures |
| Indirect sync cycle not caught at load | HIGH | Stop `git w sync`; add missing check to DFS; meanwhile, add a `break` guard to the sync fan-out loop to detect runtime cycles |
| `.gitw-stream` duplicate names cause lookup corruption | MEDIUM | Add uniqueness validation to `LoadStream`; existing manifests with duplicates need manual fix |
| `private` remote committed to `.gitw` | HIGH | Rotate the token immediately; rewrite git history with `git filter-repo`; add load-time enforcement going forward |
| `warnV1Paths` firing in tests (output noise) | LOW | Move warning out of `Load` into command layer; update tests to capture and assert on stderr separately |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| `UpdatePreservingComments` silent failure on AoT blocks | Phase 4 (CFG-04, first AoT block) or Phase 11 (CFG-12) | Golden-file test: round-trip `.gitw` with `[[remote]]` comments; assert output matches expected byte-for-byte |
| AoT comment anchor collisions (same key in multiple blocks) | Phase 11 (CFG-12) | Test: two `[[remote]]` blocks each with comment on `pattern` field; verify both comments survive round-trip |
| Two-file merge treats absent fields as zero-value overrides | Phase 7 (CFG-07) | Table-driven test: 8 combinations of present/absent per merge field; verify `critical=false` override wins |
| Cycle detection misses indirect cycles | Phase 5 (CFG-05) | Test: 3-pair indirect cycle produces error naming full cycle path; 2-pair direct cycle caught; valid DAG passes |
| v1 detection fires on `.gitw.local` | Phase 10 (CFG-10) | Test: `[[workgroup]]` in `.gitw` → error; in `.gitw.local` → no error |
| `repos/<n>` warning fires inside `Load` | Phase 3 (CFG-03) | Existing loader tests pass without modification; warning only fires via new `WarnV1Paths` call at command layer |
| Cascade returns wrong winner / nil vs empty | Phase 9 (CFG-09) | Table-driven test: all 8 combinations of metarepo/workstream/repo remotes; nil return is a test failure |
| `.gitw-stream` missing uniqueness validation | Phase 8 (CFG-08) | Test: duplicate `name` produces error with the duplicate value named in the message |
| `private = true` enforcement checks wrong file | Phase 7 (CFG-07) | Test: `private=true` in `.gitw` → error naming remote; in `.git/.gitw` → no error |
| `[[sync_pair]]` partial-key merge | Phase 7 (CFG-07) | Test: two pairs sharing `from` but different `to`; both survive merge independently |
| `diskConfig` missing new v2 fields | Every phase adding new struct fields | `Save` + `Load` round-trip test for every new field; any field not in `diskConfig` is caught by the round-trip test failing to see the value after reload |
| `ensureWorkspaceMaps` missing new collection fields | Every phase adding new slice/map fields | Load-empty-config test: assert all new fields are non-nil after loading a minimal `.gitw` |

---

## Sources

- `pkg/toml/preserve.go` — direct inspection of `applySmartUpdate` silent-return at line 80,
  `anchorIdentity` flat-key logic, `extractCommentAnchors` subsection-tracking
- `pkg/config/loader.go` — `loadMainConfig`, `mergeLocalConfig`, `prepareDiskConfig`,
  `ensureWorkspaceMaps` — all checked for v2 extension points
- `pkg/config/config.go` — `WorkspaceConfig`, `WorkspaceMeta` pointer-field pattern
- `.planning/v2/v2-schema.md` — canonical schema for all v2 block types and merge semantics
- `.planning/v2/v2-migration.md` — v1 detection triggers and partial-migration state
- `.planning/codebase/CONCERNS.md` — documented silent error swallow, `interface{}` usage,
  TOML comment preservation fragility
- `.planning/REQUIREMENTS.md` — CFG-01 through CFG-12 traceability to phases

---
*Pitfalls research for: git-w v2 M1 — config schema and loader expansion*
*Researched: 2026-04-02*
