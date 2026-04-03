---
# ccbox-00ez
title: Unit tests for stack detection
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:35:53Z
updated_at: 2026-04-03T00:00:00Z
parent: ccbox-6g75
---

## Description
Table-driven tests for the stack detection scanner:
- Single Go project (go.mod only) → detects Go
- Single Node project (package.json) → detects Node
- TypeScript project (package.json + tsconfig.json) → detects Node/TypeScript
- Python project (pyproject.toml) → detects Python
- Rust project (Cargo.toml) → detects Rust
- Ruby project (Gemfile) → detects Ruby
- Multi-stack (go.mod + package.json) → detects Go + Node
- Empty directory → no stacks detected
- Ignores vendor/, node_modules/, .git/

Use `testing/fstest.MapFS` for in-memory filesystem tests.

## Analysis

All tests described in this bean now exist in `internal/detect/detect_test.go` (21 tests, all passing).
Most were implemented as part of the stack detection feature work. The missing "both markers" test
(`TestDetect_SingleStack_Node_BothMarkers`) was added during this implementation phase.

### Existing Coverage Mapping

| Bean Requirement | Test(s) | Status |
|-----------------|---------|--------|
| Single Go (go.mod) | `TestDetect_SingleStack_Go` | ✅ |
| Single Node (package.json) | `TestDetect_SingleStack_Node` | ✅ |
| TypeScript (tsconfig.json only) | `TestDetect_SingleStack_Node_TsconfigOnly` | ✅ |
| TypeScript (package.json + tsconfig.json) | `TestDetect_SingleStack_Node_BothMarkers` | ✅ |
| Python (pyproject.toml + others) | `TestDetect_SingleStack_Python` (4 subtests) | ✅ |
| Rust (Cargo.toml) | `TestDetect_SingleStack_Rust` | ✅ |
| Ruby (Gemfile) | `TestDetect_SingleStack_Ruby_Gemfile` | ✅ |
| Multi-stack (go.mod + package.json) | `TestDetect_MultiStack` | ✅ |
| Empty directory | `TestDetect_NoStacks` | ✅ |
| Ignores vendor/ | `TestDetect_SkipsVendorDir` | ✅ |
| Ignores node_modules/ | `TestDetect_SkipsNodeModules` | ✅ |
| Ignores .git/ | `TestDetect_SkipsGitDir` | ✅ |

### Additional Coverage (beyond bean spec)

- `TestDetect_SingleStack_Ruby_Gemspec` — glob-based detection
- `TestDetect_AllStacks` — all 5 stacks simultaneously
- `TestDetect_MarkerInSubdir` — one-level-deep detection
- `TestDetect_MarkerTwoLevelsDeep_NotDetected` — depth limit
- `TestDetect_GemspecInSubdir` / `TestDetect_GemspecInSkipDir_NotDetected` — glob + skip
- `TestDetect_ResultIsSorted` — deterministic output
- `TestDetect_EmptyResult_IsNonNil` — non-nil empty slice invariant
- `TestDetect_NoDuplicates` — deduplication
- `TestDetect_PublicAPI_InvalidDir` / `TestDetect_PublicAPI_NotADirectory` — error paths

## Checklist

- [x] Tests written
- [x] No TODO/FIXME/HACK/XXX in code
- [x] Lint passes
- [x] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] All other checklist items
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | completed | 1 | 2026-04-03 |
| challenge | completed | 1 | 2026-04-03 |
| implement | completed | 1 | 2026-04-03 |
| pr | pending | | |
| review | pending | | |
| codify | pending | |