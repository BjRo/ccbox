# Decisions Context for Claude

This directory contains Architecture Decision Records (ADRs). See the `/decision` skill for the template and guidelines on when to create decisions.

## Decision Index

| File | Title | Date | Summary |
|------|-------|------|---------|
| 0001 | Cobra constructor pattern for test isolation | 2026-04-02 | Unexported `newXxxCmd()` constructors for per-test fresh command trees |
| 0002 | Use `render` instead of `template` | 2026-04-02 | Avoid stdlib `text/template` shadowing in `internal/` |
| 0003 | Defensive deep copy for package registries | 2026-04-02 | Registry accessors return deep copies via `slices.Clone` to prevent shared-backing-array mutation |
| 0004 | Separate data registries from behavior packages | 2026-04-02 | Data registries (`internal/stack/`) are leaf packages with read-only accessors; behavior packages import them, never the reverse |

## Maintenance

When creating new decisions with `/decision`, remember to add an entry to this index table.
