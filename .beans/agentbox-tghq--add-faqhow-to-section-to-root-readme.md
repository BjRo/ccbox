---
# agentbox-tghq
title: Add FAQ/How-to section to root README
status: todo
type: task
priority: normal
created_at: 2026-04-09T10:11:08Z
updated_at: 2026-04-09T10:11:13Z
---

Add a practical FAQ/How-to section to README.md covering common customization tasks:

1. **How do I change runtime versions?** — edit mise-config.toml, run agentbox update
2. **How do I add my own tools?** — add RUN commands in the custom stage (FROM agentbox AS custom)
3. **How do I allow additional domains through the firewall?** — --extra-domains flag or .agentbox.yml
4. **How do I pass API keys into the container?** — containerEnv in devcontainer.json
5. **How do I update after changing stacks?** — agentbox update --stack go,node
6. **What's safe to edit vs what gets overwritten?** — custom stage preserved, agentbox stage regenerated, mise-config.toml preserved
7. **How do I add extra VS Code extensions?** — edit devcontainer.json customizations section

Keep it concise and practical — each entry should be a question followed by a short answer with a code snippet or file reference.

## Definition of Done

- [ ] Tests written (TDD: write tests before implementation)
- [ ] No new TODO/FIXME/HACK/XXX comments introduced
- [ ] `golangci-lint run ./...` passes with no errors
- [ ] `go test ./...` passes with no failures
- [ ] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review
