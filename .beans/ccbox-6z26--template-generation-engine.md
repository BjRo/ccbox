---
# ccbox-6z26
title: Template Generation Engine
status: completed
type: epic
priority: normal
created_at: 2026-04-02T10:33:39Z
updated_at: 2026-04-03T08:33:07Z
parent: ccbox-el52
---

Generate all .devcontainer/ files from Go templates. Files: Dockerfile, devcontainer.json, claude-user-settings.json, sync-claude-settings.sh, init-firewall.sh, warmup-dns.sh, dynamic-domains.conf, README.md. Templates are parameterized by detected stacks, user options, and domain allowlists.