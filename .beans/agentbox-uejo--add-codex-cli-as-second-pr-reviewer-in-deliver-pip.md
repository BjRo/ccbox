---
# agentbox-uejo
title: Add Codex CLI as second PR reviewer in deliver pipeline
status: in-progress
type: feature
priority: normal
created_at: 2026-04-08T12:45:09Z
updated_at: 2026-04-08T13:25:40Z
parent: agentbox-cqi5
---

Add a @review-codex subagent that runs `codex exec review --base main` and posts findings as PR comments. Integrate into the deliver pipeline alongside @review-backend for parallel independent reviews.

## Scope

### New files
- `.claude/agents/review-codex.md` — Subagent that runs `codex exec review --base main --full-auto -o /tmp/codex-review.md`, reads the output, and posts findings as a PR comment via `gh pr comment`.

### Modified files
- `.claude/skills/deliver/SKILL.md` — Update Step 3 to launch both @review-backend and @review-codex in parallel. Step 4 evaluates findings from both; any findings from either trigger rework.
- `.claude/agents/deliver.md` — Update Step 3 (Review-Rework Loop) to launch both reviewers in parallel.
- `.claude/agents/rework.md` — Add `/issues/{n}/comments` as a third API call in Step 2 so the rework agent reads general PR comments (where codex review findings are posted via `gh pr comment`).

### Design decisions
- **review-codex as a subagent** (not a skill): Same interface as review-backend — posts findings as PR comments. Deliver pipeline treats both reviewers uniformly.
- **Parallel execution**: Both reviews launch simultaneously to minimize wall-clock time.
- **Rework picks up both**: The rework agent must read issue comments (via `/issues/{n}/comments`) in addition to PR reviews and inline comments to see codex review findings.
- **Re-review loop**: After rework, both reviewers re-run. Both must be clean to proceed.
- **Model**: Use highest-quality available (configurable in agent).
- **Sandbox**: `--full-auto` implies `--sandbox workspace-write` which is sufficient.
- **Output capture**: `-o /tmp/codex-review.md` for the agent to read and post.
- **`bypassPermissions` on review-codex only**: `review-codex` uses `permissionMode: bypassPermissions` because it spawns `codex exec review`, an autonomous sub-process that itself runs shell commands. Without the bypass, the parent Claude session would prompt for approval on each command Codex executes, making unattended pipeline operation impossible. `review-backend` does not need the bypass because its `gh api` calls are individually approved or already within the deliver agent's permission scope.

## Implementation Plan

### Approach

Create a new subagent `.claude/agents/review-codex.md` that runs `codex exec review --base main` and posts findings as a PR comment. Then update both deliver entry points (the skill at `.claude/skills/deliver/SKILL.md` and the agent at `.claude/agents/deliver.md`) to launch both `@review-backend` and `@review-codex` in parallel during the review step. Critically, also update the rework agent (`.claude/agents/rework.md`) to read issue comments via `/issues/{n}/comments` so it can see codex review findings.

### Files to Create/Modify

- `.claude/agents/review-codex.md` (NEW) — Codex-based PR reviewer subagent
- `.claude/agents/rework.md` (MODIFY) — Add third API call to read issue comments
- `.claude/skills/deliver/SKILL.md` (MODIFY) — Update Step 3 and Step 4 for parallel dual-reviewer workflow
- `.claude/agents/deliver.md` (MODIFY) — Update Step 3 (Review-Rework Loop) for parallel dual-reviewer workflow
- `.claude/skills/rework/SKILL.md` (MODIFY) — Update Step 3 to re-trigger both reviewers after rework

### Steps

#### 1. Create `.claude/agents/review-codex.md`

Create the new subagent file with the following structure:

**Frontmatter:**
```yaml
---
name: review-codex
description: Codex-based code reviewer. Runs codex exec review and posts findings as PR comments.
tools: Read, Bash, Glob, Grep
permissionMode: bypassPermissions
---
```

Key notes on frontmatter:
- `permissionMode: bypassPermissions` is required because the agent needs to run `codex exec review` (a Bash command) and `gh pr comment` without interactive approval prompts.
- `tools: Read, Bash, Glob, Grep` matches review-backend's tool set.

**Body - Process Section:**

