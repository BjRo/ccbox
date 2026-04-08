# 9. Multi-stage Dockerfile for user customizations

Date: 2026-04-07

## Status

Accepted

## Context

`agentbox init` generates a Dockerfile and refuses to run if `.devcontainer/` already exists. Users who add project-specific tools (e.g., beans CLI, custom linters) lose those additions if they ever need to regenerate. We need a mechanism where agentbox owns the base and the user owns an extension layer that survives regeneration.

Three approaches were evaluated:

1. **Multi-stage Dockerfile** (chosen) -- agentbox owns the `agentbox` stage, user owns the `custom` stage
2. **Devcontainer Features** -- heavyweight for simple tool installs, requires JSON merging
3. **postCreateCommand script** -- tools reinstall on every container rebuild (no Docker layer caching)

## Decision

Use a multi-stage Dockerfile where:

- `FROM debian:bookworm-slim AS agentbox` contains all generated content (system packages, runtimes, LSPs, Claude Code, firewall scripts)
- `FROM agentbox AS custom` is the user-managed stage where project-specific tools are added
- `devcontainer.json` targets the `custom` stage via `"build": {"target": "custom"}`
- `agentbox update` parses the Dockerfile at the `FROM agentbox AS custom` line, replaces everything before it with freshly rendered content, and preserves everything from that line onward verbatim
- `mise-config.toml` is also preserved on update as user-editable content

Key design decisions:

- **Docker stage syntax as delimiter** -- no comment markers needed; `FROM agentbox AS custom` is both valid Docker syntax and a reliable parsing boundary
- **`strings.Cut`-based byte-offset splitting** preserves exact whitespace (no split/join that could alter line endings)
- **`--force` flag** for recovery when the custom stage delimiter is missing
- **`--stack` and `--extra-domains` flags** on update permanently change `.agentbox.yml`
- **No `--runtime-version` on update** -- users edit `mise-config.toml` directly for version changes
- **No explicit ownership tracking** -- ownership is structural (stage boundary for Dockerfile, convention for `mise-config.toml`)
- **`renderFiles` is a pure function** taking data arguments (`[]stack.StackID`, `[]string`, `map[string]string`), not Cobra or wizard types

## Consequences

- Users get Docker-native customization with full layer caching for their tools
- Regeneration via `agentbox update` is safe and preserves user work
- Parsing is simple (line scan for FROM line, byte-offset slicing)
- `--force` provides an escape hatch for corrupted Dockerfiles
- Runtime version changes are a manual `mise-config.toml` edit, not a CLI flag on update
- The `renderFiles` helper is reusable by both `init` and `update` commands
