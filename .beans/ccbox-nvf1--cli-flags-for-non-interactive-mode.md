---
# ccbox-nvf1
title: CLI flags for non-interactive mode
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:34:41Z
updated_at: 2026-04-02T10:34:41Z
parent: ccbox-puuq
---

## Description
All wizard options must be expressible as CLI flags for scripting:

```
ccbox init \
  --stack go,node \
  --extra-domains "api.example.com,cdn.example.com" \
  --non-interactive
```

Flags:
- `--stack <comma-separated>`: Explicitly set stacks (skip detection)
- `--extra-domains <comma-separated>`: Additional domains for firewall allowlist
- `--non-interactive` / `-y`: Skip all prompts, use detected stacks + defaults
- `--dir <path>`: Target directory (default: current directory)

When flags provide all required info, skip the wizard entirely.