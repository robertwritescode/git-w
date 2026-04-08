# Milestones

## v2.0 M1: Config Schema + Loader (Shipped: 2026-04-08)

**Phases completed:** 13 phases, 26 plans, 32 tasks

**Key accomplishments:**

- Renamed WorkspaceMeta to MetarepoConfig (TOML key: metarepo), added WorkspaceBlock struct and Workspaces []WorkspaceBlock field, updated all 13 test fixtures from [workspace] to [metarepo]
- SpecFramework interface, GSDFramework implementation, and FrameworkFor/FrameworksFor registry established in pkg/agents — enables agentic_frameworks validation in 01-03
- agentic_frameworks validation and ["gsd"] default wired into pkg/config load pipeline; all Phase 1 success criteria satisfied
- `[[repo]]` array-of-tables TOML format with required `name` field and `clone_url` replacing `url` — load/save pipeline, validation, RepoByName accessor, and testutil helpers all updated
- `track_branch` and `upstream` fields on `[[repo]]` with co-presence (D-01) and per-group uniqueness (D-02) validation — env alias annotation fully deliverable at config load time
- Load-time warnings for non-conforming repo paths: cfg.Warnings field, warnNonConformingRepoPaths function, stderr output in LoadConfig, and testutil fixtures migrated to repos/<name>
- pkg/config/config.go:
- pkg/config/loader.go:
- `pkg/config/config.go`:
- `pkg/config/loader.go`:
- In-memory workstream schema primitives now exist in `pkg/config` with merge and lookup behavior covered by deterministic table-driven tests for downstream loader wiring.
- Root `[[workstream]]` config parsing now enforces strict keys, required `name`/`remotes`, remote reference integrity, duplicate rejection, and deterministic sorted normalization.
- Three field-level merge helpers (MergeRepo, MergeWorkspace, mergeMetarepo) added to pkg/config/config.go following the established non-zero-wins pattern of MergeRemote
- `mergePrivateConfig` wired into `Load()` — all callers now automatically merge `.git/.gitw` with field-level semantics, unknown repo errors, and silent skip for absent private file
- Five new exported types added to pkg/config/config.go: WorkstreamStatus typed string alias with three constants, WorktreeEntry, ShipState, StreamContext, and WorkstreamManifest structs matching the v2 .gitw-stream schema
- `pkg/config/stream.go` delivers `LoadStream` with parse-default-validate pipeline, full table-driven test coverage in `stream_test.go` including `[ship]` and `[context]` blocks
- Load-time hard error for v1 `[[workgroup]]` blocks using raw byte scanning, with migrate directive and 5-case test coverage
- VERIFICATION.md reports generated for Phase 01 (workspace block, passed 4/4) and Phase 03 (repos path convention, passed 3/3) — both M1 phases confirmed complete
- VERIFICATION.md reports generated for Phase 04 (remote + branch_rule, passed 4/4) and Phase 05 (sync_pair cycle detection, passed 3/3) — both M1 phases confirmed complete
- Phase 02 SUMMARY.md files reconstructed from plan files and codebase; 02-VERIFICATION.md generated — passed 5/5
- Phase 09 (default remotes cascade) verified passed 3/3; CFG-10 and CFG-12 confirmed already [x] in REQUIREMENTS.md
- INT-01 — Re-validate workstream remotes after private config merge

---
