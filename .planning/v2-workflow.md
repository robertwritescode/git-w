# git-w V2: GSD Workflow Integration Guide

## Purpose

This document describes how to use the get-shit-done (`/gsd-*`) commands to implement
the v2 roadmap defined in `v2-strategy.md` and `v2-spec.md`. It resolves the conceptual
mapping between GSD's project/milestone/phase model and the v2 branching strategy, and
provides concrete step-by-step instructions for each layer of work.

Read this before starting any new milestone or issue.

---

## Conceptual Mapping

| v2-strategy concept | GSD concept |
|---|---|
| The entire v2 effort | **Project** (one GSD project in `.planning/`) |
| V2 M1, M2 … M12 | **Milestones** (`/gsd-new-milestone` per milestone) |
| Each GitHub issue (#36, #37 …) | **Phase** (one GSD phase per issue) |
| Implementation tasks within an issue | **Plans** (GSD plans within each phase) |

---

## Key Decisions

| Decision | Choice | Reason |
|---|---|---|
| GSD branching strategy | `none` | 3-tier branch hierarchy is owned manually; GSD commits to active branch |
| Planning docs home | Issue branches → milestone PR → v2 | Keeps docs colocated with the code that generated them |
| Issue-to-phase naming | `#<N> — <title>` in GSD phase name | Easy cross-reference; no extra tooling needed |
| ROADMAP.md scope | All 12 milestones upfront | `/gsd-progress` gives full v2 picture at any time |
| GSD milestone tagging | Skip / keep branches | PRs are handled manually; must not interfere with Release Please |
| M12 parallel unlock | Explicit when triggered | Update v2-strategy.md Active State table when unlocking |

---

## One-Time Setup (run once, on the `v2` branch)

### 1. Initialize the GSD project

```
/gsd-new-project --auto @.planning/v2.md
```

- GSD detects the existing codebase map in `.planning/codebase/` automatically
- `--auto` mode drives PROJECT.md and requirements from `v2.md` (which references all spec documents) without a lengthy questioning loop
- At the roadmap step, guide the roadmapper to produce 12 GSD milestones × N phases,
  where each milestone maps to a v2 milestone and each phase maps to one GitHub issue
- Set `git.branching_strategy = "none"` and `commit_docs = true` when prompted
- The initial `.planning/` commit lands on `v2`

### 2. Configure GSD settings

```
/gsd-settings
```

Confirm:
- Branching: **None**
- Mode: YOLO
- Research: Yes
- Verifier: Yes

---

## Per-Milestone Workflow

Repeat for M1 → M11 (strictly sequential). M12 may be unlocked in parallel after M1
merges to `v2` — see v2-strategy.md Sequencing section.

### Step A: Create the milestone branch (manual git)

```bash
git checkout v2
git checkout -b v2-m<N>-<slug>   # e.g., v2-m1-config-schema
# Push to remote if not already there:
git push -u origin v2-m<N>-<slug>
```

### Step B: Start the GSD milestone

```
/gsd-new-milestone
```

GSD confirms which phases (issues) belong to this milestone and resets STATE.md.

---

## Per-Issue Workflow

Repeat for each issue within the active milestone (strictly sequential within a milestone).

### Step 1: Create the issue branch (manual git)

Consult `.planning/v2-issue-map.md` for the exact branch name for this issue — do not invent or abbreviate it.

```bash
git checkout v2-m<N>-<slug>
git checkout -b <issue-number>-<kebab-desc>   # e.g., 36-add-workspace-block
```

### Step 2: Discuss the phase (optional, recommended for complex issues)

```
/gsd-discuss-phase <N>
```

Captures design decisions. Reference the relevant spec document from
`.planning/v2.md` as canonical context (use the table of spec documents to
find the right file for the topic at hand).

### Step 3: Plan the phase

```
/gsd-plan-phase <N>
```

- Research, planning, and plan-checking all happen here
- Plans reference the appropriate spec documents (see `.planning/v2.md` for the index) and `.planning/codebase/` for context
- Confirm issue number, title, and scope against `.planning/v2-issue-map.md` before writing the plan
- GSD commits plan artifacts to the active issue branch

### Step 4: Execute the phase

```
/gsd-execute-phase <N>
```

- GSD implements the issue with atomic commits to the issue branch
- Acceptance criteria must include: `mage testfast` passes, `go vet ./...` clean
- Verification agent confirms implementation matches the phase goal

### Step 5: Open the PR (manual)

```bash
gh pr create \
  --title "<issue title>" \
  --body "Closes #<N>" \
  --base v2-m<N>-<slug> \
  --head <issue-number>-<kebab-desc>
```

- PR title matches the GitHub issue title exactly
- `Closes #<N>` auto-closes the issue on merge

### Step 6: Merge the PR (manual)

Merge via GitHub UI or `gh pr merge`. GSD planning docs (phase plans, summaries,
verification reports) merge to the milestone branch and flow to `v2` when the
milestone PR merges.

### Step 7: Update v2-strategy.md Active State (manual)

Update the Active State table to reflect the next issue:
- `Current issue branch` → `none` (until next issue starts)
- `Next issue to implement` → `#<next> — <title>`

---

## Milestone Completion

When all issues in a milestone are done:

### Step A: Complete the milestone in GSD

```
/gsd-complete-milestone
```

At the git tag and branch merge steps: choose **"Keep branches"**. PRs are handled
manually below.

### Step B: Open the milestone PR (manual)

```bash
gh pr create \
  --title "V2 M<N>: <milestone name>" \
  --body "Closes #<issue1> Closes #<issue2> ..." \
  --base v2 \
  --head v2-m<N>-<slug>
```

### Step C: Merge and clean up (manual)

After the milestone PR merges to `v2`:
- Delete the milestone branch
- Update v2-strategy.md Active State to the next milestone
- If M1 just merged and M12 parallel unlock is desired, update the M12 unlock field

---

## Final Cut-Over (after all 12 milestones merge to `v2`)

```bash
gh pr create \
  --title "feat!: git-w v2.0.0" \
  --body "<full v2 feature summary referencing v2-spec.md>" \
  --base main \
  --head v2
```

Release Please detects the breaking change commits accumulated on `v2` and generates
the `2.0.0` release. Homebrew tap auto-updates via GoReleaser.

---

## What GSD Does Not Handle (manual responsibility)

- Branch creation and checkout at all three levels
- PR creation and merging (issue → milestone, milestone → v2, v2 → main)
- `v2-strategy.md` Active State updates
- `mage testfast` and `go vet ./...` must be verified before marking a phase complete
  (include these as acceptance criteria in every plan)
- Release Please tag and cut-over

## What GSD Does Handle

- Research, planning, and verification for each issue
- Atomic commits with per-task granularity on the active branch
- Progress tracking across all 12 milestones in `.planning/`
- Phase summaries and verification reports that travel with the codebase

---

## Tooling Cross-References

| Document | Purpose |
|---|---|
| `.planning/v2.md` | Top-level index — overview, disk layout, motivation, and table of all spec documents |
| `.planning/v2-strategy.md` | Branching hierarchy, sequencing, and Active State |
| `.planning/v2-issue-map.md` | **Authoritative issue and milestone map** — exact GitHub issue numbers, titles, branch names, and dependencies for all 62 issues across 12 milestones. Read this before branching or planning any issue. |
| `.planning/v2-schema.md` | Config file model and all schema blocks: `[[workspace]]`, `[[repo]]`, `[[remote]]`, `[[sync_pair]]`, `[[workstream]]`, `.gitw-stream` manifest |
| `.planning/v2-commands.md` | Full command tree, cut list from v1, per-command specifications, status output format |
| `.planning/v2-remote-management.md` | `git w sync` fan-out, `git w remote` subcommand, workstream push protection, `pre-push` hook, state file, implementation notes |
| `.planning/v2-infra-patterns.md` | Pattern A (branch-per-env) and Pattern B (folder-per-env) with all creation examples |
| `.planning/v2-agent-interop.md` | Three-level `AGENTS.md` strategy, `git w agent context --json`, GSD interop specifics |
| `.planning/v2-migration.md` | `git w migrate` spec, `pkg/migrate` package, v1 breaking changes summary |
| `.planning/v2-milestones.md` | All 12 milestone scope descriptions, resolved design decisions, deferred items |
| `.planning/PROJECT.md` | GSD project context (generated by `/gsd-new-project`) |
| `.planning/ROADMAP.md` | GSD phase-level roadmap across all 12 milestones |
| `.planning/STATE.md` | GSD current position tracker |