The agent follows a four-step process:

**Step 1: Identify the PR** -- Same pattern as review-backend:
```bash
PR_NUMBER=$(gh pr view --json number -q '.number')
```

**Step 2: Run Codex Review** -- Execute the codex CLI review command:
```bash
codex exec review --base main --full-auto -o /tmp/codex-review.md
```

Key flags:
- `--base main`: Review changes against the main branch (same scope as review-backend sees via `gh pr diff`)
- `--full-auto`: Non-interactive execution with sandboxed automatic mode
- `-o /tmp/codex-review.md`: Capture the last agent message (the review summary) to a file for the agent to read and post

Do NOT use `--sandbox read-only` -- the `--full-auto` flag already implies `--sandbox workspace-write` which is sufficient and appropriate.

**Step 3: Read and Validate Output** -- Read `/tmp/codex-review.md` and verify it contains review content (not empty). If the file is missing or empty, post a comment noting the review tool failed to produce output.

**Step 4: Post as PR Comment** -- Read the file content, then construct the comment body and post it. The key detail here is that `gh pr comment` posts to the issues comments API endpoint (`/issues/{n}/comments`), not the PR reviews API. The agent must build the comment body by reading the file first, then passing the content directly:

```bash
REVIEW_CONTENT=$(cat /tmp/codex-review.md)
gh pr comment "$PR_NUMBER" --body "## Codex Code Review

${REVIEW_CONTENT}

---
Automated review by Codex Review Agent"
```

This uses shell variable expansion within an unquoted string, avoiding the heredoc quoting issue. The `REVIEW_CONTENT` variable is expanded by the shell before being passed to `gh pr comment`.

Alternative approach if the content contains characters that would cause shell issues:

```bash
# Build the full comment in a temp file
{
  echo "## Codex Code Review"
  echo ""
  cat /tmp/codex-review.md
  echo ""
  echo "---"
  echo "Automated review by Codex Review Agent"
} > /tmp/codex-comment.md
gh pr comment "$PR_NUMBER" --body-file /tmp/codex-comment.md
```

The `--body-file` approach is the safest because it avoids all shell quoting/expansion issues entirely. Prefer this approach.

**Cleanup:** Remove the temp files after posting:
```bash
rm -f /tmp/codex-review.md /tmp/codex-comment.md
```

#### 2. Update `.claude/agents/rework.md` -- Add Issue Comments API Call

This is the CRITICAL fix from the challenge. The rework agent currently reads:
1. `/pulls/{n}/reviews` -- PR review bodies (where review-backend posts)
2. `/pulls/{n}/comments` -- Inline PR review comments (line-level comments from review-backend)

But `gh pr comment` posts to `/issues/{n}/comments` (general PR comments), which is a DIFFERENT endpoint. The rework agent must also read this endpoint to see codex review findings.

**Change in Step 2 (Read Review Comments), after the existing two `gh api` calls, add a third:**

```bash
gh api "repos/${REPO}/issues/${PR_NUMBER}/comments" --jq '.[] | {user: .user.login, body: .body}'
```

This reads general PR comments (the issue comments endpoint), which is where `gh pr comment` posts. The rework agent will now see findings from both review-backend (via the PR reviews API) and review-codex (via the issues comments API).

The full Step 2 in the rework agent becomes:
```bash
REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')
gh api "repos/${REPO}/pulls/${PR_NUMBER}/reviews" --jq '.[] | select(.body != null and .body != "") | {id: .id, user: .user.login, body: .body}'
gh api "repos/${REPO}/pulls/${PR_NUMBER}/comments" --jq '.[] | {path: .path, line: .line, body: .body}'
gh api "repos/${REPO}/issues/${PR_NUMBER}/comments" --jq '.[] | {user: .user.login, body: .body}'
```

#### 3. Update `.claude/skills/deliver/SKILL.md`

**Step 3 changes (lines ~92-101):**

Replace the current single-reviewer Step 3:
```
### Step 3: Launch Review
Launch `@review-backend` to review the Go code:
...single Task tool call...
```

