# Architecture Research

**Domain:** Go CLI tool evolution (v1 to v2)
**Researched:** 2026-04-01
**Confidence:** HIGH

## v1 Current State

### System Overview

```
                        ┌─────────────────────────────────┐
                        │          main.go                │
                        │     cmd.Execute(version)        │
                        └───────────────┬─────────────────┘
                                        │
                        ┌───────────────▼─────────────────┐
                        │         pkg/cmd/root.go         │
                        │   newRootCmd + Register calls   │
                        └───────────────┬─────────────────┘
                                        │
          ┌─────────┬─────────┬─────────┼─────────┬──────────┬──────────┐
          ▼         ▼         ▼         ▼         ▼          ▼          ▼
     workspace    repo      git     worktree   branch   workgroup   display
     Register   Register  Register  Register  Register  Register    output
                                                                    cmdutil
          │         │         │         │         │          │
          └─────────┴─────────┴────┬────┴─────────┴──────────┘
                                   ▼
                            ┌─────────────┐
                            │ pkg/config  │
                            │ Load/Save   │
                            └──────┬──────┘
                                   ▼
                            ┌─────────────┐
                            │  pkg/toml   │
                            └─────────────┘
```

**v1 data flow:** cobra dispatch -> load config -> resolve repos -> execute in parallel -> collect reports -> write output.

**v1 packages (16):** `cmd`, `config`, `repo`, `git`, `gitutil`, `workspace`, `worktree`, `branch`, `workgroup`, `parallel`, `display`, `output`, `cmdutil`, `toml`, `testutil`.

Each domain package exposes `Register(*cobra.Command)`. Config is `.gitw` + `.gitw.local` TOML, discovered by walking up from CWD. Parallel execution via `parallel.RunFanOut[T,R]` with `runtime.NumCPU()` workers.

---

## v2 System Overview

```
                         ┌──────────────────────────────────┐
                         │           main.go                │
                         │      cmd.Execute(version)        │
                         └────────────────┬─────────────────┘
                                          │
                         ┌────────────────▼─────────────────┐
                         │          pkg/cmd/root.go         │
                         │    newRootCmd + Register calls    │
                         └────────────────┬─────────────────┘
                                          │
    ┌──────────┬──────────┬───────────────┼───────────────┬──────────┬──────────┐
    ▼          ▼          ▼               ▼               ▼          ▼          ▼
 workspace  workstream   repo          remote           git       branch     migrate
 Register   Register   Register       Register        Register   Register   Register
    │          │          │               │               │          │          │
    │          │          │          ┌────┴────┐          │          │          │
    │          │          │          ▼         ▼          │          │          │
    │          │          │      provider  branchrule     │          │          │
    │          │          │                                │          │          │
    │          ├──────────┤                               │          │          │
    │          ▼          │                               │          │          │
    │        agents       │                               │          │          │
    │     (generators)    │                               │          │          │
    │                     │                               │          │          │
    └──────┬──────────────┴────────────┬──────────────────┴──────────┘          │
           ▼                           ▼                                        │
      ┌─────────┐               ┌─────────────┐                                │
      │ stream  │               │ pkg/config  │◄───────────────────────────────┘
      │ (.gitw- │               │ Load/Merge  │
      │  stream)│               └──────┬──────┘
      └─────────┘                      ▼
                                ┌─────────────┐
           ┌─────────┐         │  pkg/toml   │
           │  state   │         └─────────────┘
           │ (.git/   │
           │  state)  │         ┌─────────────┐     ┌─────────────┐
           └─────────┘         │   hook       │     │   parallel  │
                                │ (pre-push)  │     │  (fan-out)  │
                                └─────────────┘     └─────────────┘
```

---

## v2 Package Map

### New packages (9)

