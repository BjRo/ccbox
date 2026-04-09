---
# agentbox-31g0
title: Update root README.md for Codex CLI support
status: todo
type: task
created_at: 2026-04-09T09:03:43Z
updated_at: 2026-04-09T09:03:43Z
---

The root README.md is outdated after the Codex CLI epic. Needs updates:

1. Intro and features list — mention Codex CLI alongside Claude Code
2. Dockerfile section — add @openai/codex and bubblewrap
3. Firewall always-on table — add all OpenAI domains (api.openai.com, auth.openai.com, auth0.openai.com, chatgpt.com, accounts.openai.com)
4. Settings sync section — document Codex copy-on-first-run strategy
5. devcontainer.json section — add containerEnv (OPENAI_API_KEY), Codex volume mount, Codex extension
6. Generated files table — add codex-config.toml, sync-codex-settings.sh, update config.toml to mise-config.toml
7. Go version — fix 1.25+ to 1.24+
