---
# ccbox-v9jt
title: devcontainer.json template
status: completed
type: task
priority: high
created_at: 2026-04-02T10:35:10Z
updated_at: 2026-04-02T16:02:24Z
parent: ccbox-6z26
---

## Description
Create a Go template for devcontainer.json. Based on credfolio2 reference, stripped of app-specific config:

**Structure:**
- `build.dockerfile`: Points to `Dockerfile`
- `remoteUser`: `node`
- `customizations.vscode.extensions`: `["anthropics.claude-code"]`
- `mounts`: Bash history volume, Claude config volume, `~/.config/gh` bind mount, `~/.gitconfig` bind mount
- `postStartCommand`: Runs `sync-claude-settings.sh` and `init-firewall.sh`
- `capAdd`: `["NET_ADMIN", "NET_RAW"]` (required for iptables)
- `securityOpt`: `["seccomp=unconfined"]` (required for iptables in some Docker versions)
- `workspaceMount`/`workspaceFolder`: Standard `/workspace` setup

**NOT included (app-specific):**
- No port forwards (user adds their own)
- No containerEnv (user adds their own)
- No docker-compose references
- No custom network

## Implementation Plan

### Approach

Add a Go template file for `devcontainer.json` to the `internal/render/` package, embedded via `//go:embed`, and expose a `DevContainer` rendering function that accepts `GenerationConfig` and writes valid JSON to an `io.Writer`. The template itself is entirely static (no conditional branches based on detected stacks) because all variable parts of the devcontainer configuration are handled by *other* templates (Dockerfile, firewall scripts, Claude settings). The `devcontainer.json` file wires those pieces together with fixed structure.

The one exception is the `remoteUser` field. The bean specifies `"node"` (the default user created in the Dockerfile), and this is correct for the MVP. It is hardcoded in the template since every generated Dockerfile creates the same `node` user regardless of stack.

Since devcontainer.json is strict JSON (no comments, no trailing commas), the template is best written as a `.json.tmpl` file containing a pre-formatted JSON document with zero Go template actions. This is the simplest correct approach: the output is byte-for-byte deterministic, there are no JSON-escaping concerns, and there is nothing in `GenerationConfig` that the devcontainer.json needs to vary on. If future requirements add parameterization (e.g., extra mounts per stack), the template can be upgraded to use `{{.Field}}` actions at that time.

### Files to Create/Modify

- `internal/render/templates/devcontainer.json.tmpl` (NEW) -- The embedded Go template file containing the static devcontainer.json content.
- `internal/render/devcontainer.go` (NEW) -- The `DevContainer` function that parses and executes the template, writing to an `io.Writer`. Also contains the `//go:embed` directive and the `embed.FS` variable for the templates directory.
- `internal/render/devcontainer_test.go` (NEW) -- Tests for the `DevContainer` function.

### Steps

1. **Create the templates directory and template file**

   Create `internal/render/templates/devcontainer.json.tmpl` with the following static JSON content:

   ```json
   {
     "build": {
       "dockerfile": "Dockerfile"
     },
     "remoteUser": "node",
     "customizations": {
       "vscode": {
         "extensions": [
           "anthropics.claude-code"
         ]
       }
     },
     "mounts": [
       "source=ccbox-bash-history,target=/home/node/.bash_history_volume,type=volume",
       "source=ccbox-claude-config,target=/home/node/.claude,type=volume",
       "source=${localEnv:HOME}/.config/gh,target=/home/node/.config/gh,type=bind,consistency=cached",
       "source=${localEnv:HOME}/.gitconfig,target=/home/node/.gitconfig,type=bind,consistency=cached"
     ],
     "postStartCommand": "bash .devcontainer/sync-claude-settings.sh && sudo bash .devcontainer/init-firewall.sh",
     "capAdd": [
       "NET_ADMIN",
       "NET_RAW"
     ],
     "securityOpt": [
       "seccomp=unconfined"
     ],
     "workspaceMount": "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached",
     "workspaceFolder": "/workspace"
   }
   ```

   Key design decisions in the template content:
   - The `mounts` array uses named Docker volumes (`ccbox-bash-history`, `ccbox-claude-config`) for persistent state that should survive container rebuilds (shell history, Claude configuration). The `ccbox-` prefix namespaces them to avoid collisions with other devcontainers.
   - The `~/.config/gh` and `~/.gitconfig` bind mounts use `${localEnv:HOME}` which is the standard devcontainer variable for the host user's home directory, and `consistency=cached` for macOS performance.
   - `postStartCommand` chains two scripts: `sync-claude-settings.sh` (managed by sibling bean ccbox-v1zh) runs as the `node` user, then `init-firewall.sh` (managed by sibling bean ccbox-dttd) runs with `sudo` since iptables requires root.
   - `capAdd` and `securityOpt` are required for the iptables-based firewall to function inside the container.

