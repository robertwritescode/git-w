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

Also contains: all available git-w commands with descriptions, workspace and
active workstream list, env-group summary.

### Workspace `AGENTS.md` — "what is this product area?"

Contains workspace description, repo membership, env-group summary for any
upstream repos, active workstreams with goals and repo/branch breakdown,
archived workstreams (names and paths), conventions stub.

### Workstream `AGENTS.md` — "what is my current task?"

Contains workstream name, workspace, goal, status; worktree table (repo, `name`
if multi-worktree, `track_branch` if alias, feature branch, `scope` if set,
relative path); `.planning/` path and GSD initialization instructions; how to
work across multiple worktrees in one session; explicit "when done, call
`git w workstream ship`; then call `git w workstream close`"; explicit "do not
open PRs directly."

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

For Pattern B multi-worktree repos, each worktree entry includes `name` and
`scope` fields:

```json
{
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

---

## `pkg/agents` package

All generator functions are pure (no file I/O):

- `GenerateMetaRepoAgentsMD(cfg Config, state State) string`
- `GenerateWorkspaceAgentsMD(cfg Config, ws Workspace, state State) string`
- `GenerateWorkstreamAgentsMD(cfg Config, stream Workstream) string`
- `GenerateContextMD(cfg Config, state State) string`

**Tests:** generator functions must produce:
- Required prohibition strings
- Command references
- Env-group `create_hint` values
- Scope-boundary warnings for Pattern B workstreams