| Package | Responsibility | Dependencies | Milestone |
|---------|---------------|--------------|-----------|
| `pkg/workstream` | Workstream CRUD, ship, close, worktree add; replaces workgroup + worktree | config, stream, hook, agents, gitutil, parallel | M7 |
| `pkg/remote` | `git w remote add/list/status/remove`; wizard flow, git remote upsert | config, provider, state, gitutil | M4 |
| `pkg/branchrule` | Pure branch rule evaluation engine; glob matching, criteria evaluation | none (zero I/O) | M2 |
| `pkg/hook` | `reconcileHooks`, pre-push hook evaluation, `git-w hook pre-push` subcommand | config, stream, branchrule | M6 |
| `pkg/agents` | SpecFramework interface, GSDFramework, AGENTS.md/CONTEXT.md generators, registry | config types, stream types | M9 (interface M1, generators M7, commands M9) |
| `pkg/stream` | `.gitw-stream` manifest load/save/validate; worktree uniqueness enforcement | config types, toml | M1 |
| `pkg/state` | `.git/git-w-state.json` read/write; per-repo per-remote timestamps | none (stdlib JSON) | M3 |
| `pkg/provider` | Remote API providers (Gitea, GitHub, generic) behind Provider interface | stdlib net/http | M4 |
| `pkg/migrate` | v1-to-v2 migration: DetectV1, ReportPlan, ApplyPlan | config, stream, gitutil | M12 |

### Modified packages (6)

| Package | Changes | Milestone |
|---------|---------|-----------|
| `pkg/config` | New types: `[[workspace]]`, `[[remote]]`, `[[remote.branch_rule]]`, `[[sync_pair]]`, `[[workstream]]`, `[metarepo]` additions (`default_remotes`, `agentic_frameworks`); `[[repo]]` additions (`track_branch`, `upstream`, `remotes`, `branch_override`); two-file field-level merge; `private=true` enforcement; v1 `[[workgroup]]` detection; cycle detection | M1 |
| `pkg/repo` | New fields on `Repo` (`TrackBranch`, `Upstream`, `Remotes`, `BranchOverrides`); `--upstream` filter; env-group resolution | M1, M8 |
| `pkg/git` | `sync` rewritten for multi-remote fan-out with `[[sync_pair]]` routing; cascade resolution; `reconcileHooks` side effect; state file writes | M3 |
| `pkg/cmd` | Wire new packages: workstream, remote, migrate, hook; remove workgroup, worktree, group registrations; repurpose `context` command | M7+ |
| `pkg/branch` | `--from <remote>` flag on checkout; remove `branch default` subcommand | M5 |
| `pkg/display` | Env-group display, `<repo> / <name>` format for Pattern B, remote staleness section | M5 |

### Retired packages (2)

| Package | Replaced by | Milestone |
|---------|------------|-----------|
| `pkg/workgroup` | `pkg/workstream` | M7 |
| `pkg/worktree` | Absorbed into `pkg/workstream` | M7 |

---

## Component Boundaries

### Config and Data Layer

| Component | Owns | Exposes | Consumes | Notes |
|-----------|------|---------|----------|-------|
| `config` | `.gitw` + `.git/.gitw` parsing, merge, validation, save | All config types; `Load()`, `Save()`, merge functions | `toml` | Boundary: only package that touches `.gitw` files |
| `stream` | `.gitw-stream` manifest parsing, validation, atomic write | `Workstream`, `Worktree` types; `Load()`, `Save()` | `toml`, config types | Boundary: only package that touches `.gitw-stream` files |
| `state` | `.git/git-w-state.json` read/write | `State` type; `Load()`, `Save()`, per-remote timestamp methods | stdlib JSON | Boundary: only package that touches state file |
| `toml` | Comment-preserving TOML round-trip | `Marshal`, `Unmarshal`, `UpdatePreservingComments` | go-toml/v2 | Unchanged from v1 |

### Rule Engine Layer

| Component | Owns | Exposes | Consumes | Notes |
|-----------|------|---------|----------|-------|
| `branchrule` | Branch rule evaluation logic, glob matching | `EvaluateRule()`, `BranchInfo`, action types | nothing | Zero I/O. Pure functions only. Testable without filesystem. |

### Domain Command Layer

