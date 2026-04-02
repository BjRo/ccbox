---
name: deliver
description: Automated delivery pipeline that chains refine, challenge, implement, review, and rework into a single command. Runs the full pipeline with human escalation when loops don't converge. Use with a bean ID argument, e.g. /deliver ccbox-abc1
argument-hint: <bean-id>
---

# Deliver Pipeline

Orchestrate the full delivery pipeline for a bean: refine -> challenge -> implement -> PR -> review -> rework -> codify. Automates the loops (refine-challenge and implement-review-rework) with bounded retries and human escalation.

## Prerequisites

- The bean must exist and have a description/body with enough context to refine
- You must NOT be on main (the implement agent will create/use a feature branch)

## Phase 1: Refine-Challenge Loop (max 3 iterations)

### For each iteration:

#### Step 1: Launch Refine

```
Task tool call:
  subagent_type: "refine"
  description: "Refine <bean-title>"
  prompt: |
    Refine bean <BEAN_ID>. Read the bean, explore the codebase, and develop a detailed implementation plan. Update the bean body with the plan.

    IMPORTANT — You are in deliver mode:
    - Use your best judgment for ambiguities instead of asking the user.
    - Only use AskUserQuestion for genuinely blocking choices where making the wrong call would lead to a fundamentally different implementation.
    - If a previous Challenge Report exists in the bean body, read it carefully and incorporate its feedback into your revised plan.
```

#### Step 2: Launch Challenge

```
Task tool call:
  subagent_type: "challenge"
  description: "Challenge <bean-title>"
  prompt: "Challenge the implementation plan for bean <BEAN_ID>. Read the bean, determine which reviewer personas are relevant based on the files being changed, and evaluate the plan against each persona's checklist. Write the Challenge Report to the bean (replacing any existing one)."
```

#### Step 3: Parse Verdict

Read the bean body and find the `### Verdict` line in the `## Challenge Report` section.

- If `APPROVED` -> proceed to Phase 2
- If `NEEDS REVISION` and iteration < 3 -> loop back to Step 1
- If `NEEDS REVISION` and iteration >= 3 -> escalate to user

## Phase 2: Implement-Review-Rework Loop (max 2 iterations)

### Step 1: Launch Implement

```
Task tool call:
  subagent_type: "implement"
  description: "Implement <bean-title>"
  prompt: "Implement bean <BEAN_ID>. Read the bean, follow its implementation plan using TDD, commit as you go, and update the bean checklist. Report what was completed when done."
```

### Updating the Bean's Definition of Done

The implement agent checks off items it completes directly. The deliver orchestrator checks off remaining items:

- After **PR creation** -> check off "PR created"
- After **reviews pass clean** -> check off "Automated code review passed"
- After **rework completes** -> check off "Review feedback worked in"
- After **codify** -> check off "ADR written" (if applicable)
- At **completion** -> check off "All other checklist items above are completed" and "User notified for human review"

### Step 2: Create PR

```bash
gh pr create --title "<type>: <description based on bean>" --body "$(cat <<'EOF'
## Summary
<summary based on implement agent's report>

## Bean
<BEAN_ID>

## Test Plan
- All tests pass (`go test ./...`)
- Lint clean (`golangci-lint run ./...`)

Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

### Step 3: Launch Review

Launch `@review-backend` to review the Go code:

```
Task tool call:
  subagent_type: "review-backend"
  description: "Backend code review"
  prompt: "Review the current PR. Post your findings as PR comments using the gh CLI."
```

### Step 4: Evaluate Review Results

- **No actionable findings** -> check off "Automated code review passed". Proceed to Codify.
- **Actionable findings, iteration 1** -> Rework
- **Actionable findings, iteration 2** -> Escalate

### Step 5a: Rework

```
Task tool call:
  subagent_type: "rework"
  description: "Rework review feedback"
  prompt: "Rework review feedback for bean <BEAN_ID>. Read the PR review comments, address all findings (CRITICAL, WARNING, and SUGGESTION), then push the fixes."
```

After rework, loop back to Step 3 (re-launch review). This is iteration 2.

### Step 5b: Codify Learnings

```
Task tool call:
  subagent_type: "codify"
  description: "Codify learnings from <bean-title>"
  prompt: "Codify learnings from bean <BEAN_ID>. Read the bean body and git diff against main. Identify new patterns or decisions. Write to CLAUDE.md or decisions/. Commit and push."
```

### Step 6: Completion

Check off remaining DoD items. Notify user with PR URL and review summary.

## Important Notes

- **Do NOT mark the bean as completed** — that happens after human merge
- **Do NOT merge the PR** — always wait for human review
- **Do NOT skip reviews**
