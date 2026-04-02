---
# ccbox-00ez
title: Unit tests for stack detection
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:35:53Z
updated_at: 2026-04-02T10:35:53Z
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