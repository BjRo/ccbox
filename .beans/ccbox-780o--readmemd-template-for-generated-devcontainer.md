---
# ccbox-780o
title: README.md template for generated devcontainer
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:35:29Z
updated_at: 2026-04-02T10:35:29Z
parent: ccbox-6z26
---

## Description
Generate a comprehensive README.md inside .devcontainer/ that explains:

1. **Overview**: What this devcontainer does and why (Claude Code sandbox with network isolation)
2. **Prerequisites**: Docker, VS Code with Dev Containers extension
3. **Getting Started**: How to open the project in the devcontainer
4. **Firewall Architecture**: How iptables + ipset + dnsmasq work together, default DROP policy
5. **Adding Domains**: How to add new domains to the allowlist (static vs dynamic, edit dynamic-domains.conf)
6. **Claude Code Permissions**: Explains bypass mode and why it is safe inside the sandbox
7. **Settings Sync**: How sync-claude-settings.sh works and why it is needed
8. **Customization**: How to add port forwards, env vars, VS Code extensions, services
9. **Troubleshooting**: Common issues (firewall blocking needed domain, permission errors, volume issues)

Template should include the detected stacks and their specific domains for context.