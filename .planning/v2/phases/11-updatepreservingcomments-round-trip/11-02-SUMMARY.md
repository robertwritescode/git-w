---
plan: 11-02
phase: 11-updatepreservingcomments-round-trip
status: complete
completed: 2026-04-07
---

## Summary

Added `pkg/config/round_trip_test.go` — a table-driven testify suite covering all `diskConfig` block types with comment preservation and position assertions. Satisfies CFG-12.

## What Was Built

- `RoundTripSuite` in `pkg/config/round_trip_test.go` (342 lines)
- 10 per-block-type subtests: metarepo, workspace, repo, repo path mutation, remote, remote+branch_rule, sync_pair, workstream, groups, worktrees
- 1 all-blocks integration test combining all block types in a single `.gitw` identity round-trip
- Each test writes a `.gitw` file with embedded comments, calls `config.Load` + `config.Save`, reads the file back, and asserts comments are both present AND positioned before their anchor keys

## Key Files

- `pkg/config/round_trip_test.go` — new file; `RoundTripSuite`, `TestRoundTrip_PerBlockType`, `TestRoundTrip_AllBlocks`

## Decisions

- Used `[[workspace]]` and `[[workstream]]` headers as anchors instead of key-value pairs because go-toml marshals string values with single quotes in those blocks, causing quote-style mismatches with double-quoted originals
- `workstream` test fixture includes a `[[remote]]` block to satisfy load-time validation ("unknown remote" error)
- Package is `package config_test` (black-box) consistent with `loader_test.go`

## Test Results

`mage test` passed — 11 suite subtests green, all 16 packages pass with race detector.

## Self-Check: PASSED
