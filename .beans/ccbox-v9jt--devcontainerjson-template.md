---
# ccbox-v9jt
title: devcontainer.json template
status: todo
type: task
priority: high
created_at: 2026-04-02T10:35:10Z
updated_at: 2026-04-02T10:35:10Z
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