2. **Create the embed and render function**

   Create `internal/render/devcontainer.go` with:

   - A package-level `//go:embed templates/devcontainer.json.tmpl` directive and an `embed.FS` variable. This is the first embedded template in the project, so this file establishes the pattern that sibling template beans (Dockerfile, firewall scripts, etc.) will follow.
   - A `DevContainer(w io.Writer, cfg GenerationConfig) error` function that:
     1. Parses the embedded template via `text/template.ParseFS`.
     2. Executes the template against `cfg` and writes the result to `w`.
   - Even though the current template has no template actions, the function still accepts `GenerationConfig` as a parameter. This establishes the consistent API signature that all render functions will share, and avoids a breaking change when parameterization is added later.
   - Use `text/template` (not `html/template`) since this is JSON output, not HTML. ADR-0002 already decided the package is named `render` specifically to avoid import shadowing with `text/template`.

3. **Write tests**

   Create `internal/render/devcontainer_test.go` with the following test cases:

   - **`TestDevContainer_ValidJSON`** -- Renders the template with a representative `GenerationConfig` (e.g., Go + Node stacks), then verifies the output is valid JSON by unmarshaling into `map[string]any`. This is the most important structural test: if the template produces invalid JSON, nothing downstream works.

   - **`TestDevContainer_FixedStructure`** -- Unmarshals the output and spot-checks key fields:
     - `build.dockerfile` equals `"Dockerfile"`
     - `remoteUser` equals `"node"`
     - `customizations.vscode.extensions` contains `"anthropics.claude-code"`
     - `capAdd` contains `"NET_ADMIN"` and `"NET_RAW"`
     - `securityOpt` contains `"seccomp=unconfined"`
     - `workspaceFolder` equals `"/workspace"`
     - `mounts` is a 4-element array
     - `postStartCommand` contains both `sync-claude-settings.sh` and `init-firewall.sh`

   - **`TestDevContainer_EmptyConfig`** -- Passes an empty `GenerationConfig{}` (zero stacks) and verifies the output is still valid JSON with the same structure. This confirms the template does not break when no stacks are detected.

   - **`TestDevContainer_MountsContent`** -- Spot-checks the `mounts` array entries for expected substrings: `ccbox-bash-history`, `ccbox-claude-config`, `.config/gh`, `.gitconfig`. Verifies bind mounts use `${localEnv:HOME}`.

   - **`TestDevContainer_Deterministic`** -- Renders twice with the same config and asserts byte-for-byte identical output. This matters because JSON consumer tools may diff the output.

   All tests use `bytes.Buffer` as the `io.Writer`.

### Design Decisions

- **Static template (no Go template actions)**: The devcontainer.json output is identical regardless of detected stacks. All stack-specific variation lives in the Dockerfile (runtimes, LSPs), firewall scripts (domains), and Claude settings (plugins). The devcontainer.json only orchestrates those files via `build.dockerfile`, `postStartCommand`, and `mounts`. Making the template a literal JSON file avoids JSON-escaping pitfalls in Go templates and keeps the output trivially verifiable.

- **`embed.FS` scoped to `templates/` directory**: Using `//go:embed templates/devcontainer.json.tmpl` (embedded into an `embed.FS`) rather than `//go:embed` into a `string` allows future template files (Dockerfile.tmpl, init-firewall.sh.tmpl, etc.) to share the same `embed.FS` variable. Sibling beans can add their templates to the same `templates/` directory and the embed directive can be widened to `templates/*` when the second template lands.

- **Function signature `DevContainer(w io.Writer, cfg GenerationConfig) error`**: The `io.Writer` parameter follows Go conventions for rendering functions (like `text/template.Execute`). Accepting `GenerationConfig` even though it is unused today keeps the API uniform across all render functions and avoids a signature change when parameterization is needed.

- **No ADR needed**: This bean introduces embedded templates and a rendering function, but the embedding pattern is standard Go (`//go:embed` + `embed.FS`) and the render package name was already decided in ADR-0002. No new architectural decision is being made.

### Testing Strategy

- All tests are pure unit tests using `bytes.Buffer` -- no filesystem I/O, no temp files.
- The primary safety net is `json.Unmarshal` validation: if the template ever produces invalid JSON (e.g., trailing comma added by mistake), the test fails immediately.
- Spot-check assertions verify structural correctness of specific fields that downstream tools depend on (capabilities, security options, workspace path).
- Determinism test catches accidental introduction of non-deterministic template behavior.

### Open Questions

None. The bean description is sufficiently prescriptive, and the template content is entirely static for the MVP.

## Checklist

- [x] Tests written
- [x] No TODO/FIXME/HACK/XXX comments
- [x] Lint passes
- [x] Tests pass
- [x] Branch pushed
- [x] PR created
- [x] Automated code review passed
- [x] Review feedback worked in
- [x] ADR written (if applicable)
- [x] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|----------|
| refine | complete | 1 | 2026-04-02 |
| challenge | completed | 1 | 2026-04-02 | | |
| implement | completed | 1 | 2026-04-02 | | |
| pr | completed | 1 | 2026-04-02 | | |
| review | completed | 1 | 2026-04-02 | | |
| rework | completed | 1 | 2026-04-02 | | |
| codify | completed | 1 | 2026-04-02 | | |