---
# agentbox-2byr
title: Support user-managed Dockerfile customizations that survive regeneration
status: in-progress
type: feature
priority: normal
created_at: 2026-04-07T19:30:23Z
updated_at: 2026-04-08T07:55:20Z
---

Users need a way to add project-specific tools (e.g., beans CLI) to the devcontainer that won't be overwritten when agentbox regenerates files. Currently agentbox refuses to run if .devcontainer/ exists, and there's no update path at all.

Explore a Dockerfile.custom or similar mechanism where agentbox owns the base Dockerfile and the user owns an extension layer. The base Dockerfile should reference the custom file so both are used during build. Agentbox regeneration updates the base, leaves the custom file untouched.

This is a design exploration — needs /explore-approaches before implementation.

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [x] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
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

**Variant**: Multi-stage Dockerfile. Agentbox owns the `agentbox` stage (FROM debian:bookworm-slim AS agentbox), user owns a `custom` stage (FROM agentbox AS custom). devcontainer.json uses `build.target: custom`. On update, agentbox replaces the agentbox stage and preserves the custom stage. Docker's own stage syntax acts as a natural delimiter — no comment markers needed.

**Why not C**: While simpler to implement, `setup-user.sh` reinstalls tools on every container rebuild (no Docker layer caching). For tools like golangci-lint or beans CLI that take 30+ seconds to compile, this adds friction. Approach A bakes customizations into the image layer.

**Why not B**: Devcontainer Features are heavyweight for simple tool installs and require JSON merging logic.

**Key implementation decisions for /refine**:
- Agentbox stage: `FROM debian:bookworm-slim AS agentbox` — contains all generated content
- Custom stage: `FROM agentbox AS custom` — stub with helpful comments, user adds RUN commands here
- `devcontainer.json` uses `"build": {"dockerfile": "Dockerfile", "target": "custom"}`
- On first `agentbox init`: generate Dockerfile with both stages
- On `agentbox update`: parse Dockerfile to find the `FROM agentbox AS custom` line, replace everything before it, preserve everything from that line onward
- If custom stage is missing on update: error out, require `--force` to proceed
- `.agentbox.yml` tracks which files are agentbox-owned vs user-editable (config.toml is already user-editable)


## Implementation Plan

**Date**: 2026-04-08 (revised)
**Planner**: Claude (Opus 4.6)

### Approach

Multi-stage Dockerfile where agentbox owns the `agentbox` stage and the user owns the `custom` stage, combined with a new `agentbox update` command that re-renders agentbox-owned content while preserving user customizations. The Dockerfile uses Docker's own `FROM ... AS ...` stage syntax as the delimiter -- no comment markers needed. On init, both stages are generated. On update, the Dockerfile is split at the `FROM agentbox AS custom` line: everything before it is replaced with freshly rendered content, everything from that line onward is preserved verbatim. A `--force` flag allows full regeneration when the custom stage delimiter is missing.

**Key challenge decisions incorporated**:
1. `renderFiles` signature uses pure data arguments -- `renderFiles(stackIDs []stack.StackID, extraDomains []string, versionOverrides map[string]string) (map[string][]byte, error)` -- no `*cobra.Command`, no `wizard.Choices`.
2. `SplitAtCustomStage` uses `strings.Cut` on the raw string to find the byte offset, slicing the original content without split/join that could alter whitespace.
3. No `--runtime-version` flag on the update command. Users edit `config.toml` directly for version changes. Documented in README and help text.
4. `custom_stage.go` uses `templateFS` + `ParseFS` for consistency with other render files (not individual `//go:embed`).

### Files to Create

- `cmd/update.go` -- New `agentbox update` subcommand
- `cmd/update_test.go` -- Unit tests for the update command
- `cmd/update_integration_test.go` -- Integration tests for update flow
- `internal/render/templates/custom-stage.tmpl` -- Template for the custom stage stub
- `internal/render/custom_stage.go` -- Render function for the custom stage stub
- `internal/render/custom_stage_test.go` -- Tests for custom stage rendering
- `internal/dockerfile/split.go` -- Dockerfile parsing: split at stage boundary
- `internal/dockerfile/split_test.go` -- Tests for Dockerfile splitting
- `internal/dockerfile/doc.go` -- Package doc comment
- `decisions/0009-multi-stage-dockerfile-for-user-customizations.md` -- ADR for this approach

