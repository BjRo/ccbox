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

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [ ] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-09T10:12:00Z |
| challenge | done | 1 | 2026-04-09T10:16:00Z |
| implement | done | 1 | 2026-04-09T10:20:00Z |
| pr | in-progress | 1 | 2026-04-09T10:20:00Z |
| review | pending | | |
| codify | pending | | |

## Implementation Plan

### Approach

Add a new `## FAQ` section to `README.md` placed between the existing `## Generated Files` section and the `## Contributing` section (i.e., after line 200, before line 202 in the current file). The section contains 7 question-and-answer entries covering common customization tasks. Each answer is grounded in actual codebase behavior verified during refinement.

### Files to Create/Modify

- `README.md` — Add `## FAQ` section between `## Generated Files` and `## Contributing`

No other files are modified. This is a documentation-only change.

### Steps

1. **Add the `## FAQ` section header after `## Generated Files`** — Insert the new section starting at the blank line between the `.agentbox.yml` paragraph (line 200) and `## Contributing` (line 202). The section uses `## FAQ` as a level-2 heading consistent with the rest of the document.

2. **Write FAQ entry 1: "How do I change runtime versions?"** — Answer: Edit `.devcontainer/mise-config.toml` directly (it uses `[tools]` with `tool = "version"` pairs like `go = "latest"`, `node = "lts"`). Then rebuild the container. On `agentbox update`, `mise-config.toml` is explicitly preserved and NOT regenerated (see `cmd/update.go` lines 98-125). On `agentbox init`, the `--runtime-version` flag accepts `tool=version` pairs (e.g., `--runtime-version go=1.22,node=20`). Note: `--runtime-version` exists only on `init`, not on `update`.

   ```markdown
   ### How do I change runtime versions?

   Edit `.devcontainer/mise-config.toml`:

   ```toml
   [tools]
   go = "1.22"
   node = "20"
   ```

   Then rebuild the container. This file is preserved across `agentbox update` runs — it is never overwritten.

   During initial setup, you can also use the `--runtime-version` flag:

   ```bash
   agentbox init --runtime-version go=1.22,node=20
   ```
   ```

3. **Write FAQ entry 2: "How do I add my own tools?"** — Answer: Add `RUN` commands in the custom stage of `.devcontainer/Dockerfile` (the `FROM agentbox AS custom` stage at the bottom). This stage is preserved by `agentbox update`. Include `&& mise reshim` after `go install` or `pip install` so binaries land on PATH. Reference the comments already present in the generated custom stage template (`internal/render/templates/custom-stage.tmpl`).

   ```markdown
   ### How do I add my own tools?

   Add `RUN` commands in the custom stage at the bottom of `.devcontainer/Dockerfile`:

   ```dockerfile
   FROM agentbox AS custom

   RUN go install github.com/user/tool@latest && mise reshim
   RUN pip install my-tool && mise reshim
   RUN npm install -g some-cli
   ```

   The custom stage is preserved when you run `agentbox update`. The agentbox stage above it is regenerated. Add `&& mise reshim` after `go install` or `pip install` so the binaries are discoverable on PATH.
   ```

4. **Write FAQ entry 3: "How do I allow additional domains through the firewall?"** — Answer: Two methods. (a) At generation time: `--extra-domains` flag on both `agentbox init` and `agentbox update`. (b) Stored in `.agentbox.yml` under `extra_domains` — `agentbox update` reads from this file when `--extra-domains` is not passed. User extras are classified as dynamic domains (managed by dnsmasq with periodic re-resolution). Reference `internal/firewall/merge.go` line 83 for the "always Dynamic" classification.

   ```markdown
   ### How do I allow additional domains through the firewall?

   Use the `--extra-domains` flag on `init` or `update`:

   ```bash
   agentbox init --extra-domains api.example.com,cdn.example.com
   agentbox update --extra-domains api.example.com
   ```

   Extra domains are saved in `.agentbox.yml` and reused on subsequent `agentbox update` runs (unless overridden with `--extra-domains`). All user-specified domains are classified as dynamic and managed by dnsmasq with automatic re-resolution.
   ```

5. **Write FAQ entry 4: "How do I pass API keys into the container?"** — Answer: Add environment variables to the `containerEnv` object in `.devcontainer/devcontainer.json`. The generated file already forwards `OPENAI_API_KEY`. Use the `${localEnv:VAR}` syntax to forward host environment variables. Note: `devcontainer.json` is regenerated by `agentbox update`, so edits here will be overwritten — this is a trade-off to document.

   ```markdown
   ### How do I pass API keys into the container?

   Add entries to `containerEnv` in `.devcontainer/devcontainer.json`:

   ```json
   "containerEnv": {
     "OPENAI_API_KEY": "${localEnv:OPENAI_API_KEY}",
     "ANTHROPIC_API_KEY": "${localEnv:ANTHROPIC_API_KEY}"
   }
   ```

   The `${localEnv:VAR}` syntax forwards the variable from your host. Note that `devcontainer.json` is regenerated by `agentbox update`, so you will need to re-add custom entries after updating.
   ```

