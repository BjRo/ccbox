---
# ccbox-6j8r
title: Project file scanner for stack detection
status: todo
type: task
priority: high
created_at: 2026-04-02T10:34:14Z
updated_at: 2026-04-02T10:34:14Z
parent: ccbox-2n15
---

## Description
Implement a scanner that walks the target project directory and detects tech stacks by looking for marker files:

| Stack | Marker Files |
|-------|-------------|
| Go | `go.mod` |
| Node/TypeScript | `package.json`, `tsconfig.json` |
| Python | `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile` |
| Rust | `Cargo.toml` |
| Ruby | `Gemfile`, `*.gemspec` |

Returns a list of detected stacks. Only scan root and one level deep (avoid vendor/node_modules).