### Files to Modify

- `cmd/root.go` -- Wire `newUpdateCmd` into the command tree
- `cmd/init.go` -- Extract `renderFiles` helper (pure data in, file map out); modify RunE to use it and append custom stage to Dockerfile output
- `cmd/init_test.go` -- Add tests for `renderFiles`; update Dockerfile content assertions to expect `AS agentbox` and custom stage
- `cmd/init_integration_test.go` -- Update Dockerfile content assertions to expect `AS agentbox` stage name and custom stage stub; update devcontainer.json assertions for `"target": "custom"`
- `internal/render/templates/Dockerfile.tmpl` -- Add `AS agentbox` to the FROM line
- `internal/render/templates/devcontainer.json.tmpl` -- Add `"target": "custom"` to the build section
- `internal/render/dockerfile_test.go` -- Update tests to expect `AS agentbox` in FROM line
- `internal/render/devcontainer_test.go` -- Update tests to expect `"target": "custom"`; `TestDevContainer_IsStatic` will need updating since template now has `target` field
- `internal/render/templates/README.md.tmpl` -- Add section documenting the custom stage, `agentbox update`, and config.toml editing for version changes; remove "do not edit manually" footer
- `decisions/README.md` -- Add ADR-0009 to index

### Steps

#### 1. Add `AS agentbox` to Dockerfile template

Modify `internal/render/templates/Dockerfile.tmpl` line 3: change `FROM debian:bookworm-slim` to `FROM debian:bookworm-slim AS agentbox`.

This is the foundational change. The stage name is what the custom stage references (`FROM agentbox AS custom`) and what the update command uses as a parsing boundary.

Update all existing Dockerfile tests in `internal/render/dockerfile_test.go` that assert on `FROM debian:bookworm-slim` to expect `FROM debian:bookworm-slim AS agentbox`. Key tests affected:
- `TestDockerfile_BaseImage` -- assert `FROM debian:bookworm-slim AS agentbox`
- `TestDockerfile_EmptyConfig` -- same
- `TestDockerfile_DirectConfig_MinimalValid` -- same

#### 2. Add `"target": "custom"` to devcontainer.json template

Modify `internal/render/templates/devcontainer.json.tmpl` to change the build section from:
```json
"build": {
    "dockerfile": "Dockerfile"
}
```
to:
```json
"build": {
    "dockerfile": "Dockerfile",
    "target": "custom"
}
```

Update `internal/render/devcontainer_test.go`:
- `TestDevContainer_FixedStructure` -- assert `build["target"] == "custom"`
- `TestDevContainer_IsStatic` -- still static (target is hardcoded, not templated), so the existing byte-equality assertion still passes
- Integration tests that unmarshal devcontainer.json -- add target field check

#### 3. Create `internal/dockerfile` package for Dockerfile parsing

Create a new package `internal/dockerfile/` with a single responsibility: splitting a Dockerfile at the custom stage boundary.

**`internal/dockerfile/doc.go`**: Package doc comment explaining the package provides utilities for parsing multi-stage Dockerfiles used by agentbox's update workflow.

**`internal/dockerfile/split.go`**:

```go
package dockerfile

import (
    "errors"
    "strings"
)

// CustomStageLine is the exact FROM line that marks the boundary between
// the agentbox-managed stage and the user-managed custom stage.
const CustomStageLine = "FROM agentbox AS custom"

// ErrNoCustomStage is returned when the Dockerfile does not contain
// the expected custom stage boundary line.
var ErrNoCustomStage = errors.New("dockerfile: custom stage not found (expected \"FROM agentbox AS custom\")")

// SplitAtCustomStage splits a Dockerfile into two parts:
// the agentbox-managed content (everything before the custom stage line)
// and the user-managed content (everything from the custom stage line onward, inclusive).
//
// The function uses strings.Cut on the raw string to find the boundary,
// preserving all original whitespace and newlines exactly as-is (no
// split/join that could alter line endings). It scans for a line whose
// trimmed content matches CustomStageLine (case-insensitive on FROM/AS
// keywords, exact on stage names agentbox/custom).
//
// If no match is found, it returns ErrNoCustomStage.
func SplitAtCustomStage(content string) (agentboxPart, userPart string, err error)
```

