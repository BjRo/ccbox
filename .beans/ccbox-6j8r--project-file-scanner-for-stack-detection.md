---
# ccbox-6j8r
title: Project file scanner for stack detection
status: in-progress
type: task
priority: high
created_at: 2026-04-02T10:34:14Z
updated_at: 2026-04-02T12:50:49Z
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

## Checklist

- [ ] Tests written (TDD)
- [ ] No TODO/FIXME/HACK/XXX in code
- [ ] Lint passes
- [ ] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] ADR written (if architectural)
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | pending | | |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |