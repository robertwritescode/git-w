---
phase: 6
slug: add-workstream-root-config-block
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-05
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test via Mage |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `mage testfast` |
| **Full suite command** | `mage test` |
| **Estimated runtime** | ~25 seconds |

---

## Sampling Rate

- **After every task commit:** Run `mage testfast`
- **After every plan wave:** Run `mage test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | CFG-06 | unit+integration | `mage testfast` | ✅ | ⬜ pending |
| 06-01-02 | 01 | 1 | CFG-06 | unit+integration | `mage testfast` | ✅ | ⬜ pending |
| 06-02-01 | 02 | 2 | CFG-06 | unit+integration | `mage testfast` | ✅ | ⬜ pending |
| 06-02-02 | 02 | 2 | CFG-06 | unit+integration | `mage test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