Implementation approach using `strings.Cut` on raw bytes:
1. Scan through the content looking for the `FROM agentbox AS custom` line. Instead of `strings.Split` (which would create a new slice and lose original whitespace on rejoin), iterate character-by-character or line-by-line using index arithmetic.
2. Concrete implementation: find each newline in the raw string, extract the line between the previous newline and the current one, trim it and check if it matches. When found, slice the original string at the start of that line.
3. The `matchesCustomStage` helper tokenizes the trimmed line: split on whitespace, check 4 tokens, tokens[0] is `FROM` (case-insensitive via `strings.EqualFold`), tokens[1] is `agentbox` (exact), tokens[2] is `AS` (case-insensitive), tokens[3] is `custom` (exact).

Key detail: use index-based slicing on the original `content` string. Track the byte offset of each line start as we scan. When we find the match at offset `pos`, return `content[:pos]` as `agentboxPart` and `content[pos:]` as `userPart`. This ensures zero whitespace alteration.

```go
func SplitAtCustomStage(content string) (string, string, error) {
    offset := 0
    for offset < len(content) {
        // Find end of current line.
        nl := strings.IndexByte(content[offset:], '\n')
        var line string
        if nl == -1 {
            line = content[offset:]
        } else {
            line = content[offset : offset+nl]
        }
        if matchesCustomStage(strings.TrimSpace(line)) {
            return content[:offset], content[offset:], nil
        }
        if nl == -1 {
            break
        }
        offset += nl + 1
    }
    return "", "", ErrNoCustomStage
}
```

**`internal/dockerfile/split_test.go`**: Comprehensive test coverage:
- Happy path: standard two-stage Dockerfile splits correctly
- Custom stage with user content (multiple RUN lines) preserved verbatim
- Case variations on FROM/AS keywords (`from agentbox as custom`, `FROM agentbox AS custom`)
- Extra whitespace around the FROM line (leading/trailing spaces)
- Dockerfile with no custom stage returns `ErrNoCustomStage`
- Empty input returns `ErrNoCustomStage`
- Custom stage line as the very first line (agentbox part is empty string)
- Multiple FROM lines (only the `FROM agentbox AS custom` one is the split point)
- Whitespace preservation: input with Windows-style `\r\n` line endings -- the split preserves them exactly
- Trailing content after custom stage line preserved byte-for-byte (round-trip: `agentboxPart + userPart == original`)

#### 4. Create custom stage stub template

**`internal/render/templates/custom-stage.tmpl`**:

```dockerfile
FROM agentbox AS custom

# =============================================================================
# USER CUSTOMIZATIONS
# =============================================================================
# Add your project-specific tools and configuration below.
# This stage is preserved when running `agentbox update`.
#
# Examples:
#   RUN go install github.com/user/tool@latest
#   RUN npm install -g some-cli
#   RUN pip install my-tool
#   COPY my-config.toml /home/node/.config/my-tool/config.toml
#
# The agentbox stage above provides: mise runtimes, LSP servers, Claude Code,
# firewall scripts, and all system dependencies. Your customizations layer
# on top via Docker's multi-stage build caching.
# =============================================================================
```

WORKDIR is not repeated in the custom stage -- Docker stages inherit WORKDIR from their parent via `FROM`. The agentbox stage sets `WORKDIR /workspace` and the custom stage inherits it.

**`internal/render/custom_stage.go`**:

```go
package render

import (
    "bytes"
    "fmt"
    "text/template"
)

var customStageTmpl = template.Must(template.ParseFS(templateFS, "templates/custom-stage.tmpl"))

// CustomStage renders the custom stage stub template. The template is static
// (no GenerationConfig needed) and produces the FROM line plus helpful
// comments for user customizations.
func CustomStage() (string, error) {
    var buf bytes.Buffer
    if err := customStageTmpl.ExecuteTemplate(&buf, "custom-stage.tmpl", nil); err != nil {
        return "", fmt.Errorf("render custom stage: %w", err)
    }
    return buf.String(), nil
}
```