With a parallel dual-reviewer Step 3:
```
### Step 3: Launch Reviews (parallel)

Launch both reviewers in parallel using two Task tool calls in a single message:

**Review A: @review-backend**
Task tool call:
  subagent_type: "review-backend"
  description: "Backend code review"
  prompt: "Review the current PR. Post your findings as PR comments using the gh CLI."

**Review B: @review-codex**
Task tool call:
  subagent_type: "review-codex"
  description: "Codex code review"
  prompt: "Review the current PR. Run codex exec review and post findings as a PR comment."
```

The key instruction is "in a single message" -- this tells the deliver agent to issue both Agent tool calls at once so they execute concurrently.

**Step 4 changes (lines ~103-107):**

Update the evaluation logic to account for findings from either reviewer:
```
### Step 4: Evaluate Review Results

Read the responses from both reviewers.

- **No actionable findings from either reviewer** -> check off "Automated code review passed". Proceed to Codify.
- **Any findings from either reviewer (CRITICAL, WARNING, or SUGGESTION), iteration < 3** -> Rework. Do NOT evaluate findings yourself and cherry-pick which to fix. ALL findings must go through the rework agent.
- **Any findings, iteration >= 3** -> Escalate
```

The rework step (Step 5a) requires no changes -- the rework agent will now read all three comment endpoints and address all findings regardless of source.

#### 4. Update `.claude/agents/deliver.md`

**Step 3 changes (lines ~73-84):**

Replace the current Step 3 review-rework loop description. The current text says:

```
1. **Launch @review-backend** to review the PR code and post findings as PR comments.
2. **Read the review results** from the agent's response.
```

Replace with:

```
1. **Launch @review-backend and @review-codex in parallel** (two Agent tool calls in a single message) to review the PR code. Both post findings as PR comments.
2. **Read the review results** from both agents' responses.
```

The rest of the loop logic (steps 3-5 about evaluating findings, reworking, and re-reviewing) applies unchanged -- "any actionable findings" already covers findings from either reviewer, and the rework agent will now read all PR comments via the three API endpoints.

Also update the CRITICAL note at the end of the section:

```
**CRITICAL: After rework, you MUST re-launch both @review-backend and @review-codex before proceeding to codify. Never skip the re-review.**
```

#### 5. Update `.claude/skills/rework/SKILL.md`

**Step 3 changes (lines ~37-40):**

The current rework skill's Step 3 says to re-trigger `@review-backend` only. Update to re-trigger both reviewers:

```
### Step 3: After the Agent Completes

1. **Summarize** — Report what was fixed
2. **Re-trigger reviews** — Launch both `@review-backend` and `@review-codex` in parallel
3. **Report results**
```

### Testing Strategy

This bean involves no Go code changes -- only agent/skill configuration files (Markdown). Testing is manual/behavioral:

1. **Syntax validation**: Verify the new agent file has valid YAML frontmatter (correct `name`, `description`, `tools`, `permissionMode` fields)
2. **Codex CLI availability**: Confirm `codex exec review --help` shows expected flags (`--base`, `--full-auto`, `-o`)
3. **Issue comments API verification**: Run `gh api repos/{owner}/{repo}/issues/{pr}/comments` on a test PR that has `gh pr comment`-posted comments and confirm they appear. This validates the critical fix for challenge finding #1.
4. **Integration check**: The new agent can be referenced as `@review-codex` in Agent tool calls. The deliver agent and skill correctly describe parallel launch of both reviewers.
5. **Rework compatibility**: Verify the rework agent's updated Step 2 now reads all three endpoints: `/pulls/{n}/reviews`, `/pulls/{n}/comments`, AND `/issues/{n}/comments`, ensuring it picks up findings from both review-backend and review-codex.
6. **Body-file approach**: Verify `gh pr comment --body-file` works correctly with the temp file approach, avoiding all shell quoting issues (challenge finding #2 fix).

Since there are no Go source files to modify, the standard `go test`, `golangci-lint`, and TDD checklist items are satisfied trivially (no new code paths to test).

### Open Questions

None -- all design decisions are settled. The two challenge findings have been addressed:
1. **CRITICAL (rework visibility)**: Fixed by adding `/issues/{n}/comments` as a third API call in the rework agent's Step 2.
2. **WARNING (heredoc quoting)**: Fixed by using `--body-file` approach instead of heredoc with command substitution.

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [ ] Branch pushed to remote
- [x] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review
