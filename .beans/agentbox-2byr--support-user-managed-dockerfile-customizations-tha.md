---
# agentbox-2byr
title: Support user-managed Dockerfile customizations that survive regeneration
status: in-progress
type: feature
priority: normal
created_at: 2026-04-07T19:30:23Z
updated_at: 2026-04-08T07:23:01Z
---

Users need a way to add project-specific tools (e.g., beans CLI) to the devcontainer that won't be overwritten when agentbox regenerates files. Currently agentbox refuses to run if .devcontainer/ exists, and there's no update path at all.

Explore a Dockerfile.custom or similar mechanism where agentbox owns the base Dockerfile and the user owns an extension layer. The base Dockerfile should reference the custom file so both are used during build. Agentbox regeneration updates the base, leaves the custom file untouched.

This is a design exploration — needs /explore-approaches before implementation.

## Definition of Done

- [ ] Tests written (TDD: write tests before implementation)
- [ ] No new TODO/FIXME/HACK/XXX comments introduced
- [ ] `golangci-lint run ./...` passes with no errors
- [ ] `go test ./...` passes with no failures
- [ ] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review

## Approach Exploration

**Date**: 2026-04-07
**Researcher**: Claude (Opus 4.6)

### Problem Summary

Currently `agentbox init` refuses to run if `.devcontainer/` already exists (`cmd/init.go:46`). There is no update/regeneration path. Users who add project-specific tools (e.g., beans CLI, custom linters) lose those additions if they ever need to regenerate. The bean asks for a mechanism where agentbox owns the base and the user owns an extension layer.

### Current Architecture Context

- `cmd/init.go` renders 9 files into `.devcontainer/` via `render.Merge` + per-file render functions.
- The Dockerfile template (`internal/render/templates/Dockerfile.tmpl`) is a single-stage build ending at `WORKDIR /workspace`.
- `devcontainer.json.tmpl` is static JSON referencing `"dockerfile": "Dockerfile"`.
- `.agentbox.yml` records stacks, extra domains, and agentbox version but is not currently used for regeneration.
- No concept of "owned by agentbox" vs "owned by user" exists for any generated file.

### Approach A: Dockerfile.user with FROM (Multi-Stage Extension)

**Mechanism**: Agentbox generates `Dockerfile` (owned by agentbox, safe to overwrite). A separate `Dockerfile.user` is created on first init with a stub (`FROM agentbox AS base` or similar). `devcontainer.json` references `Dockerfile.user` as the build file. On regeneration, agentbox overwrites `Dockerfile` but leaves `Dockerfile.user` untouched.

**Implementation sketch**:
1. Rename the current Dockerfile template output to keep its role as the base. Add a final build stage name: `FROM debian:bookworm-slim AS agentbox-base` at top.
2. Generate a new `Dockerfile.user` template: `FROM agentbox-base` plus a comment block explaining where to add custom RUN commands. This file is only created if it does not exist.
3. Change `devcontainer.json.tmpl` to point at `Dockerfile.user`.
4. Add `agentbox update` (or `agentbox init --force`) that overwrites agentbox-owned files, skips user-owned files.
5. Track owned vs user files in `.agentbox.yml` or by convention (file naming).

**Tradeoffs**:
- (+) Standard Docker pattern. No new concepts for Docker-literate users.
- (+) Full Dockerfile power for customization (apt-get, COPY, multi-stage, etc.).
- (+) Clean separation: agentbox files are overwritable, user file is untouched.
- (+) Works with the existing devcontainer build pipeline (single Dockerfile reference).
- (-) Multi-stage FROM requires the stages to be in the same Dockerfile OR use `docker build --target`. Actually, a simpler variant: `Dockerfile.user` just does `FROM` referencing the base image built from `Dockerfile`. But devcontainer only supports a single Dockerfile. The correct approach is to keep both stages in one file OR have `Dockerfile.user` use the Dockerfile as a build stage via relative COPY --from. This needs careful design.
- (-) Docker build context: devcontainer builds from a single Dockerfile. Having two Dockerfiles requires either concatenation at build time or a wrapper script, which adds complexity.

**Revised variant (single-file with markers)**: Instead of two Dockerfiles, use marker comments in a single Dockerfile: `# --- BEGIN USER CUSTOMIZATION ---` / `# --- END USER CUSTOMIZATION ---`. Agentbox regeneration preserves content between markers and overwrites everything else. This is simpler but fragile (marker parsing, user accidentally deleting markers).

**Confidence**: 55% -- The two-Dockerfile approach has a fundamental friction with devcontainer expecting a single Dockerfile. The marker approach works but is fragile.

### Approach B: Devcontainer Features for User Customizations