Uses `templateFS` + `ParseFS` pattern consistent with `mise.go`, `devcontainer.go`, and `readme.go` (challenge finding 4).

**`internal/render/custom_stage_test.go`**:
- Renders without error
- Contains `FROM agentbox AS custom`
- Contains helpful comment text ("USER CUSTOMIZATIONS", "agentbox update")
- Is deterministic (two renders produce identical output)
- No template artifacts (`<no value>`, `<nil>`, `{{`, `}}`)
- Does NOT contain `WORKDIR` (inherited from parent stage)

#### 5. Extract `renderFiles` helper in `cmd/init.go`

Extract the rendering pipeline from `init.go`'s RunE into a pure helper function with no Cobra or wizard dependencies:

```go
// renderFiles produces all agentbox-managed file content for the given
// configuration. It returns a map from filename to content. The Dockerfile
// value contains only the agentbox stage (no custom stage). The caller is
// responsible for appending the custom stage (init) or preserving the
// existing one (update).
//
// versionOverrides maps tool names to version strings (e.g., "go" -> "1.22").
// Entries that do not match any runtime in the merged config are silently
// ignored (no coupling to the registry).
func renderFiles(stackIDs []stack.StackID, extraDomains []string, versionOverrides map[string]string) (map[string][]byte, error)
```

This function:
1. Calls `render.Merge(stackIDs, extraDomains)`
2. Calls `render.EnsureNode(&cfg)`
3. Applies version overrides from `versionOverrides` map to `cfg.Runtimes`
4. Renders all 9 templates (Dockerfile, devcontainer.json, firewall, claude, readme, mise config)
5. Returns the file map

No `*cobra.Command` parameter -- the function is pure data transformation (challenge finding 1). No `wizard.Choices` -- the caller is responsible for merging wizard choices into `versionOverrides` before calling.

Update `init.go`'s RunE to:
1. Build `versionOverrides` map from wizard choices + CLI flag (existing layering logic)
2. Call `renderFiles(stackIDs, extraDomains, versionOverrides)` to get file content map
3. Call `render.CustomStage()` to get the custom stage stub
4. Append the custom stage to the Dockerfile: `files["Dockerfile"] = append(files["Dockerfile"], '\n'); files["Dockerfile"] = append(files["Dockerfile"], []byte(customStage)...)`
5. Write all files as before
6. Write `.agentbox.yml` as before

The pre-existence guard remains (init still fails if `.devcontainer/` exists).

#### 6. Create `agentbox update` command

**`cmd/update.go`**:

```go
func newUpdateCmd() *cobra.Command
```

Note: no `prompter` parameter. The update command is non-interactive -- it reads configuration from `.agentbox.yml` and CLI flags. No wizard flow.

Flags:
- `--dir` (string): Target directory, same behavior as init
- `--stack` (string slice): Override stacks permanently. When set, replaces stacks in `.agentbox.yml`. Help text: `"Override stacks (persists to .agentbox.yml). Auto-detects if omitted."`
- `--extra-domains` (string slice): Override extra domains permanently
- `--force` (bool): Force full regeneration even if custom stage is missing

No `--runtime-version` flag (challenge finding 3). Users edit `config.toml` directly for version changes. The help text for `update` should note: "To change runtime versions, edit .devcontainer/config.toml directly."

No `--non-interactive` / `-y` flag -- the update command is always non-interactive.

RunE logic:

