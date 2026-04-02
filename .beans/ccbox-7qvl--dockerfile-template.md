---
# ccbox-7qvl
title: Dockerfile template
status: completed
type: task
priority: high
created_at: 2026-04-02T10:35:05Z
updated_at: 2026-04-02T15:55:29Z
parent: ccbox-6z26
---

## Description
Create a Go template for the generated Dockerfile. Based on the credfolio2 reference but parameterized:

**Always included (static):**
- Base: `debian:bookworm-slim`
- System packages: curl, git, sudo, zsh, gh, iptables, ipset, iproute2, dnsutils, dnsmasq, build-essential, jq, fzf
- Locale: en_US.UTF-8
- Mise installation from official apt repo
- `node` user (UID 1000) with passwordless sudo
- Claude Code via npm global install
- QoL: zsh-in-docker, git-delta, fzf
- Firewall scripts: COPY + chmod + sudoers

**Parameterized by stack:**
- mise.toml content (runtimes per stack)
- LSP server installations (gopls, typescript-language-server, pyright, rust-analyzer, solargraph)
- Stack-specific system deps (e.g., Ruby needs libssl-dev, libreadline-dev)

**Template variables:**
- `Stacks []Stack` — detected stacks with runtime/LSP info
- `MiseTools map[string]string` — tool→version for mise.toml
- `ExtraDomains []string` — user-specified domains for dynamic-domains.conf

Use Go embed (`//go:embed`) for template files.

## Implementation Plan

### Approach