| Component | Owns | Exposes | Consumes | Notes |
|-----------|------|---------|----------|-------|
| `workstream` | Workstream CRUD, ship pipeline, close/archive, worktree add | `Register()` | config, stream, hook, agents, gitutil, parallel, repo | Largest new package; replaces workgroup + worktree |
| `remote` | Remote management wizard, list, status, remove | `Register()` | config, provider, state, gitutil | Wizard flow uses `output` for interactive prompts |
| `git` (modified) | Multi-remote sync fan-out, exec, commit, status | `Register()` | config, repo, branchrule, state, hook, parallel, gitutil | Sync is the most complex rewrite |
| `branch` (modified) | Branch create/checkout across repos | `Register()` | config, repo, gitutil, parallel | `--from` flag added; `branch default` removed |
| `workspace` (modified) | Workspace create, list | `Register()` | config, agents | Trimmed: init stays, group/context commands removed |
| `migrate` | v1 detection, plan, apply | `Register()` | config, stream, gitutil | One-shot tool; no ongoing runtime role |
| `hook` | Hook installation/removal, pre-push evaluation | `Register()` for `git-w hook pre-push`; `ReconcileHooks()` for internal use | config, stream, branchrule | Called by workstream create, sync, ship, close |

### Infrastructure Layer

| Component | Owns | Exposes | Consumes | Notes |
|-----------|------|---------|----------|-------|
| `provider` | Remote API communication (Gitea, GitHub, generic) | `Provider` interface; `RepoExists()`, `CreateRepo()` | stdlib net/http | Behind interface; new providers added without touching other code |
| `agents` | SpecFramework interface, GSD implementation, MD generators | `SpecFramework`, `FrameworkFor()`, `Generate*MD()` | config types, stream types | Pure generators (no file I/O); command wiring in M9 |
| `parallel` | Bounded fan-out execution | `RunFanOut[T,R]()` | stdlib | Unchanged from v1 |
| `repo` | Repo struct, resolution, filtering | `Repo`, `Filter()`, `FromConfig()` | config | Extended with `TrackBranch`, `Upstream`, env-group resolution |

---

## Data Flows

### 1. Config Loading (v2)

```
.gitw (committed)           .git/.gitw (private)
     │                            │
     ▼                            ▼
  Parse TOML                   Parse TOML
     │                            │
     └──────────┬─────────────────┘
                ▼
      Field-Level Merge
      (remote by name, repo by name,
       sync_pair by from+to, workstream by name)
                │
                ▼
      Validation:
        - private=true only in .git/.gitw
        - [[sync_pair]] cycle detection
        - [[repo]] path = repos/<n> (warn if not)
        - [[workgroup]] detection (error: run migrate)
        - agentic_frameworks validated against registry
                │
                ▼
      MergedConfig (in-memory, read-only for commands)
```

### 2. Cascade Resolution (effective remote list per repo)

```
[metarepo] default_remotes = ["origin", "personal"]
                │
                ▼
[[workstream]] remotes = ["personal"]     (overrides metarepo)
                │
                ▼
[[repo]] remotes = ["origin"]             (overrides workstream)
                │
                ▼
         Effective Remote List
         (innermost wins; used by sync and hook evaluation)
```

### 3. Sync Fan-Out (git w sync)

```
git w sync
    │
    ▼
Load merged config + resolve effective remotes per repo
    │
    ▼
For each repo (parallel via errgroup):
    │
    ├── Fetch phase (parallel across remotes with direction=fetch|both)
    │       │
    │       ▼
    │   track_branch used as pull target for alias repos
    │
    ├── Push phase (per [[sync_pair]], sequential after fetch):
    │       │
    │       ▼
    │   For each (from, to) pair:
    │       Filter refs by [[sync_pair]] refs globs
    │       Evaluate branch rules:
    │           1. [[repo.branch_override]] for this remote (first match)
    │           2. [[remote.branch_rule]] entries (first match)
    │           3. No match -> default allow
    │       Execute: allow/block/warn/require-flag per branch
    │       push_mode=mirror -> --force
    │
    ├── Push local-only WIP branches to permissive remotes
    │
    ├── reconcileHooks(repo) as side effect
    │
    └── Write state file timestamps
            │
            ▼
    Collect results; critical=true failure marks repo failed
            │
            ▼
    Per-remote grouped output with summary line
```

