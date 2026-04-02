---
# ccbox-tsca
title: Project README and usage documentation
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:36:14Z
updated_at: 2026-04-02T10:36:14Z
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