Add a Dockerfile Go template to the `internal/render/` package, embedded via `//go:embed`, and a `Dockerfile` rendering function that takes `GenerationConfig` and returns the rendered Dockerfile as a string. The template consumes the existing `GenerationConfig` struct directly -- no new data types are needed for the template itself. However, the `stack.Stack` type needs a new `SystemDeps []string` field to capture per-stack apt packages (e.g., Ruby's `libssl-dev`, `libreadline-dev`), and `GenerationConfig` needs a corresponding merged `SystemDeps []string` field.

The template is structured as a single `.tmpl` file in a `templates/` subdirectory under `internal/render/`. This keeps template source separate from Go source and scales naturally as sibling beans (ccbox-dttd, ccbox-v9jt, ccbox-v1zh, ccbox-780o) add more templates.

Testing uses structural assertions on the rendered output (checking that expected blocks/lines are present) rather than golden files. Golden files are brittle for templates that depend on registry data that evolves. A small number of spot-checks assert exact substrings for well-known sections (base image, specific LSP install commands).

### Files to Create/Modify

- `internal/render/templates/Dockerfile.tmpl` (CREATE) -- The Go template file for the Dockerfile
- `internal/render/dockerfile.go` (CREATE) -- Embeds the template, exposes the `Dockerfile(GenerationConfig) (string, error)` function
- `internal/render/dockerfile_test.go` (CREATE) -- Tests for template rendering
- `internal/stack/stack.go` (MODIFY) -- Add `SystemDeps []string` field to the `Stack` struct and populate it for relevant stacks
- `internal/stack/stack_test.go` (MODIFY) -- Add tests for the new `SystemDeps` field (defensive copy, data integrity)
- `internal/render/render.go` (MODIFY) -- Add `SystemDeps []string` to `GenerationConfig`, merge system deps in `Merge()`

### Steps

#### 1. Add `SystemDeps` to the stack registry

**File**: `internal/stack/stack.go`

Add a `SystemDeps []string` field to the `Stack` struct. This lists apt packages required to build the runtime from source (e.g., Ruby needs `libssl-dev`, `libreadline-dev`, `libyaml-dev`, `zlib1g-dev`). Most stacks will have an empty (non-nil) slice.

Populate the field in the registry:
- `Go`: `[]string{}` (no extra deps)
- `Node`: `[]string{}` (no extra deps)
- `Python`: `[]string{"libssl-dev", "zlib1g-dev", "libbz2-dev", "libreadline-dev", "libsqlite3-dev", "libffi-dev"}` (mise builds CPython from source)
- `Rust`: `[]string{}` (rustup handles everything)
- `Ruby`: `[]string{"libssl-dev", "libreadline-dev", "libyaml-dev", "zlib1g-dev"}` (mise builds Ruby from source)

Update `copyStack()` to also clone `SystemDeps` via `slices.Clone`.

**File**: `internal/stack/stack_test.go`

Add a test that verifies `SystemDeps` is cloned (mutation of returned slice does not corrupt registry). Add a test that each stack's `SystemDeps` is non-nil (follows non-nil empty slice convention).

#### 2. Add `SystemDeps` to `GenerationConfig` and `Merge()`

**File**: `internal/render/render.go`

Add `SystemDeps []string` to `GenerationConfig`. In `Merge()`, after collecting runtimes and LSPs, collect system deps from each unique stack, deduplicate by string value (a system dep like `libssl-dev` might appear for both Python and Ruby), sort, and assign to the config. Follow the same pattern: `seen` map, `slices.SortFunc`, ensure non-nil empty slice.

**File**: `internal/render/render_test.go`

Add a test `TestMerge_SystemDeps` that:
- Verifies Go-only yields empty (but non-nil) system deps
- Verifies Ruby yields its expected system deps
- Verifies Go+Ruby+Python deduplicates shared deps (e.g., `libssl-dev` appears in both Ruby and Python, but only once in the merged output)

#### 3. Create the Dockerfile template

**File**: `internal/render/templates/Dockerfile.tmpl`

The template structure (top to bottom):

```
# --- Base image ---
FROM debian:bookworm-slim

# --- Locale ---
ENV LANG=en_US.UTF-8
RUN apt-get update && apt-get install -y locales \
    && sed -i '/en_US.UTF-8/s/^# //' /etc/locale.gen \
    && locale-gen

# --- System packages (always installed) ---
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl git sudo zsh ca-certificates gnupg \
    iptables ipset iproute2 dnsutils dnsmasq \
    build-essential jq fzf \
    {{- range .SystemDeps }}
    {{ . }} \
    {{- end }}
    && rm -rf /var/lib/apt/lists/*

# --- GitHub CLI (gh) ---
RUN (curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg) \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=...] ..." \
    > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update && apt-get install -y gh && rm -rf /var/lib/apt/lists/*

# --- Create non-root user ---
RUN groupadd --gid 1000 node \
    && useradd --uid 1000 --gid node --shell /bin/zsh --create-home node \
    && echo 'node ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers.d/node

# --- Mise (runtime manager) ---
RUN curl https://mise.jdx.dev/install.sh | sh
COPY <<'MISE' /home/node/.config/mise/config.toml
[tools]
{{- range .Runtimes }}
{{ .Tool }} = "{{ .Version }}"
{{- end }}
MISE

# --- Install runtimes via mise ---
USER node
RUN mise install
USER root

# --- LSP servers ---
{{- range .LSPs }}
RUN {{ .InstallCmd }}
{{- end }}

# --- Claude Code ---
RUN npm install -g @anthropic-ai/claude-code

# --- QoL: zsh-in-docker, git-delta ---
RUN sh -c "$(curl -fsSL https://github.com/deluan/zsh-in-docker/releases/download/v1.2.1/zsh-in-docker.sh)" -- \
    -t robbyrussell
RUN curl -fsSL https://github.com/dandavison/delta/releases/download/... \
    | dpkg -i /tmp/delta.deb

# --- Firewall scripts ---
COPY init-firewall.sh /usr/local/bin/init-firewall.sh
COPY warmup-dns.sh /usr/local/bin/warmup-dns.sh
COPY dynamic-domains.conf /etc/dnsmasq.d/dynamic-domains.conf
RUN chmod +x /usr/local/bin/init-firewall.sh /usr/local/bin/warmup-dns.sh

USER node
WORKDIR /workspace
```

The exact commands in the template are illustrative above. The real template will use correct apt repo URLs, pinned delta versions, etc. The key parameterized sections are:
- `{{ range .SystemDeps }}` -- inserts stack-specific apt packages into the system packages RUN
- `{{ range .Runtimes }}` -- generates mise.toml content
- `{{ range .LSPs }}` -- generates LSP install RUN commands

#### 4. Create the rendering function and embed

**File**: `internal/render/dockerfile.go`

```go
package render

import (
    "bytes"
    "embed"
    "text/template"
)

//go:embed templates/Dockerfile.tmpl
var dockerfileTemplate string

// Dockerfile renders the Dockerfile template using the given GenerationConfig.
// It returns the rendered content as a string.
func Dockerfile(cfg GenerationConfig) (string, error) {
    tmpl, err := template.New("Dockerfile").Parse(dockerfileTemplate)
    if err != nil {
        return "", fmt.Errorf("render: parse Dockerfile template: %w", err)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, cfg); err != nil {
        return "", fmt.Errorf("render: execute Dockerfile template: %w", err)
    }

    return buf.String(), nil
}
```

Design notes:
- Embed a single template string rather than an `embed.FS`. This is simpler for a single file per function and avoids the `ParseFS` overhead. If sibling beans later introduce shared template helpers (partials), we can switch to `embed.FS` with `ParseFS` at that point.
- Parse on every call (not `sync.Once`) for simplicity. Template parsing is fast for small templates and there is no hot loop. If profiling shows this matters, caching can be added later.
- The function name `Dockerfile` matches the ADR-0002 convention (`render.DevContainer(...)` pattern) -- call sites read as `render.Dockerfile(cfg)`.

#### 5. Write the tests

**File**: `internal/render/dockerfile_test.go`

Tests use structural assertions on the rendered string. Each test calls `Merge()` to produce a `GenerationConfig`, then `Dockerfile()` to render, then checks the output.

**Test cases:**

1. **`TestDockerfile_BaseImage`** -- Render with any config (e.g., Go only). Assert output `strings.Contains` `"FROM debian:bookworm-slim"`. Verifies the static base is always present.

2. **`TestDockerfile_AlwaysIncludedPackages`** -- Assert the rendered output contains key always-on system packages: `curl`, `git`, `sudo`, `zsh`, `iptables`, `dnsmasq`, `build-essential`. Not a full apt-get line match -- just that these package names appear in the output.

3. **`TestDockerfile_MiseToolsSingleStack`** -- Merge with `[Go]`, render. Assert the mise config section contains `go = "latest"`. Assert it does NOT contain `node`, `python`, etc.

4. **`TestDockerfile_MiseToolsMultiStack`** -- Merge with `[Go, Node, Python]`, render. Assert mise config section contains all three tool entries: `go = "latest"`, `node = "lts"`, `python = "latest"`.

5. **`TestDockerfile_LSPInstallCommands`** -- Merge with `[Go]`, render. Assert output contains `"go install golang.org/x/tools/gopls@latest"`. Merge with `[Go, Node]`, render. Assert output also contains `"npm install -g typescript-language-server typescript"`.

6. **`TestDockerfile_SystemDepsIncluded`** -- Merge with `[Ruby]`, render. Assert output contains `libssl-dev` and `libreadline-dev`. Merge with `[Go]`, render. Assert output does NOT contain `libssl-dev`.

7. **`TestDockerfile_SystemDepsDeduplication`** -- Merge with `[Ruby, Python]`, render. Assert that `libssl-dev` appears exactly once (both stacks declare it). Use `strings.Count`.

8. **`TestDockerfile_EmptyConfig`** -- Merge with `[]` (no stacks), render. Assert it still produces a valid Dockerfile with the base image, system packages, user creation, and Claude Code install. Assert no mise tools are listed. Assert no LSP installs are present.

9. **`TestDockerfile_FirewallScriptsCopied`** -- Assert output contains `COPY init-firewall.sh`, `COPY warmup-dns.sh`, `COPY dynamic-domains.conf`.

10. **`TestDockerfile_UserAndWorkdir`** -- Assert the final lines switch to `USER node` and `WORKDIR /workspace`.

11. **`TestDockerfile_ClaudeCodeInstall`** -- Assert output contains `npm install -g @anthropic-ai/claude-code`.

12. **`TestDockerfile_NoTrailingWhitespace`** -- Split output by `\n`, assert no line has trailing spaces or tabs. Template rendering is notorious for producing trailing whitespace via `{{- }}` misuse. This guards against it.

### Testing Strategy

- **Structural assertions over golden files**: Each test checks for the presence (or absence) of specific substrings in the rendered output. This is resilient to template formatting changes and registry data evolution.
- **Spot-checks tied to registry data**: Tests for mise tools and LSP installs pull expected values from `stack.Get()` so they stay in sync with the registry. If a version changes in the registry, the test automatically adapts.
- **No golden file maintenance burden**: Golden files require updating whenever the template or registry changes. With structural assertions, only the semantic contract is tested.
- **Edge case: empty config**: Ensures the template renders cleanly even when no stacks are detected (all `{{range}}` blocks produce nothing).
- **Edge case: trailing whitespace**: Catches a common Go template pitfall.

### Open Questions

1. **Exact `mise install` invocation**: Mise's CLI may require the user context (`USER node`) and specific `PATH` setup. The real template will need to ensure `mise` is on the `PATH` when running `mise install`. This is a template content detail, not an architectural decision -- it will be resolved during implementation by testing the generated Dockerfile in a real Docker build. For now, the plan assumes `mise` is installed to a standard location (e.g., `/usr/local/bin/mise` or `~/.local/bin/mise`).

2. **Pinned versions for QoL tools**: The bean mentions zsh-in-docker and git-delta. The template should pin these to specific release versions to avoid nondeterministic builds. The exact versions will be chosen during implementation.

3. **Stack-specific system deps accuracy**: The Python and Ruby system deps listed above are educated guesses for mise source builds. These will be validated during implementation by building test containers. If a dep is wrong, the fix is a registry data change, not an architecture change.

## Definition of Done

- [x] Tests written (TDD)
- [x] No TODO/FIXME/HACK/XXX comments in new code
- [x] Lint passes (`golangci-lint run ./...`)
- [x] Tests pass (`go test ./...`)
- [x] Branch pushed
- [x] PR created (https://github.com/BjRo/ccbox/pull/8)
- [x] Automated code review passed (`@review-backend`)
- [x] Review feedback worked in (ccbox-9ium)
- [ ] ADR written (if architectural changes)
- [ ] User notified

## Agent Checkpoint

All implementation steps complete. Steps completed:
1. Added `SystemDeps` field to `stack.Stack` with registry data for all 5 stacks
2. Added `SystemDeps` to `GenerationConfig` and `Merge()` with dedup/sort logic
3. Created `Dockerfile.tmpl` template with correct whitespace handling
4. Created `Dockerfile()` rendering function with `//go:embed`
5. Wrote 22 tests total (5 stack, 5 merge, 17 dockerfile -- some overlap with existing)
6. All tests pass, lint clean

Next: push branch.

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-02T17:30:00Z |
| challenge | complete | 1 | 2026-04-02T17:35:00Z |
| implement | complete | 1 | 2026-04-02T18:00:00Z |
| pr | complete | 1 | 2026-04-02T18:10:00Z |
| review | complete | 1 | 2026-04-02T18:20:00Z |
| rework | complete | 1 | 2026-04-02T19:00:00Z |
| codify | pending | | |