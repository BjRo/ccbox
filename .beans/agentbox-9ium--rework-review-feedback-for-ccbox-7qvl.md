---
# agentbox-9ium
title: Rework review feedback for agentbox-7qvl
status: completed
type: task
created_at: 2026-04-02T15:51:27Z
updated_at: 2026-04-02T15:51:27Z
parent: agentbox-6z26
---

Address 2 WARNINGs and 2 SUGGESTIONs from PR #8 code review.

WARNING 1: git-delta .deb URL hardcodes amd64 architecture -- fixed with dpkg --print-architecture.
WARNING 2: mise install runs as root but config is in /home/node/ -- fixed by running mise install as USER node and copying mise to /usr/local/bin.
SUGGESTION 1: npm install -g claude-code assumes Node.js on PATH -- fixed by always including node=lts in mise config.
SUGGESTION 2: curl | sh for mise lacks integrity verification -- added comment noting future hardening concern.