### 4. Workstream Lifecycle

```
git w workstream create <workspace> <name> --repo/--env-group/--worktree
    │
    ├── Create workspaces/<ws>/active/<name>/
    ├── Resolve worktree specs:
    │     --repo <r>:<b>       -> single worktree, name=repo
    │     --env-group <u>:<b>  -> expand all upstream aliases
    │     --worktree k=v,...   -> explicit multi-worktree (Pattern B)
    ├── Validate name + path uniqueness within workstream
    ├── For each worktree: git worktree add <path> -b <branch> from repos/<repo>
    ├── Write .gitw-stream manifest
    ├── Generate workstream AGENTS.md (via agents generators)
    ├── Create .planning/ directory
    ├── reconcileHooks on each affected repo
    ├── Auto-commit (.gitw-stream, AGENTS.md, .planning/)
    └── Run git w context rebuild
                │
                ▼
git w workstream ship
    │
    ├── Validate: all worktrees clean (or warn)
    ├── Optional squash pass (--squash):
    │     Create pre-ship backup branch on personal
    │     Record in .gitw-stream [ship] pre_ship_branches
    │     Soft-reset + single commit per worktree
    ├── Lift push protection:
    │     Add origin to [[workstream]] remotes in .git/.gitw
    │     reconcileHooks on all repos
    ├── --push-all: push worktree branches to origin
    ├── --open-prs: one PR per worktree entry (Pattern B: multiple PRs)
    │     Record URLs in .gitw-stream [ship] pr_urls
    ├── Set status="shipped", shipped_at
    └── Commit updated .gitw-stream
                │
                ▼
git w workstream close
    │
    ├── Verify shipped (prompt if still active)
    ├── git worktree remove for each worktree
    ├── reconcileHooks cleanup on each repo
    ├── Optional branch pruning (prompt per-branch)
    ├── mv active/<name>/ -> archived/<name>/
    ├── Update .gitw-stream: status="archived", closed_at
    ├── Update AGENTS.md to reflect archived state
    ├── Commit all changes
    └── Run git w context rebuild
```

### 5. Push Protection (pre-push hook)

```
Direct git push from worktree
    │
    ▼
.git/hooks/pre-push (installed by reconcileHooks)
    │
    ▼
git-w hook pre-push <remote-name> <remote-url>
    │
    ├── Walk up from $GIT_DIR to find .gitw (workspace root)
    ├── Load merged config
    ├── Resolve current worktree absolute path
    ├── Scan all .gitw-stream manifests in active workstream dirs
    │     Match [[worktree]] path against current worktree path
    │
    ├── No match found -> exit 0 (allow; not in any workstream)
    │
    └── Match found:
          Retrieve [[workstream]] remotes from .git/.gitw
          Is push target remote in the whitelist?
              YES -> exit 0 (allow)
              NO  -> exit 1 (block) + formatted error:
                     workstream name, blocked remote,
                     allowed remotes, suggested commands
```

### 6. Ship Pipeline Detail

```
git w workstream ship [--squash] [--push-all] [--open-prs] [--dry-run]
    │
    ▼
┌────────────────────────────────────────────────┐
│ 1. Dirty Check                                 │
│    For each worktree: git status --porcelain   │
│    Uncommitted changes -> warn (not block)      │
└────────────────────┬───────────────────────────┘
                     ▼
┌────────────────────────────────────────────────┐
│ 2. Squash Pass (--squash or prompted)          │
│    For each worktree with unpushed commits:    │
│      a. Detect divergence from origin/<branch> │
│      b. Push pre-ship backup to personal       │
│      c. Record in .gitw-stream pre_ship_branches│
│      d. Prompt for squash commit message       │
│      e. Soft-reset + commit                    │
└────────────────────┬───────────────────────────┘
                     ▼
┌────────────────────────────────────────────────┐
│ 3. Lift Protection                             │
│    Add "origin" to [[workstream]] remotes      │
│    in .git/.gitw                               │
│    reconcileHooks on all affected repos        │
└────────────────────┬───────────────────────────┘
                     ▼
┌────────────────────────────────────────────────┐
│ 4. Push (--push-all)                           │
│    Scoped sync: push worktree branches to      │
│    origin via sync machinery                   │
└────────────────────┬───────────────────────────┘
                     ▼
┌────────────────────────────────────────────────┐
│ 5. PRs (--open-prs)                            │
│    One PR per [[worktree]] entry               │
│    Pattern B: multiple PRs for same repo       │
│    Record URLs in .gitw-stream [ship] pr_urls  │
└────────────────────┬───────────────────────────┘
                     ▼
┌────────────────────────────────────────────────┐
│ 6. Finalize                                    │
│    Set status="shipped", shipped_at            │
│    Commit updated .gitw-stream                 │
└────────────────────────────────────────────────┘
```

### 7. Agent Context Generation

```
git w context rebuild
    │
    ├── Load merged config
    ├── Resolve active SpecFrameworks from agentic_frameworks
    │     agents.FrameworksFor(cfg.MetaRepo.AgenticFrameworks)
    │
    ├── Generate CONTEXT.md (meta-repo root):
    │     All repos, upstream groupings, workspaces,
    │     active + archived workstreams
    │
    ├── Generate meta-repo AGENTS.md:
    │     git-w prohibitions (always present)
    │     + framework-specific prohibitions (from each fw.ProhibitedActions())
    │     + available commands
    │     + workspace/workstream summary
    │
    ├── For each workspace: generate workspace AGENTS.md:
    │     Description, repo membership, env-group summary,
    │     active/archived workstreams
    │
    ├── For each active workstream: generate workstream AGENTS.md:
    │     Name, workspace, status, worktree table
    │     (name, repo, branch, path, scope for Pattern B)
    │     + framework init instructions (from each fw.InitInstructions())
    │     + cross-modification warning (if repo appears >1 time)
    │
    └── Commit all generated files

git w agent context [--json]
    │
    ├── Resolve scope: CWD -> workstream -> workspace -> meta-repo
    │     Fallback: .git/git-w-state.json active pointer
    │
    └── Emit JSON:
          agentic_frameworks, workstream details,
          worktrees array (name, repo, branch, path, scope, abs_path),
          env_groups (with create_hint),
          capabilities (machine-readable prohibitions),
          commands (available git-w commands)
```

---

## Architectural Patterns

### Pattern 1: Config Cascade Resolution

**What:** Remote lists resolved through a three-level cascade where innermost scope wins: `[metarepo] default_remotes` < `[[workstream]] remotes` < `[[repo]] remotes`.

**When to use:** Anywhere the effective remote list for a repo is needed (sync, hook evaluation, status display).

**Trade-offs:** Simple mental model (innermost wins) but requires callers to always resolve through the cascade, never read raw config values directly. A helper function in `config` or `repo` should encapsulate this.

**Example:**
```go
// EffectiveRemotes resolves the cascade for a given repo within
// an optional workstream context.
func EffectiveRemotes(cfg *Config, repo RepoConfig, wsName string) []string {
    if len(repo.Remotes) > 0 {
        return repo.Remotes
    }
    if ws, ok := cfg.WorkstreamByName(wsName); ok && len(ws.Remotes) > 0 {
        return ws.Remotes
    }
    return cfg.MetaRepo.DefaultRemotes
}
```

### Pattern 2: Self-Contained Workstream Manifests

**What:** Each workstream's state lives in a `.gitw-stream` TOML file inside its directory, not in the root `.gitw`. The manifest is the source of truth for worktree membership, status, and ship state.

**When to use:** Any operation that needs to know a workstream's worktrees, status, or ship state reads the `.gitw-stream` file directly.

**Trade-offs:** Avoids root config bloat and merge conflicts when multiple workstreams change simultaneously. However, operations that need to scan "all active workstreams" (like hook evaluation) must walk `workspaces/*/active/*/` directories and load each manifest. This is acceptable because the number of active workstreams is small (typically < 20).

