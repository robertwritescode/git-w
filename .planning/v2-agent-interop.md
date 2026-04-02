# git-w v2: agent interop

## Philosophy

git-w is opinionated about conventions, not implementations. It does not know
GSD's internals. GSD does not know git-w's internals. They compose because they
agree on directory structure and because git-w's generated `AGENTS.md` files
declare capabilities explicitly enough that agents do not reinvent what git-w
already handles.

Explicit prohibitions with alternatives ("do not do X, instead call Y") are
more reliable than vague guidance.

---

## Spec framework model

git-w's agent interop layer is designed to work with **multiple agentic spec
frameworks** — tools like GSD (get-shit-done), speckit, openspec, or any future
spec-driven development harness that manages `.planning/`-style state in a
meta-repo. At v2 launch, only GSD is supported.

### `SpecFramework` Go interface

All framework-specific behavior is isolated behind a single Go interface in
`pkg/agents`:

```go
// SpecFramework describes the contract that any agentic spec framework must
// satisfy to integrate with git-w's agent context layer.
//
// At v2 launch only GSD is supported. Future frameworks (speckit, openspec,
// or others) satisfy this interface; no other code changes are required.
type SpecFramework interface {
    // Name returns the canonical identifier for this framework (e.g. "gsd").
    // Must match one of the values declared in [metarepo] agentic_frameworks in .gitw.
    Name() string

    // PlanningDirExists reports whether the framework's planning state
    // directory is present at the given workstream or workspace root path.
    PlanningDirExists(rootPath string) bool

    // InitInstructions returns the human- and agent-readable string that
    // explains how to initialize this framework's planning state inside a new
    // workstream directory. Embedded verbatim in workstream AGENTS.md.
    InitInstructions(workstreamPath string) string

    // ProhibitedActions returns a list of actions the framework must not
    // perform inside a git-w-managed environment. Each item is a (action,
    // alternative) pair. Surfaced in meta-repo AGENTS.md prohibition section.
    ProhibitedActions() []ProhibitedAction

    // WorkspaceCreationProhibited reports whether this framework has its own
    // workspace/project scaffolding that must be suppressed when running
    // inside a git-w workstream. When true, git-w adds an explicit prohibition
    // for the framework's workspace-creation command to the generated AGENTS.md.
    WorkspaceCreationProhibited() bool
}

// ProhibitedAction is a (what, why, alternative) triple surfaced in AGENTS.md.
type ProhibitedAction struct {
    Action      string // "do not run git worktree add/remove directly"
    Reason      string // optional context
    Alternative string // "Use git w workstream create / git w workstream close"
}
```

### Framework registry

`pkg/agents` maintains a registry of known frameworks:

```go
var knownFrameworks = map[string]SpecFramework{
    "gsd": GSDFramework{},
    // Future entries: "speckit": SpeckitFramework{}, "openspec": OpenspecFramework{}, ...
}

// FrameworkFor returns the SpecFramework for the given name, or an error if
// the name is not registered. Called at context-rebuild time for each entry
// in [metarepo] agentic_frameworks in the loaded config.
func FrameworkFor(name string) (SpecFramework, error)

// FrameworksFor resolves a slice of framework names to their implementations.
// Returns an error on the first unrecognised name.
func FrameworksFor(names []string) ([]SpecFramework, error)
```

New frameworks are added by implementing `SpecFramework` and registering in
`knownFrameworks`. No changes to any other package are required.

### `GSDFramework` — v2 launch implementation

`GSDFramework` implements `SpecFramework` for GSD. It encapsulates all
GSD-specific knowledge currently embedded in the generator functions:

- `Name()` → `"gsd"`
- `PlanningDirExists` → checks for `.planning/` directory
- `InitInstructions` → returns GSD `/gsd:new-project` invocation with
  `--auto` flag pointed at the workstream directory
- `ProhibitedActions` → the five GSD-specific prohibitions (workspace
  scaffolding, worktree lifecycle, push protection bypass, sync, PR creation)
- `WorkspaceCreationProhibited` → `true` (GSD has `/gsd:new-workspace` and
  `/gsd:new-project` scaffolding that must not fire inside a workstream)