1. **Resolve directory** via `resolveDir(dir)`.
2. **Load `.agentbox.yml`**: Read from `filepath.Join(targetDir, config.Filename)`. If not found, error: `"no .agentbox.yml found in %s; run 'agentbox init' first"`.
3. **Determine stacks**: If `--stack` flag is set, use those (validate against registry via `validateStackIDs`). Otherwise, convert stacks from `.agentbox.yml` to `[]stack.StackID`.
4. **Determine extra domains**: If `--extra-domains` flag is set, use those. Otherwise, use `ExtraDomains` from `.agentbox.yml`.
5. **Read existing Dockerfile**: `os.ReadFile(filepath.Join(outDir, "Dockerfile"))`. If not found, error: `"no Dockerfile found in .devcontainer/; run 'agentbox init' first"`.
6. **Split Dockerfile**: Call `dockerfile.SplitAtCustomStage(string(existingDockerfile))`. If it returns `ErrNoCustomStage`:
   - If `--force` is set: proceed without preserving user content (full regeneration, generate fresh custom stage stub)
   - If `--force` is not set: error with message: `"Dockerfile does not contain custom stage (FROM agentbox AS custom); use --force to regenerate fully"`
7. **Read existing config.toml**: `os.ReadFile(filepath.Join(outDir, "config.toml"))`. If not found, proceed without preserving (will be freshly rendered). This handles the edge case of a partial `.devcontainer/` directory.
8. **Render fresh files**: Call `renderFiles(stackIDs, extraDomains, nil)` -- no version overrides (versions come from the preserved config.toml, challenge finding 3).
9. **Assemble Dockerfile**: Concatenate fresh agentbox stage + "\n" + preserved user part (from step 6). If `--force` was used (no user part), concatenate fresh agentbox stage + "\n" + fresh custom stage stub from `render.CustomStage()`.
10. **Write files**: Create `outDir` if missing (it should exist, but `MkdirAll` is safe). Write all files from `renderFiles` to `.devcontainer/`, with two overrides:
    - Replace `files["Dockerfile"]` with the assembled Dockerfile from step 9
    - Replace `files["config.toml"]` with the preserved config.toml from step 7 (skip re-rendering), unless config.toml was not found (step 7 returned error), in which case keep the freshly rendered one
11. **Make shell scripts executable**: Same chmod loop as init.
12. **Write `.agentbox.yml`**: Write updated config with potentially new stacks/domains, fresh `GeneratedAt` timestamp, current `AgentboxVersion`.
13. **Print summary**: `"Updated .devcontainer/ in %s\n"` to stderr.

**Ownership model**: No explicit ownership tracking in `.agentbox.yml`. Ownership is structural:
- Dockerfile: agentbox owns everything before `FROM agentbox AS custom`, user owns everything after
- config.toml: user-owned (preserved on update, unless missing)
- All other files in `.devcontainer/`: agentbox-owned (overwritten on update)

#### 7. Wire update command into root

Modify `cmd/root.go` `newRootCmd` to add: `cmd.AddCommand(newUpdateCmd())`.

Note: `newUpdateCmd` takes no parameters since the update command is non-interactive. This differs from `newInitCmd(prompter)` which needs the prompter for the wizard.

Update `newRootCmd` signature consideration: `newRootCmd` currently takes `prompter wizard.Prompter` which is passed to `newInitCmd`. Since `newUpdateCmd` takes no arguments, no change to `newRootCmd`'s signature is needed.

#### 8. Update README template

Modify `internal/render/templates/README.md.tmpl`:

**Add after the "Customization" section** a new "Updating" section:

```markdown
## Updating

To regenerate agentbox-managed files after changing stacks or updating agentbox:

    agentbox update --dir .

This preserves your custom Dockerfile stage and `config.toml` runtime versions.

To change detected stacks permanently:

    agentbox update --stack go,node,python

The `--stack` flag persists the new stack selection to `.agentbox.yml`.

To change runtime versions, edit `.devcontainer/config.toml` directly.

### Dockerfile Structure

The Dockerfile uses a multi-stage build:

1. **`agentbox` stage** -- Managed by agentbox. Contains system packages, runtimes, LSPs, Claude Code, and firewall scripts. Regenerated by `agentbox update`.
2. **`custom` stage** -- Managed by you. Add project-specific tools here. Preserved by `agentbox update`.
```

**Update the existing "Customization" section** to reference the Dockerfile custom stage as the primary customization mechanism (instead of only mentioning devcontainer.json edits).

