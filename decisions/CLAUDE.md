# Decisions Context for Claude

This directory contains Architecture Decision Records (ADRs). See the `/decision` skill for the template and guidelines on when to create decisions.

## Decision Index

| File | Title | Date | Summary |
|------|-------|------|---------|
| 0001 | Cobra constructor pattern for test isolation | 2026-04-02 | Unexported `newXxxCmd()` constructors for per-test fresh command trees |
| 0002 | Use `render` instead of `template` | 2026-04-02 | Avoid stdlib `text/template` shadowing in `internal/` |

## Maintenance

When creating new decisions with `/decision`, remember to add an entry to this index table.
