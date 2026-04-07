# agentbox

Generate devcontainer setups for running Claude Code in sandboxed environments with full permissions and network isolation.

## Why

Claude Code works best with full permissions -- file read/write, command execution, and network access. But granting those on your host machine is risky.

agentbox generates a [devcontainer](https://containers.dev/) that gives Claude Code full permissions inside a network-isolated Docker container. An iptables firewall with domain-level allowlisting ensures Claude Code can only reach explicitly approved domains, making "bypass permissions" mode safe to use.

## Features

- **Auto-detection** -- Scans for marker files (`go.mod`, `package.json`, `Cargo.toml`, etc.) at the project root and one level deep
- **Multi-stack support** -- Go, Node/TypeScript, Python, Rust, and Ruby
- **Network isolation** -- iptables default-DROP policy with ipset allowlist and dnsmasq for dynamic domains
- **Interactive wizard** -- TUI for stack selection and domain configuration (powered by [charmbracelet/huh](https://github.com/charmbracelet/huh))
- **Non-interactive mode** -- `--non-interactive` / `-y` flag for CI pipelines and scripting
- **Claude Code settings sync** -- Copies host settings into the container with jq deep-merge on subsequent runs
- **LSP plugin configuration** -- Auto-configures Claude Code LSP plugins per detected stack
- **Runtime management via mise** -- Installs language runtimes through [mise](https://mise.jdx.dev/)

## Installation

### Homebrew

```bash
brew install bjro/tap/agentbox
```

### GitHub Releases

Download a pre-built binary from the [releases page](https://github.com/bjro/agentbox/releases).

### From source

```bash
go install github.com/bjro/agentbox@latest
```

> **Note:** Homebrew and GitHub Releases will be available with the first release. Building from source works today.

## Quick Start

```bash
cd my-project
agentbox init          # interactive wizard
# or
agentbox init -y       # auto-detect stacks, no prompts
```

Example output:

```
Stacks: [go]
Generated .devcontainer/ with 8 files and .agentbox.yml
```

Then open the project in VS Code and select **Dev Containers: Reopen in Container**, or use [DevPod](https://devpod.sh/) to launch the container.

See [Generated Files](#generated-files) for details on what gets created.

## CLI Reference

### `agentbox init`

Generate a `.devcontainer/` directory with all configuration files.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--stack` | | string slice | auto-detect | Comma-separated stack IDs (e.g., `--stack go,node`). Overrides auto-detection. |
| `--extra-domains` | | string slice | none | Additional domains to allowlist beyond per-stack defaults (e.g., `--extra-domains api.example.com`) |
| `--dir` | | string | current directory | Target project directory |
| `--non-interactive` | `-y` | bool | `false` | Skip all prompts, use detected stacks and defaults |

### `agentbox --version`

Print the version string and exit.

## Supported Stacks

| Stack | Name | Runtime | LSP | Detection Triggers |
|-------|------|---------|-----|--------------------|
| `go` | Go | go@latest | gopls | `go.mod` |
| `node` | Node/TypeScript | node@lts | typescript-language-server | `package.json`, `tsconfig.json` |
| `python` | Python | python@latest | pyright | `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile` |
| `rust` | Rust | rust@latest | rust-analyzer | `Cargo.toml` |
| `ruby` | Ruby | ruby@latest | solargraph | `Gemfile`, `*.gemspec`\* |

\* `*.gemspec` is detected via glob pattern in the detection engine, not as an exact filename in the stack registry.

Node/npm is always included in the generated container regardless of detected stacks, because Claude Code requires it.

Detection scans the project root and one level deep, skipping `vendor/`, `node_modules/`, `.git/`, `testdata/`, and `.devcontainer/`.

## How It Works

agentbox generates a self-contained devcontainer with three main components: a Dockerfile, a firewall, and a settings sync mechanism.

### Dockerfile

The generated `Dockerfile` uses `debian:bookworm-slim` as the base image and installs:

- **mise** for language runtime management (Go, Node, Python, Rust, Ruby)
- **LSP servers** per detected stack (gopls, pyright, typescript-language-server, etc.)
- **Claude Code** via `npm install -g @anthropic-ai/claude-code`
- **Firewall tooling**: iptables, ipset, dnsmasq
- **Developer experience**: zsh, git-delta, GitHub CLI, fzf

### Firewall

The firewall uses a three-layer architecture to enforce domain-level network isolation:

```
                        Container Startup
                              |
                    postStartCommand runs
                              |
                      chown volumes
                              |
                  sync-claude-settings.sh
                              |
                      init-firewall.sh
                              |
          +-----------------------+-----------------------+
          |                       |                       |
Resolve static domains    Configure dnsmasq        Set iptables
into ipset hash:ip        for dynamic domains      OUTPUT -> DROP
          |                       |                       |
(dig once, cache IPs)     (re-resolve on TTL)     (allow ipset only)
          |                       |                       |
          +-----------------------+-----------------------+
                                  |
                          Claude Code ready
                    (network limited to allowlist)
```

**Static domains** (e.g., `api.github.com`, `registry.npmjs.org`) have stable IPs. They are resolved once at startup and cached in an ipset.

**Dynamic domains** (e.g., `*.anthropic.com`, `proxy.golang.org`) use CDNs or rotating IPs. They are managed by dnsmasq, which re-resolves them on TTL expiry and updates the ipset automatically.

The always-on allowlist includes domains required for Claude Code to function regardless of stack:

| Domain | Category | Purpose |
|--------|----------|---------|
| `github.com` | static | GitHub web and git-over-HTTPS |
| `api.github.com` | static | GitHub REST API |
| `*.anthropic.com` | dynamic | Anthropic API for Claude Code |
| `sentry.io` | static | Error reporting for Claude Code |
| `statsig.com` | static | Feature flags and experimentation for Claude Code |

Each detected stack adds its own domains (package registries, module proxies, etc.).

### Settings Sync

`sync-claude-settings.sh` copies the generated `claude-user-settings.json` into `~/.claude/settings.json` inside the container. On first run it creates the file; on subsequent runs it uses jq to deep-merge new settings with existing ones, preserving any manual changes.

### devcontainer.json

The generated `devcontainer.json` configures:

- **Mounts** for bash history, Claude config, GitHub CLI config, and gitconfig
- **`postStartCommand`** that chains settings sync and firewall initialization
- **Capabilities**: `NET_ADMIN` and `NET_RAW` (required for iptables/ipset)
- **Security**: `seccomp=unconfined` (required for iptables inside the container)

## Generated Files

Running `agentbox init` creates a `.devcontainer/` directory and a `.agentbox.yml` config file.

### `.devcontainer/`

| File | Description |
|------|-------------|
| `Dockerfile` | Container image with runtimes, LSPs, Claude Code, and firewall tooling |
| `devcontainer.json` | VS Code / DevPod configuration with mounts, capabilities, and startup commands |
| `init-firewall.sh` | Network isolation setup script (runs as root via `sudo`) |
| `warmup-dns.sh` | Pre-resolves dynamic domains through dnsmasq after firewall init |
| `dynamic-domains.conf` | Editable list of dynamic domains for dnsmasq |
| `claude-user-settings.json` | Claude Code settings with bypass permissions mode and LSP plugins |
| `sync-claude-settings.sh` | Copies/merges Claude Code settings into the container |
| `README.md` | Per-project documentation for the generated devcontainer |

### `.agentbox.yml`

Created in the project root. Records the stacks, extra domains, generation timestamp, and agentbox version used. This file enables future `agentbox` commands to understand the current configuration.

## Contributing

### Prerequisites

- Go 1.25+
- golangci-lint v2

### Build and Test

```bash
go build ./...                          # Build
go test ./...                           # Unit tests
go test -tags integration ./...         # Unit + integration tests
golangci-lint run ./...                 # Lint
```

### Project Structure

```
cmd/                  CLI commands (Cobra)
internal/
  stack/              Stack metadata registry
  detect/             Stack auto-detection
  render/             Template rendering engine
  firewall/           Domain allowlist logic
  config/             .agentbox.yml handling
  wizard/             Interactive TUI wizard
```

### Architecture Decisions

Significant design choices are documented as Architecture Decision Records in [`decisions/`](decisions/README.md).

## License

MIT. See [LICENSE](LICENSE) for the full text.