**Remove the footer**: Remove the `*Generated by agentbox -- do not edit manually.*` footer since users now DO edit parts of the output (the custom Dockerfile stage and config.toml).

Update `internal/render/readme_test.go` to:
- Assert "Updating" section is present
- Assert "agentbox update" command is documented
- Assert "config.toml" is mentioned for version changes
- Assert "do not edit manually" is NOT present
- Assert "custom" stage is documented

#### 9. Write ADR-0009

Create `decisions/0009-multi-stage-dockerfile-for-user-customizations.md`:

**Context**: agentbox init generates a Dockerfile and refuses to run if .devcontainer/ exists. Users need to add project-specific tools that survive regeneration.

**Decision**: Multi-stage Dockerfile. Agentbox owns the `agentbox` stage (FROM debian:bookworm-slim AS agentbox). User owns the `custom` stage (FROM agentbox AS custom). devcontainer.json targets the `custom` stage. `agentbox update` parses the Dockerfile at the stage boundary line, replaces the agentbox stage, preserves user content. config.toml is also preserved as user-editable.

**Key design decisions**:
- Docker stage syntax as delimiter (no comment markers)
- `strings.Cut`-based byte-offset splitting to preserve exact whitespace
- `--force` flag for recovery when delimiter is missing
- `--stack` flag on update permanently changes `.agentbox.yml`
- No `--runtime-version` on update -- users edit config.toml directly
- No explicit ownership tracking -- ownership is structural (stage boundary for Dockerfile, convention for config.toml)
- `renderFiles` is a pure function taking data arguments, no Cobra or wizard types

**Consequences**:
- Users get Docker-native customization with full layer caching
- Regeneration is safe and preserves user work
- Parsing is simple (line scan for FROM line, byte-offset slicing)
- `--force` provides escape hatch for corrupted Dockerfiles
- Runtime version changes are a manual config.toml edit, not a CLI flag

Update `decisions/README.md` to add the new ADR entry.

### Testing Strategy

#### New unit tests (`internal/dockerfile/split_test.go`):
- **TestSplitAtCustomStage_HappyPath**: Standard two-stage Dockerfile splits correctly; verify `agentboxPart + userPart == original`
- **TestSplitAtCustomStage_UserContentPreserved**: Custom stage with multiple RUN lines; user content preserved verbatim byte-for-byte
- **TestSplitAtCustomStage_CaseInsensitiveKeywords**: `from agentbox as custom` and `FROM agentbox AS custom` both match
- **TestSplitAtCustomStage_LeadingWhitespace**: Line with leading spaces matches
- **TestSplitAtCustomStage_NoCustomStage**: Returns `ErrNoCustomStage`
- **TestSplitAtCustomStage_EmptyInput**: Returns `ErrNoCustomStage`
- **TestSplitAtCustomStage_CustomStageFirstLine**: agentbox part is empty string
- **TestSplitAtCustomStage_MultipleFROMLines**: Only the `FROM agentbox AS custom` line is the split point
- **TestSplitAtCustomStage_RoundTrip**: For various inputs, verify `agentboxPart + userPart == original` (whitespace preservation guarantee)

#### New unit tests (`internal/render/custom_stage_test.go`):
- **TestCustomStage_RendersWithoutError**: No error returned
- **TestCustomStage_ContainsFROMLine**: Contains `FROM agentbox AS custom`
- **TestCustomStage_ContainsHelpfulComments**: Contains "USER CUSTOMIZATIONS" and "agentbox update"
- **TestCustomStage_Deterministic**: Two renders produce identical output
- **TestCustomStage_NoTemplateArtifacts**: No `<no value>`, `<nil>`, `{{`, `}}`
- **TestCustomStage_NoWORKDIR**: Does not contain `WORKDIR` (inherited from parent)