---

## Config-declared frameworks

The active spec frameworks for a meta-repo are declared in `.gitw`:

```toml
[metarepo]
name                = "platform-work"
default_remotes     = ["origin", "personal"]
agentic_frameworks  = ["gsd"]   # slice — multiple frameworks may be active
                                 # e.g. ["gsd", "speckit"] if both are in use
```

`agentic_frameworks` is a string slice validated at load time. Each entry is
checked against the registry of known frameworks. Any unknown value is a
load-time error with an actionable message listing the known values. Omitting
the field defaults to `["gsd"]` for backward compatibility with early v2
configs.

A meta-repo may legitimately declare more than one framework — for example,
when different product areas or workspaces use different spec-driven tooling.
The set of resolved frameworks drives:
- Which prohibition blocks are embedded in `AGENTS.md` files (union of all
  active frameworks' `ProhibitedActions`)
- Whether `.planning/` init instructions appear in workstream `AGENTS.md`
  (any framework where `PlanningDirExists` returns true)
- What `InitInstructions` text is emitted (one block per active framework)
- How `PlanningDirExists` is evaluated (checked per framework in order)

---

## Three-level `AGENTS.md` strategy

### Meta-repo `AGENTS.md` — "what is this environment?"

Critical section always present verbatim:

```markdown
## What git-w manages — do not do these manually

- **Worktree lifecycle**: do not run `git worktree add/remove` directly.
  Use `git w workstream create` and `git w workstream close`.
- **Push protection**: pre-push hooks enforce allowed remotes during WIP.
  Do not attempt to bypass hooks or force-push to origin from a workstream.
- **Sync**: do not push directly to remotes. Use `git w sync` or
  `git w workstream ship`.
- **Workspace creation**: do not use GSD's `/gsd:new-workspace`,
  `/gsd:new-project` workspace scaffolding, or any equivalent command from
  another tool. git-w creates workstreams; GSD initializes `.planning/` inside
  them via `/gsd:new-project` pointed at the workstream directory.
- **PR creation**: do not open PRs directly.
  Use `git w workstream ship --open-prs`.
```

The framework-specific prohibition items (workspace creation command names,
init instructions) are generated by the active `SpecFramework` implementation.
The git-w-owned prohibitions (worktree lifecycle, push protection, sync, PR
creation) are always present regardless of framework.

Also contains: all available git-w commands with descriptions, workspace and
active workstream list, env-group summary.

### Workspace `AGENTS.md` — "what is this product area?"

Contains workspace description, repo membership, env-group summary for any
upstream repos, active workstreams with goals and repo/branch breakdown,
archived workstreams (names and paths), conventions stub.

### Workstream `AGENTS.md` — "what is my current task?"

Contains workstream name, workspace, goal, status; worktree table (repo, `name`
if multi-worktree, `track_branch` if alias, feature branch, `scope` if set,
relative path); `.planning/` path and framework initialization instructions
(from `SpecFramework.InitInstructions`); how to work across multiple worktrees
in one session; explicit "when done, call `git w workstream ship`; then call
`git w workstream close`"; explicit "do not open PRs directly."

When a repo appears more than once in the workstream (Pattern B), the table
includes a `Scope` column and a cross-modification warning block is appended:

```markdown
**Important:** `consolidated-infra` has multiple worktrees in this workstream.
Each worktree must only modify files within its declared scope:
- `dev` (feat/TICKET-123-dev): modify only `environments/dev/`
- `prod` (feat/TICKET-123-prod): modify only `environments/prod/`
Do not cross-modify between worktrees. Each branch merges into `main`
independently.
```

---

## `git w agent context --json`

Emits full context for the current scope (CWD-based: workstream → workspace →
meta-repo root, then state file pointer).

JSON output includes:
- `workstream`: current workstream details, worktrees array with `name`,
  `repo`, `branch`, `path`, `scope`, `abs_path`
- `env_groups`: env-group definitions with `create_hint` per group
- `capabilities`: machine-readable equivalent of the AGENTS.md prohibition
  section
- `commands`: available git-w commands
- `agentic_frameworks`: the active framework names (e.g. `["gsd"]`)

For Pattern B multi-worktree repos, each worktree entry includes `name` and
`scope` fields:

```json
{
  "agentic_frameworks": ["gsd"],
  "workstream": {
    "worktrees": [
      {
        "name": "dev",
        "repo": "consolidated-infra",
        "branch": "feat/TICKET-123-dev",
        "path": "dev",
        "scope": "environments/dev",
        "abs_path": "/Users/robert/platform/workspaces/platform-infra/active/TICKET-123/dev"
      }
    ]
  }
}
```

For Pattern A env-groups, the `create_hint` field gives agents the correct
workstream creation incantation without requiring alias enumeration:

```json
{
  "env_groups": [
    {
      "name": "infra",
      "upstream_url": "https://github.com/work-org/infra",
      "aliases": [
        { "name": "infra-dev",  "track_branch": "dev" },
        { "name": "infra-test", "track_branch": "test" },
        { "name": "infra-prod", "track_branch": "prod" }
      ],
      "create_hint": "git w workstream create <workspace> <n> --env-group infra"
    }
  ]
}
```

---

## `git w context rebuild`

Regenerates and commits:
- `CONTEXT.md` at meta-repo root: all repos (with upstream groupings), all
  workspaces, all active workstreams, all archived workstreams.
- `AGENTS.md` at meta-repo root, each workspace root, and each workstream
  directory.

The active `SpecFramework` (resolved from config at rebuild time) drives the
prohibition content and init instructions in generated `AGENTS.md` files.

Idempotent. Called automatically by `workstream create`, `workstream close`,
and `workspace create`.

---

## GSD interop specifics

When GSD is invoked inside a workstream directory it sees:

```
workspaces/platform-infra/active/INFRA-42/
  AGENTS.md       <- GSD reads this; knows git-w manages worktrees and push
  .planning/      <- GSD reads/writes here
  .gitw-stream    <- GSD can read for context; does not write
  infra-dev/      <- GSD executor works here (normal repo directory)
  infra-test/     <- GSD executor works here
  infra-prod/     <- GSD executor works here
```

GSD's `/gsd:new-project` skips workspace scaffolding because `.planning/`
already exists and `AGENTS.md` explicitly prohibits GSD from creating
workspaces, worktrees, or opening PRs.

GSD manages the planning lifecycle inside the workstream. git-w manages the git
lifecycle. They compose through directory convention and the AGENTS.md contract,
not API coupling.

This behavior is entirely contained in `GSDFramework`. A future framework would
implement the same interface with its own directory conventions and prohibition
set — git-w's core logic does not change.

---

## `pkg/agents` package

All generator functions are pure (no file I/O):

- `GenerateMetaRepoAgentsMD(cfg Config, state State, fws []SpecFramework) string`
- `GenerateWorkspaceAgentsMD(cfg Config, ws Workspace, state State, fws []SpecFramework) string`
- `GenerateWorkstreamAgentsMD(cfg Config, stream Workstream, fws []SpecFramework) string`
- `GenerateContextMD(cfg Config, state State) string`

The `fws []SpecFramework` parameter replaces all previously hard-coded
GSD-specific strings. Callers resolve frameworks from config using
`FrameworksFor` before calling generators. When multiple frameworks are
active, prohibition sections and init instructions are emitted once per
framework in declaration order, clearly labelled.

**Tests:** generator functions must produce:
- Required prohibition strings (git-w-owned, invariant across frameworks)
- Framework-specific prohibition strings (from each `fw.ProhibitedActions()`)
- Command references
- Env-group `create_hint` values
- Scope-boundary warnings for Pattern B workstreams
- Framework init instructions (from each `fw.InitInstructions()`)
- `agentic_frameworks` slice in JSON output

**Framework registry tests:**
- `FrameworkFor("gsd")` returns `GSDFramework{}`
- `FrameworkFor("unknown")` returns an error with actionable message
- `FrameworksFor(["gsd"])` returns `[]SpecFramework{GSDFramework{}}`
- `FrameworksFor(["gsd", "unknown"])` returns an error on the unknown entry
- `GSDFramework` satisfies all `SpecFramework` contract requirements
