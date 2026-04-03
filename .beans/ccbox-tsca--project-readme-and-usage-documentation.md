---
# ccbox-tsca
title: Project README and usage documentation
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:36:14Z
updated_at: 2026-04-03T10:15:00Z
parent: ccbox-vydo
---

## Description
Write the ccbox project README.md:

- **Tagline**: One-line description of what ccbox does
- **Why**: Motivation — run Claude Code with full permissions safely inside a sandboxed devcontainer
- **Features**: Auto-detection, firewall, multi-stack, interactive wizard
- **Installation**: Homebrew (`brew install bjro/tap/ccbox`) and GitHub releases
- **Quick Start**: `cd my-project && ccbox init` walkthrough with example output
- **CLI Reference**: All flags and subcommands
- **Supported Stacks**: Table of stacks with their runtimes, LSPs, and default domains
- **Architecture**: How the generated devcontainer works (diagram of firewall, dnsmasq, Claude Code)
- **Contributing**: How to build from source, run tests, submit PRs
- **License**: MIT

## Checklist

- [x] README.md written with all sections
- [x] No TODO/FIXME/HACK/XXX in code
- [x] Lint passes
- [x] Tests pass
- [x] Branch pushed
- [x] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-03 |
| challenge | complete | 1 | 2026-04-03 |
| implement | complete | 1 | 2026-04-03 |
| pr | complete | 1 | 2026-04-03 |
| review | pending | | |
| codify | pending | | |

## Implementation Plan

### Approach

Create a single `README.md` file at the project root. This is a pure documentation task with no source code changes -- the README is hand-authored Markdown, not a generated template. The content will be structured to serve two audiences: users who want to install and use ccbox (top half: tagline, motivation, installation, quick start, CLI reference, supported stacks) and contributors who want to build from source or understand the architecture (bottom half: architecture, how it works, contributing, license).

All factual claims in the README (flag names, stack IDs, marker files, domain lists, CLI output) will be derived from the actual codebase explored during refinement, not invented. This avoids documentation drift from day one.

### Files to Create/Modify

- `README.md` (new) -- The project README at repository root. All content in a single file.

### Steps

1. **Write the header and tagline**
   - Project name: `ccbox`
   - One-line tagline: "Generate devcontainer setups for running Claude Code in sandboxed environments with full permissions and network isolation."
   - This mirrors the `Long` description from `cmd/root.go` line 23.

2. **Write the "Why" / motivation section**
   - Problem: Claude Code works best with full permissions (file read/write, command execution, network access), but granting those on your host machine is risky.
   - Solution: ccbox generates a devcontainer that gives Claude Code full permissions inside a network-isolated Docker container.
   - Key insight: The firewall ensures Claude Code can only reach explicitly allowlisted domains, making "bypass permissions" mode safe.

3. **Write the "Features" section**
   - Bullet list of capabilities, each grounded in actual codebase features:
     - **Auto-detection**: Scans for marker files (`go.mod`, `package.json`, `Cargo.toml`, etc.) at root and one level deep (from `internal/detect/detect.go`).
     - **Multi-stack support**: 5 stacks: Go, Node/TypeScript, Python, Rust, Ruby (from `internal/stack/stack.go` registry).
     - **Network isolation**: iptables default-DROP + ipset allowlist + dnsmasq for dynamic domains (from `init-firewall.sh.tmpl`).
     - **Interactive wizard**: charmbracelet/huh-based TUI for stack selection and domain configuration (from `internal/wizard/wizard.go`).
     - **Non-interactive mode**: `--non-interactive` / `-y` flag for CI pipelines (from `cmd/init.go` line 189).
     - **Claude Code settings sync**: Copies user settings into container with deep merge (from `sync-claude-settings.sh.tmpl`).
     - **LSP plugin configuration**: Auto-configures Claude Code LSP plugins per detected stack (from `claude-user-settings.json.tmpl`).
     - **Runtime management via mise**: Installs language runtimes via mise (from `Dockerfile.tmpl`).

4. **Write the "Installation" section**
   - Homebrew: `brew install bjro/tap/ccbox`
   - GitHub Releases: Direct binary download (link to releases page)
   - From source: `go install github.com/bjro/ccbox@latest`
   - Note: The GoReleaser config does not exist yet (sibling bean `ccbox-xeg2`), so the Homebrew and releases instructions are forward-looking but the `go install` path works today.

5. **Write the "Quick Start" section**
   - Show a minimal 3-step workflow:
     1. `cd my-project`
     2. `ccbox init` (interactive wizard) or `ccbox init -y` (auto-detect, no prompts)
     3. Open in VS Code and "Reopen in Container"
   - Show the 8 files generated in `.devcontainer/` plus `.ccbox.yml` (derived from `cmd/init.go` lines 139-148 and 166-179):
     - `Dockerfile`
     - `devcontainer.json`
     - `init-firewall.sh`
     - `warmup-dns.sh`
     - `dynamic-domains.conf`
     - `claude-user-settings.json`
     - `sync-claude-settings.sh`
     - `README.md`
   - Include a brief example of what `ccbox init -y` prints to stderr (the "Stacks: [go]" and "Generated .devcontainer/ with 8 files" messages from `cmd/init.go` lines 100 and 181).