6. **Write FAQ entry 5: "How do I update after changing stacks?"** — Answer: Run `agentbox update --stack go,node` to change the stack list. This regenerates the agentbox stage of the Dockerfile, firewall scripts, and domain configuration for the new stacks, while preserving the custom stage and `mise-config.toml`. The new stacks are persisted to `.agentbox.yml`. Without `--stack`, update reuses the stacks from `.agentbox.yml`.

   ```markdown
   ### How do I update after changing stacks?

   ```bash
   agentbox update --stack go,node
   ```

   This regenerates the agentbox-managed portion of `.devcontainer/` for the new stack combination while preserving your custom stage in the Dockerfile and `mise-config.toml`. The new stacks are saved to `.agentbox.yml`. Without `--stack`, the update reuses whatever stacks are recorded in `.agentbox.yml`.
   ```

7. **Write FAQ entry 6: "What's safe to edit vs. what gets overwritten?"** — Answer: Create a table showing each file and its update behavior. Preserved: custom stage in Dockerfile, `mise-config.toml`. Regenerated: agentbox stage in Dockerfile, `devcontainer.json`, all `.sh` scripts, `dynamic-domains.conf`, `claude-user-settings.json`, `codex-config.toml`, `.devcontainer/README.md`. The `.agentbox.yml` in the project root is also updated with new timestamps/stacks.

   ```markdown
   ### What's safe to edit vs. what gets overwritten?

   | File | On `agentbox update` |
   |------|---------------------|
   | `Dockerfile` (custom stage) | **Preserved** — your `FROM agentbox AS custom` block is kept intact |
   | `mise-config.toml` | **Preserved** — your version pins are never overwritten |
   | `Dockerfile` (agentbox stage) | Regenerated |
   | `devcontainer.json` | Regenerated |
   | `init-firewall.sh`, `warmup-dns.sh` | Regenerated |
   | `sync-claude-settings.sh`, `sync-codex-settings.sh` | Regenerated |
   | `claude-user-settings.json`, `codex-config.toml` | Regenerated |
   | `dynamic-domains.conf` | Regenerated |
   | `.devcontainer/README.md` | Regenerated |
   | `.agentbox.yml` | Updated (stacks, domains, timestamp) |
   ```

8. **Write FAQ entry 7: "How do I add extra VS Code extensions?"** — Answer: Edit the `customizations.vscode.extensions` array in `.devcontainer/devcontainer.json`. The generated file includes `anthropic.claude-code` and `openai.chatgpt` by default. Add extension IDs to the array. Caveat: this file is regenerated by `agentbox update`, so custom extensions will need to be re-added after updating.

   ```markdown
   ### How do I add extra VS Code extensions?

   Add extension IDs to the `customizations.vscode.extensions` array in `.devcontainer/devcontainer.json`:

   ```json
   "customizations": {
     "vscode": {
       "extensions": [
         "anthropic.claude-code",
         "openai.chatgpt",
         "esbenp.prettier-vscode",
         "dbaeumer.vscode-eslint"
       ]
     }
   }
   ```

   Note that `devcontainer.json` is regenerated by `agentbox update`, so custom extensions will need to be re-added after updating.
   ```

9. **Verify formatting** — Ensure the new section uses consistent Markdown style: level-2 heading for the section, level-3 headings for each question, fenced code blocks with language tags, and a blank line between each entry.

### Testing Strategy

- No Go code is modified, so no new tests are needed
- Run `go test ./...` to confirm nothing is broken (README.md changes cannot affect Go tests, but this validates the working tree is clean)
- Run `golangci-lint run ./...` to confirm no lint regressions
- Manual review: verify the new section renders correctly in Markdown and all file/flag references match the actual codebase

### Codebase References

- `cmd/update.go` — Update command implementation; preserves custom stage (line 86) and mise-config.toml (lines 98-125); `--stack` and `--extra-domains` flags
- `cmd/init.go` — Init command; `--runtime-version` flag (line 180-181); `--extra-domains` flag
- `internal/render/templates/custom-stage.tmpl` — Custom stage template with usage examples
- `internal/render/templates/Dockerfile.tmpl` — Agentbox stage template (the "DO NOT EDIT" portion)
- `internal/render/templates/devcontainer.json.tmpl` — Generated devcontainer.json with `containerEnv`, extensions, mounts
- `internal/render/templates/mise-config.toml.tmpl` — Mise config template (`[tools]` section)
- `internal/firewall/merge.go` — Domain merge logic; user extras are always classified as Dynamic (line 83)
- `internal/config/config.go` — `.agentbox.yml` structure with `extra_domains` field

### Challenge Findings (to address during implementation)

1. **WARNING: CLI Reference gap** — FAQ references `agentbox update` and `--runtime-version` which aren't in the existing CLI Reference section. Fix: Add `### agentbox update` subsection to CLI Reference and add `--runtime-version` to the `agentbox init` flags table. Same file, minimal scope increase.

2. **WARNING: No durable workaround for overwritten edits** — FAQ entries 4 (API keys) and 7 (VS Code extensions) tell users to edit `devcontainer.json` which gets overwritten by `agentbox update`. Fix: Add a sentence suggesting users version-control the file and use `git diff` to restore custom entries after update, or keep a note of custom entries.

### Open Questions

- None. All 7 FAQ entries have been verified against the actual codebase behavior. The `agentbox update` command does exist (contrary to the task briefing's caution), and its behavior has been confirmed by reading `cmd/update.go`.