### Pattern 3: Pure Rule Engine (branchrule)

**What:** `pkg/branchrule` has zero I/O dependencies. It receives a `BranchInfo` struct (name, upstream tracking status, explicit flag) and a slice of rules, and returns the matched action. No filesystem access, no git calls, no config loading.

**When to use:** Branch rule evaluation during sync push phase and hook evaluation.

**Trade-offs:** The caller must construct `BranchInfo` by querying git state (does the branch have an upstream on this remote? is it in a `[[repo.branch_override]]` with `explicit=true`?). This pushes I/O to the edges and keeps the rule engine purely testable with table-driven tests.

**Example:**
```go
type BranchInfo struct {
    Name           string
    HasUpstreamOn  func(remoteName string) bool
    ExplicitOn     func(remoteName string) bool
}

func EvaluateRule(branch BranchInfo, rules []BranchRule, remoteName string) (Action, *BranchRule) {
    // Pure: iterate rules, match criteria, return first match or default allow
}
```

### Pattern 4: Self-Healing Hooks (reconcileHooks)

**What:** `reconcileHooks` is called as a side effect on every `git w sync` run, in addition to explicit calls from `workstream create`, `ship`, and `close`. This means config changes are automatically reflected in hook state without any manual step.

**When to use:** The function is internal to `pkg/hook` but exposed for use by `pkg/workstream` and `pkg/git` (sync). It is idempotent: safe to call any number of times with the same result.

**Trade-offs:** Slight overhead on every sync (scanning `.gitw-stream` manifests). Acceptable because manifest files are small TOML and the scan is bounded by active workstream count. The benefit is that hooks never drift out of sync with config.

### Pattern 5: SpecFramework Interface for Agent Interop

**What:** All framework-specific behavior (GSD prohibitions, init instructions, planning directory detection) is behind a `SpecFramework` Go interface. At v2 launch, only `GSDFramework` is implemented. New frameworks are added by implementing the interface and registering in `knownFrameworks`.

**When to use:** Anywhere AGENTS.md content or `agent context --json` output includes framework-specific information.

**Trade-offs:** Adds an interface where a simpler direct approach could work for a single framework. The benefit is future-proofing: when a second framework is needed, the integration point is already clean. The interface is small (5 methods) so the abstraction cost is low.

---

## Dependency Graph

```
                                    ┌────────┐
                                    │  toml  │
                                    └───┬────┘
                                        │
                        ┌───────────────▼────────────────┐
                        │            config              │
                        │  (types, load, merge, save)    │
                        └───┬──────────┬─────────────────┘
                            │          │
              ┌─────────────▼──┐    ┌──▼───────────┐
              │     stream     │    │     repo      │
              │ (.gitw-stream) │    │ (resolution)  │
              └──┬──────────┬──┘    └──┬────────────┘
                 │          │          │
    ┌────────────▼──┐  ┌───▼──────────▼─────┐
    │   branchrule  │  │       state        │
    │ (pure engine) │  │ (git-w-state.json) │
    │  [no deps]    │  └────────────────────┘
    └───────┬───────┘
            │
    ┌───────▼───────┐    ┌──────────────┐
    │     hook      │    │   provider   │
    │(reconcile +   │    │ (Gitea/GH/   │
    │ pre-push)     │    │  generic)    │
    └───────────────┘    └──────────────┘
            │                    │
    ┌───────▼────────────────────▼──────┐
    │        Domain Commands            │
    │  workstream, remote, git (sync),  │
    │  branch, workspace, migrate       │
    └───────────────┬───────────────────┘
                    │
            ┌───────▼───────┐
            │    agents     │
            │ (generators,  │
            │  framework    │
            │  registry)    │
            └───────────────┘
```

**Key dependency rule:** `agents` depends on config and stream *types* only (from M1), not on `workstream`. It provides pure generator functions that `workstream` and `workspace` call. No circular dependency.

---

## Build Order and Milestone Mapping

