---
phase: 01-add-workspace-block
plan: 02
subsystem: agents
tags: [agents, framework, registry, interface]

requires:
  - phase: 01-01
    provides: AgenticFrameworks []string field on MetarepoConfig (needed by 01-03 to validate)
provides:
  - SpecFramework interface with 5 methods (Name, PlanningDirExists, InitInstructions, ProhibitedActions, WorkspaceCreationProhibited)
  - ProhibitedAction struct (Action, Reason, Alternative)
  - GSDFramework implementing SpecFramework with compile-time assertion
  - FrameworkFor(name string) (SpecFramework, error) registry lookup
  - FrameworksFor(names []string) ([]SpecFramework, error) bulk resolver
affects:
  - 01-03

tech-stack:
  added: []
  patterns:
    - "knownFrameworks map[string]SpecFramework registry — add new frameworks by inserting one entry"
    - "var _ SpecFramework = GSDFramework{} compile-time interface assertion"

key-files:
  created:
    - pkg/agents/framework.go
    - pkg/agents/gsd.go
    - pkg/agents/registry.go
    - pkg/agents/registry_test.go
  modified: []

key-decisions:
  - "No register.go in Phase 1 — pkg/agents has no cobra commands until Phase 9 (M9)"
  - "GSDFramework.InitInstructions and ProhibitedActions are stubs; full content generated in Phase 9"
  - "FrameworkFor error message lists all valid names so users know what to write in agentic_frameworks"

requirements-completed:
  - CFG-11

duration: 5min
completed: 2026-04-02
---

# Phase 01 Plan 02: pkg/agents Bootstrap Summary

**SpecFramework interface, GSDFramework implementation, and FrameworkFor/FrameworksFor registry established in pkg/agents — enables agentic_frameworks validation in 01-03**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-02
- **Completed:** 2026-04-02
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- `SpecFramework` interface exported with 5 methods; `ProhibitedAction` struct exported
- `GSDFramework` satisfies `SpecFramework` via compile-time assertion `var _ SpecFramework = GSDFramework{}`
- `FrameworkFor("gsd")` returns `(GSDFramework{}, nil)`; unknown names return actionable error listing valid identifiers
- `FrameworksFor` handles nil/empty input correctly (returns empty slice, no error)
- All 14 packages pass `mage testfast`

## Task Commits

1. **Tasks 1 + 2: framework.go, gsd.go, registry.go, registry_test.go** - `64f60ce` (feat)

## Files Created/Modified
- `pkg/agents/framework.go` — SpecFramework interface + ProhibitedAction type
- `pkg/agents/gsd.go` — GSDFramework implementation with PlanningDirExists (os.Stat), stub InitInstructions, ProhibitedActions, WorkspaceCreationProhibited=true
- `pkg/agents/registry.go` — knownFrameworks map, FrameworkFor, FrameworksFor, validFrameworkList helper
- `pkg/agents/registry_test.go` — Table-driven tests for FrameworkFor, FrameworksFor, GSDFramework methods

## Decisions Made
- Stubs for `InitInstructions` and `ProhibitedActions` are intentional — full AGENTS.md generation is Phase 9 work
- `FrameworksFor(nil)` returns an empty slice (not nil) for safe range iteration downstream

## Deviations from Plan
None - plan executed exactly as written

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `agents.FrameworksFor(names)` is ready for pkg/config to call in 01-03
- All tests passing; no blockers

---
*Phase: 01-add-workspace-block*
*Completed: 2026-04-02*
