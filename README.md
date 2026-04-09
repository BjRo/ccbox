# agentbox

Generate devcontainer setups for running Claude Code and Codex CLI in sandboxed environments with full permissions and network isolation.

## Why

Claude Code and Codex CLI work best with full permissions -- file read/write, command execution, and network access. But granting those on your host machine is risky.

agentbox generates a [devcontainer](https://containers.dev/) that gives Claude Code and Codex CLI full permissions inside a network-isolated Docker container. An iptables firewall with domain-level allowlisting ensures the coding tools can only reach explicitly approved domains, making "bypass permissions" mode safe to use.

## Features

- **Auto-detection** -- Scans for marker files (`go.mod`, `package.json`, `Cargo.toml`, etc.) at the project root and one level deep
- **Multi-stack support** -- Go, Node/TypeScript, Python, Rust, and Ruby
- **Network isolation** -- iptables default-DROP policy with ipset allowlist and dnsmasq for dynamic domains
- **Interactive wizard** -- TUI for stack selection and domain configuration (powered by [charmbracelet/huh](https://github.com/charmbracelet/huh))
- **Non-interactive mode** -- `--non-interactive` / `-y` flag for CI pipelines and scripting
- **Settings sync** -- Copies host Claude Code settings with jq deep-merge; copies Codex CLI config on first run
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
Generated .devcontainer/ with 11 files and .agentbox.yml
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
| `--runtime-version` | | string slice | none | Runtime version overrides as `tool=version` pairs (e.g., `--runtime-version go=1.22,node=20`) |

### `agentbox update`

Regenerate agentbox-managed devcontainer files while preserving user customizations.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--stack` | | string slice | from `.agentbox.yml` | Override stacks (persists to `.agentbox.yml`). Reuses recorded stacks if omitted. |
| `--extra-domains` | | string slice | from `.agentbox.yml` | Override extra domains (persists to `.agentbox.yml`) |
| `--dir` | | string | current directory | Target project directory |
| `--force` | | bool | `false` | Force full regeneration even if the Dockerfile custom stage is missing |

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
- **Claude Code and Codex CLI** via `npm install -g @anthropic-ai/claude-code @openai/codex`
- **Sandbox runtime**: bubblewrap (required by Codex CLI sandbox mode)
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
                  sync-codex-settings.sh
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
                   Claude Code and Codex CLI ready
                    (network limited to allowlist)
```

**Static domains** (e.g., `sentry.io`, `registry.npmjs.org`) have stable IPs. They are resolved once at startup and cached in an ipset.

**Dynamic domains** (e.g., `*.anthropic.com`, `proxy.golang.org`) use CDNs or rotating IPs. They are managed by dnsmasq, which re-resolves them on TTL expiry and updates the ipset automatically.

The always-on allowlist includes domains required for Claude Code and Codex CLI to function regardless of stack:

| Domain | Category | Purpose |
|--------|----------|---------|
| `github.com` | dynamic | GitHub web and git-over-HTTPS |
| `api.github.com` | dynamic | GitHub REST API |
| `*.anthropic.com` | dynamic | Anthropic API for Claude Code |
| `api.openai.com` | dynamic | OpenAI API for Codex CLI |
| `auth.openai.com` | dynamic | OpenAI auth for Codex ChatGPT login flow |
| `auth0.openai.com` | dynamic | OpenAI auth0 for Codex ChatGPT token refresh |
| `chatgpt.com` | dynamic | ChatGPT for Codex ChatGPT login auth flow |
| `accounts.openai.com` | dynamic | OpenAI accounts for Codex ChatGPT auth |
| `sentry.io` | static | Error reporting for Claude Code |
| `statsig.com` | static | Feature flags and experimentation for Claude Code |

Each detected stack adds its own domains (package registries, module proxies, etc.).

### Settings Sync

`sync-claude-settings.sh` copies the generated `claude-user-settings.json` into `~/.claude/settings.json` inside the container. On first run it creates the file; on subsequent runs it uses jq to deep-merge new settings with existing ones, preserving any manual changes.

`sync-codex-settings.sh` copies the generated `codex-config.toml` into `~/.codex/config.toml` inside the container. On first run it creates the file; on subsequent runs it skips the copy to preserve any manual changes (copy-on-first-run strategy, unlike the Claude Code deep-merge approach).

### devcontainer.json

The generated `devcontainer.json` configures:

- **containerEnv** forwards `OPENAI_API_KEY` from the host for Codex CLI authentication
- **Mounts** for bash history, Claude config, Codex config, GitHub CLI config, and gitconfig
- **`postStartCommand`** that chains settings sync (Claude Code and Codex CLI) and firewall initialization
- **Capabilities**: `NET_ADMIN` and `NET_RAW` (required for iptables/ipset)
- **Security**: `seccomp=unconfined` (required for iptables inside the container)
- **Extensions**: Claude Code (`anthropic.claude-code`) and Codex (`openai.chatgpt`) VS Code extensions are auto-configured

## Generated Files

Running `agentbox init` creates a `.devcontainer/` directory and a `.agentbox.yml` config file.

### `.devcontainer/`

| File | Description |
|------|-------------|
| `Dockerfile` | Container image with runtimes, LSPs, Claude Code, Codex CLI, and firewall tooling |
| `devcontainer.json` | VS Code / DevPod configuration with mounts, capabilities, and startup commands |
| `init-firewall.sh` | Network isolation setup script (runs as root via `sudo`) |
| `warmup-dns.sh` | Pre-resolves dynamic domains through dnsmasq after firewall init |
| `dynamic-domains.conf` | Editable list of dynamic domains for dnsmasq |
| `claude-user-settings.json` | Claude Code settings with bypass permissions mode and LSP plugins |
| `sync-claude-settings.sh` | Copies/merges Claude Code settings into the container |
| `codex-config.toml` | Codex CLI settings with full-auto approval policy and sandbox mode |
| `sync-codex-settings.sh` | Copies Codex CLI settings into the container (first-run only) |
| `mise-config.toml` | Runtime version configuration for mise (Go, Node, etc.) |
| `README.md` | Per-project documentation for the generated devcontainer |

### `.agentbox.yml`

Created in the project root. Records the stacks, extra domains, generation timestamp, and agentbox version used. This file enables future `agentbox` commands to understand the current configuration.

## FAQ

### How do I change runtime versions?

Edit `.devcontainer/mise-config.toml`:

```toml
[tools]
go = "1.22"
node = "20"
```

Then rebuild the container. This file is preserved across `agentbox update` runs -- it is never overwritten.

During initial setup, you can also use the `--runtime-version` flag:

```bash
agentbox init --runtime-version go=1.22,node=20
```

### How do I add my own tools?

Add `RUN` commands in the custom stage at the bottom of `.devcontainer/Dockerfile`:

```dockerfile
FROM agentbox AS custom

RUN go install github.com/user/tool@latest && mise reshim
RUN pip install my-tool && mise reshim
RUN npm install -g some-cli
```

The custom stage is preserved when you run `agentbox update`. The agentbox stage above it is regenerated. Add `&& mise reshim` after `go install` or `pip install` so the binaries are discoverable on PATH.

### How do I allow additional domains through the firewall?

Use the `--extra-domains` flag on `init` or `update`:

```bash
agentbox init -y --extra-domains api.example.com,cdn.example.com
agentbox update --extra-domains api.example.com
```

The `-y` flag is required on `init` because in interactive mode the wizard prompts for extra domains and overrides the `--extra-domains` flag. In non-interactive mode (`-y`), the flag is applied directly.

Extra domains are saved in `.agentbox.yml` and reused on subsequent `agentbox update` runs (unless overridden with `--extra-domains`). All user-specified domains are classified as dynamic and managed by dnsmasq with automatic re-resolution.

### How do I pass API keys into the container?

Add entries to `containerEnv` in `.devcontainer/devcontainer.json`:

```json
"containerEnv": {
  "OPENAI_API_KEY": "${localEnv:OPENAI_API_KEY}",
  "ANTHROPIC_API_KEY": "${localEnv:ANTHROPIC_API_KEY}"
}
```

The `${localEnv:VAR}` syntax forwards the variable from your host. Note that `devcontainer.json` is regenerated by `agentbox update`, so you will need to re-add custom entries after updating. To make this easier, version-control the file and use `git diff` to identify and restore your custom entries after an update.

### How do I update after changing stacks?

```bash
agentbox update --stack go,node
```

This regenerates the agentbox-managed portion of `.devcontainer/` for the new stack combination while preserving your custom stage in the Dockerfile and `mise-config.toml`. The new stacks are saved to `.agentbox.yml`. Without `--stack`, the update reuses whatever stacks are recorded in `.agentbox.yml`.

### What's safe to edit vs. what gets overwritten?

| File | On `agentbox update` |
|------|---------------------|
| `Dockerfile` (custom stage) | **Preserved** -- your `FROM agentbox AS custom` block is kept intact |
| `mise-config.toml` | **Preserved** -- your version pins are never overwritten |
| `Dockerfile` (agentbox stage) | Regenerated |
| `devcontainer.json` | Regenerated |
| `init-firewall.sh`, `warmup-dns.sh` | Regenerated |
| `sync-claude-settings.sh`, `sync-codex-settings.sh` | Regenerated |
| `claude-user-settings.json`, `codex-config.toml` | Regenerated |
| `dynamic-domains.conf` | Regenerated |
| `.devcontainer/README.md` | Regenerated |
| `.agentbox.yml` | Updated (stacks, domains, timestamp) |

### How do I add extra VS Code extensions?

Add extension IDs to the `customizations.vscode.extensions` array in `.devcontainer/devcontainer.json`:

```json
"customizations": {
  "vscode": {
    "extensions": [
      "anthropic.claude-code",
      "openai.chatgpt",
      "esbenp.prettier-vscode",
      "dbaeumer.vscode-eslint"
    ]
  }
}
```

Note that `devcontainer.json` is regenerated by `agentbox update`, so custom extensions will need to be re-added after updating. To make this easier, version-control the file and use `git diff` to identify and restore your custom entries after an update.

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