#### New unit tests (`cmd/init_test.go` additions):
- **TestRenderFiles_ReturnsAllExpectedKeys**: Verify all 9 file keys are present in the returned map
- **TestRenderFiles_DockerfileContainsASAgentbox**: The returned Dockerfile contains `AS agentbox` but NOT `FROM agentbox AS custom` (custom stage is caller's responsibility)
- **TestRenderFiles_AllValuesNonEmpty**: Every value in the returned map is non-empty
- **TestRenderFiles_VersionOverridesApplied**: Pass `versionOverrides` with `"go"="1.22"`, verify config.toml contains `go = "1.22"`
- **TestRenderFiles_NilVersionOverrides**: Pass `nil` for versionOverrides, verify defaults are used
- **TestRenderFiles_InvalidStack**: Returns error for unknown stack ID

#### New unit tests (`cmd/update_test.go`):
- **TestUpdateCommand_RequiresAgentboxYml**: Error when `.agentbox.yml` is missing; message mentions `agentbox init`
- **TestUpdateCommand_RequiresDockerfile**: Error when `.devcontainer/Dockerfile` is missing
- **TestUpdateCommand_RequiresCustomStage**: Error when Dockerfile lacks `FROM agentbox AS custom` and `--force` is not set
- **TestUpdateCommand_ForceRegeneration**: `--force` proceeds when custom stage is missing, generates fresh stub containing `FROM agentbox AS custom`
- **TestUpdateCommand_PreservesCustomStage**: User content after `FROM agentbox AS custom` survives update; add `RUN echo hello` to custom stage, verify it is present after update
- **TestUpdateCommand_PreservesConfigToml**: Write custom content to config.toml before update, verify it is preserved after update
- **TestUpdateCommand_StackFlagOverrides**: `--stack` flag changes stacks in output and persists to `.agentbox.yml`
- **TestUpdateCommand_ExtraDomainsFlag**: `--extra-domains` flag updates domain lists and persists to `.agentbox.yml`
- **TestUpdateCommand_NoRuntimeVersionFlag**: Verify the command does not accept `--runtime-version` flag
- **TestUpdateCommand_RegeneratesAgentboxStage**: Change stacks via `--stack`, verify new LSP appears in Dockerfile agentbox stage

#### Updated existing tests:
- `internal/render/dockerfile_test.go`: All `FROM debian:bookworm-slim` assertions updated to `FROM debian:bookworm-slim AS agentbox` (affects `TestDockerfile_BaseImage`, `TestDockerfile_EmptyConfig`, `TestDockerfile_DirectConfig_MinimalValid`)
- `internal/render/devcontainer_test.go`: `TestDevContainer_FixedStructure` asserts `build["target"] == "custom"`
- `cmd/init_test.go`: Dockerfile content in init tests now includes custom stage stub; add assertion that Dockerfile contains `FROM agentbox AS custom`
- `cmd/init_integration_test.go`: Dockerfile assertions check for both stages; devcontainer.json assertions check for `"target": "custom"`; `TestIntegration_SingleGoStack` verifies Dockerfile has both `AS agentbox` and `FROM agentbox AS custom`; `TestDockerfile_UserAndWorkdir` -- last non-empty line is still `WORKDIR /workspace` (it is at the end of the agentbox stage, now followed by the custom stage comment block, so this test needs to check the agentbox stage ends with WORKDIR or the full file's last non-empty line is the last comment line of the custom stage stub)

#### New integration tests (`cmd/update_integration_test.go`):
- **TestIntegration_UpdatePreservesCustomizations**: Full round-trip: init with `--stack go`, add `RUN echo custom` to custom stage, edit config.toml to set `go = "1.23"`, run update, verify custom RUN line preserved, verify config.toml preserved with `go = "1.23"`, verify agentbox stage freshly rendered
- **TestIntegration_UpdateWithStackChange**: Init with `--stack go`, update with `--stack go,node`, verify node LSP appears in Dockerfile, verify `.agentbox.yml` has both stacks
- **TestIntegration_UpdateForceMode**: Init, delete custom stage line from Dockerfile, update `--force`, verify fresh custom stage stub generated containing `FROM agentbox AS custom`
- **TestIntegration_UpdateIdempotent**: Init, update with same config, verify output is byte-identical (determinism), specifically `agentboxPart + userPart` round-trips correctly

### Open Questions

None -- all design decisions have been resolved via the challenge feedback. The plan incorporates all four challenge findings.