6. **Write the "CLI Reference" section**
   - Document `ccbox init` with all flags:
     - `--stack` (StringSliceVar, comma-separated, e.g., `--stack go,node`) -- overrides auto-detection
     - `--extra-domains` (StringSliceVar, comma-separated) -- additional domains to allowlist
     - `--dir` (string, default: current directory) -- target project directory
     - `--non-interactive` / `-y` (bool) -- skip all prompts
   - Document `ccbox --version` (prints version string)
   - Note: Flag details derived from `cmd/init.go` lines 186-189.

7. **Write the "Supported Stacks" section**
   - A table with columns: Stack ID, Display Name, Runtime (tool@version), LSP, Marker Files.
   - Data derived from `internal/stack/stack.go` registry (lines 72-154):

     | Stack | Name | Runtime | LSP | Marker Files |
     |-------|------|---------|-----|--------------|
     | `go` | Go | go@latest | gopls | `go.mod` |
     | `node` | Node/TypeScript | node@lts | typescript-language-server | `package.json`, `tsconfig.json` |
     | `python` | Python | python@latest | pyright | `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile` |
     | `rust` | Rust | rust@latest | rust-analyzer | `Cargo.toml` |
     | `ruby` | Ruby | ruby@latest | solargraph | `Gemfile`, `*.gemspec` |

   - Note for Ruby: `*.gemspec` is a glob pattern from `internal/detect/detect.go` line 31, not the stack registry.
   - Mention that Node/npm is always included regardless of detected stacks (Claude Code requires it).

8. **Write the "How It Works" / Architecture section**
   - Describe the generated devcontainer's components:
     - **Dockerfile**: debian:bookworm-slim base, mise for runtimes, LSP servers, Claude Code via npm, iptables/ipset/dnsmasq packages, zsh/git-delta for developer experience.
     - **Firewall**: Three-layer architecture (ASCII diagram):
       1. iptables default-DROP on OUTPUT chain
       2. ipset hash:ip allowlist for static domains (resolved once at startup)
       3. dnsmasq with ipset integration for dynamic domains (CDN/rotating IPs)
     - **Settings sync**: `sync-claude-settings.sh` copies `claude-user-settings.json` into the container's `~/.claude/settings.json`, using jq deep-merge on subsequent runs.
     - **devcontainer.json**: Mounts for bash history, Claude config, gh config, gitconfig. `postStartCommand` chains settings sync and firewall init. Requires `NET_ADMIN` + `NET_RAW` capabilities.
   - Show a simple text-based architecture diagram illustrating the flow: container start -> postStartCommand -> sync settings -> init firewall -> (resolve static domains, configure dnsmasq, set iptables DROP) -> Claude Code ready.

9. **Write the "Generated Files" section**
   - Brief description of each file generated in `.devcontainer/`:
     - `Dockerfile` -- Container image definition with runtimes, LSPs, and tooling
     - `devcontainer.json` -- VS Code / DevPod configuration
     - `init-firewall.sh` -- Network isolation setup script (runs as root via sudo)
     - `warmup-dns.sh` -- Pre-resolves dynamic domains through dnsmasq
     - `dynamic-domains.conf` -- Editable list of dynamic domains for dnsmasq
     - `claude-user-settings.json` -- Claude Code settings template (bypass mode, LSP plugins)
     - `sync-claude-settings.sh` -- Copies/merges settings into container
     - `README.md` -- Per-project documentation for the generated devcontainer
   - Plus `.ccbox.yml` in the project root (records stacks, extra domains, generation timestamp, ccbox version).

10. **Write the "Contributing" section**
    - Prerequisites: Go 1.24+, golangci-lint v2
    - Build: `go build ./...`
    - Test (unit): `go test ./...`
    - Test (integration): `go test -tags integration ./...`
    - Lint: `golangci-lint run ./...`
    - Mention the ADR process (decisions directory) for architectural changes.
    - Brief note on the project structure (`cmd/`, `internal/stack/`, `internal/detect/`, `internal/render/`, `internal/firewall/`, `internal/config/`, `internal/wizard/`).

11. **Write the "License" section**
    - State MIT license.
    - Note: There is no LICENSE file in the repository yet. The bean body says MIT. The LICENSE file creation may be part of this task or the GoReleaser sibling bean. Include the license section in the README referencing MIT, and create a `LICENSE` file with the standard MIT text alongside the README.

### Content Guidelines

- Use concise, scannable prose. Prefer bullet lists and tables over paragraphs.
- Use fenced code blocks with language hints (`bash`, `yaml`, `json`) for all examples.
- Do not include badges (no CI, no coverage, no version badge) -- the project is pre-release.
- Do not include a changelog section -- that belongs in CHANGELOG.md or GitHub releases.
- Keep the total length under ~400 lines. The README should be comprehensive but not exhaustive; link to generated `.devcontainer/README.md` for per-project details.
- All flag names, stack IDs, file names, and domain lists must match the actual codebase. Cross-reference against the source files listed in this plan.

