---
# ccbox-d39x
title: Stack metadata registry
status: todo
type: task
priority: high
created_at: 2026-04-02T10:34:21Z
updated_at: 2026-04-02T10:34:21Z
parent: ccbox-2n15
---

## Description
Define a registry of stack metadata. Each stack entry includes:

- **Name**: Display name (e.g., "Go", "Node/TypeScript")
- **Runtime**: mise tool name + version strategy (e.g., `go latest`, `node lts`)
- **LSP**: Language server package + install method (e.g., `gopls` via `go install`, `typescript-language-server` via npm)
- **LSP Plugin**: Claude Code plugin identifier (e.g., `gopls-lsp`, `typescript-lsp`)
- **Default domains**: Package registry domains to allowlist (e.g., `proxy.golang.org`, `registry.npmjs.org`)
- **Dynamic domains**: Domains that need dnsmasq (changing IPs)
- **VS Code extensions**: None for v1 (Claude Code only, added separately)

This registry is the single source of truth for all stack-specific behavior.