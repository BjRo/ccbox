---
# ccbox-7qvl
title: Dockerfile template
status: todo
type: task
priority: high
created_at: 2026-04-02T10:35:05Z
updated_at: 2026-04-02T10:35:05Z
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