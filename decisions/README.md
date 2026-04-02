# Decisions Context for Claude

This directory contains Architecture Decision Records (ADRs). See the `/decision` skill for the template and guidelines on when to create decisions.

## Decision Index

| # | Title | File | Date | Summary |
|---|-------|------|------|---------|
| 0001 | Cobra constructor pattern for test isolation | [0001](0001-cobra-constructor-pattern-for-test-isolation.md) | 2026-04-02 | Unexported `newXxxCmd()` constructors for per-test fresh command trees |
| 0002 | Use `render` instead of `template` | [0002](0002-render-package-avoids-stdlib-template-shadow.md) | 2026-04-02 | Avoid stdlib `text/template` shadowing in `internal/` |
| 0003 | Defensive deep copy for package registries | [0003](0003-defensive-copy-for-package-registries.md) | 2026-04-02 | Registry accessors return deep copies via `slices.Clone` to prevent shared-backing-array mutation |
| 0004 | Separate data registries from behavior packages | [0004](0004-separate-data-registries-from-behavior-packages.md) | 2026-04-02 | Data registries (`internal/stack/`) are leaf packages with read-only accessors; behavior packages import them, never the reverse |
| 0005 | render.Merge as single merge orchestrator | [0005](0005-render-merge-as-single-merge-orchestrator.md) | 2026-04-02 | `render.Merge` orchestrates multi-stack merging into `GenerationConfig`, delegating domain logic to `firewall.Merge` |

## Maintenance

When creating new decisions with `/decision`, remember to add an entry to this index table.