| Order | Milestone | Package(s) | Depends On | Rationale |
|-------|-----------|-----------|------------|-----------|
| 1 | M1 | config (v2 types), stream, agents (stub types only) | toml | Everything depends on the config types. Stream is a config-adjacent manifest format. Agent stubs define the SpecFramework interface and GSD placeholder so M1 config loader can validate `agentic_frameworks`. |
| 2 | M2 | branchrule | nothing | Pure engine with zero deps. Must exist before sync can evaluate rules. |
| 3 | M3 | git (sync rewrite), state | config, branchrule | Sync is the backbone operation. State file tracks sync timestamps. |
| 4 | M4 | remote, provider | config, state | Remote management builds on top of config types and needs state for status. |
| 5 | M5 | git (status rewrite), branch (--from) | config, state | Status display uses state file for remote staleness. Branch --from uses remote-fetched refs. |
| 6 | M6 | hook | config, stream, branchrule | Push protection requires config, stream manifests, and branch rule evaluation. |
| 7 | M7 | workstream, workspace (v2) | config, stream, hook, agents, gitutil | Largest milestone. Workstream create/list/status/switch/worktree-add all land here. |
| 8 | M8 | repo (Pattern A/B extensions), workstream (env-group) | config, stream, workstream | Infra patterns extend repo and workstream with --env-group, --branch-map, --worktree. |
| 9 | M9 | agents (full generators + commands) | config, stream | Agent context commands and full AGENTS.md/CONTEXT.md generation. |
| 10 | M10 | workstream (ship) | workstream, hook, provider | Ship pipeline requires protection lift, push, PR creation. |
| 11 | M11 | workstream (close) | workstream, hook, agents | Close requires worktree removal, hook cleanup, archival, context rebuild. |
| - | M12 | migrate | config, stream, gitutil | Parallel after M1. Depends only on config types existing. |

**M7/M9 circular dependency resolution:** `pkg/agents` depends only on config/stream types (available from M1), not on the `workstream` package. During M7, workstream calls `agents.Generate*MD()` functions that already exist from M9. In practice, M7 implements the command that calls the generators, and M9 implements the generators. The build works because:
1. M1 creates the `SpecFramework` interface and `GSDFramework` stub
2. M7 imports `agents` and calls generator stubs (which can return placeholder content during M7 development)
3. M9 fills in the real generator implementations

There is no Go package-level circular dependency at any point.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Reading Raw Config Instead of Cascade

**What people do:** Reading `cfg.MetaRepo.DefaultRemotes` directly instead of resolving through the cascade.
**Why it's wrong:** Misses workstream and repo overrides. A repo in a protected workstream would get the wrong remote list.
**Do this instead:** Always call `EffectiveRemotes(cfg, repo, workstreamName)` or equivalent cascade resolver.

### Anti-Pattern 2: Scanning Root Config for Workstream State

**What people do:** Putting workstream details in root `.gitw` and reading them there.
**Why it's wrong:** `.gitw-stream` is the source of truth. Root config `[[workstream]]` blocks contain only lightweight remote overrides in `.git/.gitw`.
**Do this instead:** Load `.gitw-stream` manifests from workstream directories for worktree membership, status, and ship state.

### Anti-Pattern 3: Hook Evaluation That Calls Git

**What people do:** Having the branch rule engine call `git remote` or read `.git/config` directly.
**Why it's wrong:** Makes the rule engine untestable and creates coupling to git subprocess availability.
**Do this instead:** Construct `BranchInfo` with the required data *before* calling `EvaluateRule`. Push I/O to the caller; keep the engine pure.

### Anti-Pattern 4: Framework-Specific Logic Outside pkg/agents

**What people do:** Scattering GSD-specific strings ("do not run /gsd:new-workspace") throughout workstream, workspace, or command code.
**Why it's wrong:** When a second framework is added, every scattered reference must be found and generalized.
**Do this instead:** All framework-specific content comes from `SpecFramework.ProhibitedActions()` and `SpecFramework.InitInstructions()`. Domain packages call `agents.Generate*MD(cfg, state, frameworks)` and never construct framework-specific strings themselves.

---

## Integration Points

### External Services