**Mechanism**: Users add project-specific tools via [Dev Container Features](https://containers.dev/implementors/features/) in `devcontainer.json`. Agentbox owns all files in `.devcontainer/` and can safely regenerate them. User customizations live in the `features` section of `devcontainer.json`, or in a local `.devcontainer/features/` directory with custom feature definitions.

**Implementation sketch**:
1. Add an `agentbox update` command that re-renders all agentbox-owned files but preserves user-added `features` entries in `devcontainer.json`.
2. Parse existing `devcontainer.json` before overwriting, extract the `features` block, merge it into the newly rendered JSON.
3. Optionally, store user features in `.agentbox.yml` so they survive full regeneration without JSON merging.
4. Document the Features pattern in the generated README.

**Tradeoffs**:
- (+) Uses the official devcontainer extension mechanism. No custom file conventions.
- (+) Features are cached and layered by the devcontainer runtime, so rebuilds are efficient.
- (+) Agentbox can fully own all generated files; user intent is captured in a structured way.
- (-) Features are OCI artifacts or local scripts with a specific structure (`devcontainer-feature.json` + `install.sh`). For simple "install one binary" cases, this is heavyweight.
- (-) JSON merging is error-prone (ordering, comments, trailing commas). `devcontainer.json` does not support comments in standard JSON, though the spec allows JSONC.
- (-) Not all tools are available as published Features. Users would need to create local feature definitions for custom tools, which is a learning curve.
- (-) The current `devcontainer.json.tmpl` is static. Making it dynamic (to preserve user features) requires either JSON parsing/merging in Go or splitting the template.

**Confidence**: 40% -- Correct in principle but heavyweight for the common case of "install one more CLI tool." The JSON merge logic is a significant complexity addition.

### Approach C: Dockerfile.user as a Separate Build Step via postCreateCommand

**Mechanism**: Agentbox owns the Dockerfile entirely. User customizations go in a `Dockerfile.user` (or `setup-user.sh` script) that runs as a `postCreateCommand` or `onCreateCommand` in `devcontainer.json`. This avoids the single-Dockerfile constraint entirely.

**Implementation sketch**:
1. On `agentbox init`, generate a stub `setup-user.sh` (e.g., `#!/bin/bash\n# Add your custom tool installations here\n`). Mark it as user-owned.
2. In `devcontainer.json.tmpl`, add `"onCreateCommand": "bash .devcontainer/setup-user.sh"` (runs once on container creation, before postStartCommand).
3. Add `agentbox update` command that overwrites agentbox-owned files, skips `setup-user.sh`.
4. Track ownership in `.agentbox.yml`: list agentbox-owned files explicitly; anything not listed is user-owned.

**Tradeoffs**:
- (+) Simplest implementation. No Dockerfile gymnastics. No JSON merging.
- (+) Users write plain bash -- lowest learning curve.
- (+) Clean ownership model: agentbox files vs user files, tracked in `.agentbox.yml`.
- (+) `onCreateCommand` runs after the image build, so it works even if the base image is cached.
- (-) Tools installed via `onCreateCommand`/`postCreateCommand` are NOT baked into the Docker image layer. They re-install on every container rebuild (not restart, but rebuild). This can be slow for large tools.
- (-) Does not benefit from Docker layer caching. If the user installs heavy dependencies (e.g., compiling from source), container creation becomes slow.
- (-) Some tools need root access during install; `onCreateCommand` runs as `remoteUser` (node). Users would need `sudo` in their script.

**Confidence**: 70% -- Pragmatic, simple, and solves the core problem. The re-install-on-rebuild downside is real but acceptable for most project-specific tools (which tend to be small CLI binaries downloaded via curl).

### Decision

**Approach A (single Dockerfile with user section)** — chosen for natural Docker UX and layer caching.

**Variant**: Single Dockerfile with a marker delimiter. Agentbox owns everything above the marker, user owns everything below. On regeneration, agentbox reads the existing file, preserves the user section below the marker, and overwrites the agentbox section above it.

**Why not C**: While simpler to implement, `setup-user.sh` reinstalls tools on every container rebuild (no Docker layer caching). For tools like golangci-lint or beans CLI that take 30+ seconds to compile, this adds friction. Approach A bakes customizations into the image layer.

**Why not B**: Devcontainer Features are heavyweight for simple tool installs and require JSON merging logic.

**Key implementation decisions for /refine**:
- Marker format: e.g. `# === USER CUSTOMIZATIONS BELOW (do not remove this line) ===`
- On first `agentbox init`: generate Dockerfile with marker and empty user section
- On `agentbox update` (or `agentbox init --force`): extract user section below marker from existing file, regenerate agentbox section, append user section
- Validate marker presence on update; warn if missing
- `.agentbox.yml` tracks which files are agentbox-owned vs user-editable (config.toml is already user-editable)
