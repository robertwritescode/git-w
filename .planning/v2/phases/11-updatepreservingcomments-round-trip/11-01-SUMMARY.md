---
plan: 11-01
phase: 11-updatepreservingcomments-round-trip
status: complete
completed: 2026-04-07
---

## Summary

Fixed two documented tech debt items in `pkg/toml/preserve.go` and `pkg/config/loader.go`.

## What Was Built

- Replaced all `interface{}` occurrences with `any` throughout `pkg/toml/preserve.go` (9 function signatures, local variable declarations)
- Replaced `interface{}` with `any` in `saveWithCommentPreservation` and `marshalToml` in `pkg/config/loader.go`
- Fixed `applySmartUpdate` to propagate errors from `smartUpdate` — returns `(nil, err)` instead of silently swallowing with `(newBytes, nil)`

## Key Files

- `pkg/toml/preserve.go` — all `interface{}` replaced with `any`; `applySmartUpdate` now propagates errors
- `pkg/config/loader.go` — `saveWithCommentPreservation` and `marshalToml` updated to use `any`

## Decisions

- No behavioral logic changes; the only semantic change is `applySmartUpdate` now surfaces errors from `smartUpdate` rather than masking them behind a silent fallback

## Test Results

`mage testfast` passed — all packages green, no regressions.

## Self-Check: PASSED