### Section Order

1. Title + tagline
2. Why (motivation)
3. Features
4. Installation
5. Quick Start
6. CLI Reference
7. Supported Stacks
8. How It Works (architecture)
9. Generated Files
10. Contributing
11. License

### Testing Strategy

This is a documentation-only task. There are no code changes, so no new tests are needed. Verification steps:

- **Factual accuracy**: Every claim in the README should be verifiable against the source files referenced in this plan.
- **Link validity**: All internal links (to files, sections) should be correct.
- **Markdown rendering**: Verify the README renders correctly on GitHub by pushing the branch and checking the rendered output.
- **Lint and test pass**: Run `golangci-lint run ./...` and `go test ./...` to confirm no regressions (should be no-ops since no code changes).
- **No TODO/FIXME/HACK/XXX**: Scan the README for prohibited markers.

### Open Questions

- **LICENSE file**: Should this bean also create the LICENSE file (MIT), or does that belong to the GoReleaser sibling bean (ccbox-xeg2)? Decision: include it in this bean since the README references MIT and having a LICENSE file is table stakes for any open-source project README.
- **GoReleaser not yet configured**: The Homebrew install command (`brew install bjro/tap/ccbox`) and GitHub Releases link are forward-looking. They should still be documented but may need a note that the first release is pending.

## Challenge Report

**Scope: SMALL CHANGE** (1 file: `README.md`, plus `LICENSE`)

### Scope Assessment

| Metric | Value | Threshold |
|--------|-------|-----------|
| Files | 2 | >15 = recommend split |

### Findings

#### Go Engineer

> **Finding 1: Dual domain registries create conflicting source of truth for README content** (severity: WARNING)
>
> The plan's Step 7 (Supported Stacks table) and Step 8 (Architecture/How It Works) will need to reference domain data, but the codebase has **two separate domain registries** that disagree:
>
> - `internal/stack/stack.go` has `DefaultDomains` and `DynamicDomains` fields per stack (e.g., Node lists `registry.yarnpkg.com` as dynamic, Go lists `proxy.golang.org` as a default/static domain).
> - `internal/firewall/firewall.go` has its own curated registry with different domain lists and different static/dynamic classifications (e.g., Node has `cdn.jsdelivr.net` and `unpkg.com` as dynamic instead of `registry.yarnpkg.com`; Go domains are all classified as Dynamic, not Static).
>
> The firewall registry is what actually drives rendered output via `firewall.Merge()` (called from `render.Merge()`). The stack registry's domain fields are marked as "provisional placeholders" in code comments. If the README documents specific domains, it must use the firewall registry as the source of truth, not the stack registry.
>
> The plan's Step 7 table avoids domains (good), but the bean description says "Table of stacks with their runtimes, LSPs, and default domains." The implementer could interpret that as needing a domains column. The plan should be explicit about which registry to reference if domains are mentioned anywhere.
>
> **Option A (recommended):** During implementation, source any domain references from `internal/firewall/firewall.go`, not `internal/stack/stack.go`. The Step 7 table is correct as planned (no domain column). If domains are mentioned in the Architecture section (Step 8), reference the firewall registry's actual categories.
>
> **Option B:** Add a separate "Domain Allowlists" subsection under Step 8 that lists the always-on domains and per-stack domains from the firewall registry, making it clear these are the enforced allowlists.

> **Finding 2: "Generated Files" section (Step 9) duplicates "Quick Start" section (Step 5)** (severity: SUGGESTION)
>
> Steps 5 and 9 both list the same 8 files in `.devcontainer/` plus `.ccbox.yml`. The Quick Start section (Step 5) enumerates all 8 files, and the Generated Files section (Step 9) lists them again with descriptions. This redundancy adds ~20 lines to a README that targets a 400-line cap.
>
> **Suggestion:** In Step 5 (Quick Start), show only the command output (e.g., the "Generated .devcontainer/ with 8 files and .ccbox.yml" message) without enumerating individual files. Let Step 9 (Generated Files) be the single canonical file manifest with descriptions. This saves space and avoids two places to update when the file list changes.

### Verdict

**APPROVED**

The plan is solid. The section ordering is effective for the target audience (users first, contributors second). Flag names, stack IDs, marker files, and file lists are factually accurate against the codebase. The open questions are resolved correctly -- creating the LICENSE file alongside the README is the right call, and noting the forward-looking Homebrew/releases instructions is appropriate.

The only substantive concern (Finding 1) is about which domain registry to use as the source of truth. This is a WARNING because getting it wrong would produce a README that contradicts the actual firewall behavior, but the plan's Step 7 table already avoids listing domains, so the risk is limited to any domain references in the Architecture section. The implementer should use `internal/firewall/firewall.go` as the canonical source for any domain-related content.