| Service | Integration Pattern | Package | Notes |
|---------|---------------------|---------|-------|
| Gitea REST API v1 | HTTP client behind `Provider` interface | `provider` | Also handles Forgejo (API-compatible) |
| GitHub REST API v3 | HTTP client behind `Provider` interface | `provider` | Used for repo existence check and creation |
| Git subprocess | `os/exec` via `gitutil` | `gitutil` | Unchanged from v1; all git operations are subprocess calls |
| GSD framework | `SpecFramework` interface | `agents` | No API coupling; compose through directory convention and AGENTS.md |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| config <-> stream | Stream imports config types for shared TOML patterns | Stream is self-contained for `.gitw-stream`; does not call config.Load |
| hook <-> stream | Hook scans `.gitw-stream` manifests via stream.Load | Hook never writes `.gitw-stream`; only reads for evaluation |
| workstream <-> hook | Workstream calls hook.ReconcileHooks after create/ship/close | Unidirectional: workstream calls hook, not vice versa |
| workstream <-> agents | Workstream calls agents.Generate*MD for AGENTS.md content | Agents provides pure string generators; workstream writes to disk |
| git (sync) <-> branchrule | Sync constructs BranchInfo and calls branchrule.EvaluateRule | Sync owns the I/O; branchrule owns the logic |
| git (sync) <-> hook | Sync calls hook.ReconcileHooks as side effect | Self-healing: every sync refreshes hook state |
| remote <-> provider | Remote wizard calls provider.RepoExists/CreateRepo | Provider is swappable per remote kind |

---

## Build Order Implications for Roadmap

**Phase ordering rationale:**

1. **Config types first (M1)** because every other package imports config types. No useful work can happen without v2 schema types.
2. **Pure engine next (M2)** because branchrule has no deps and is needed by sync (M3) and hook (M6). Building it early lets it be tested in isolation.
3. **Sync before commands (M3)** because sync is the most-used operation and validates the cascade resolution, fan-out, and state file patterns that other milestones reuse.
4. **Remote before workstream (M4)** because remote provisioning must exist before workstream can meaningfully sync to secondary remotes.
5. **Hook before workstream (M6)** because workstream create must call reconcileHooks. Building hooks first means workstream can integrate immediately.
6. **Workstream is the integration milestone (M7)** that ties together config, stream, hook, agents, and gitutil. It should be the last major new-code milestone before the ship/close refinements.
7. **Infra patterns (M8)** extend existing packages rather than creating new ones. They refine workstream and repo for Pattern A/B.
8. **Agent context (M9)** fills in the generator implementations that M7 calls with stubs.
9. **Ship (M10) and close (M11)** are workstream refinements that depend on everything else being solid.
10. **Migration (M12) runs in parallel** after M1 because it only needs config types and stream types.

**Critical path:** M1 -> M2 -> M3 -> M6 -> M7. Everything else can be parallelized or reordered within constraints.

---

## Sources

- `.planning/codebase/ARCHITECTURE.md` -- v1 architecture analysis (HIGH confidence, authoritative)
- `.planning/codebase/STRUCTURE.md` -- v1 file structure (HIGH confidence, authoritative)
- `.planning/v2/v2.md` -- v2 overview and disk layout (HIGH confidence, authoritative)
- `.planning/v2/v2-schema.md` -- Full config schema (HIGH confidence, authoritative)
- `.planning/v2/v2-commands.md` -- Command tree and specifications (HIGH confidence, authoritative)
- `.planning/v2/v2-milestones.md` -- Milestone scope and dependencies (HIGH confidence, authoritative)
- `.planning/v2/v2-remote-management.md` -- Sync, remote, hooks, state design (HIGH confidence, authoritative)
- `.planning/v2/v2-agent-interop.md` -- SpecFramework, generators, context (HIGH confidence, authoritative)
- `.planning/v2/v2-migration.md` -- Migration package design (HIGH confidence, authoritative)
- `.planning/v2/v2-infra-patterns.md` -- Pattern A/B architecture (HIGH confidence, authoritative)

---
*Architecture research for: git-w v2*
*Researched: 2026-04-01*
