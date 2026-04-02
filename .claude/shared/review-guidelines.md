# Shared Review Guidelines

## Engineering Preferences Calibration

@.claude/shared/engineering-calibration.md

## Comment Format

- **Be specific**: Point to exact lines, reference file paths
- **Be constructive**: Explain *why*, not just *what*

### Severity Levels

- `CRITICAL:` — Security issues, data loss, correctness bugs. Must fix.
- `WARNING:` — Performance, missing error handling, design concerns. Should fix.
- `SUGGESTION:` — Style improvements, nice-to-haves. Consider fixing.
- `QUESTION:` — Clarification needed. Please explain.

### Options-Based Findings (CRITICAL and WARNING only)

Present 2-3 options with effort/risk for each.

## General Rules

- Don't nitpick formatting issues linters catch
- Acknowledge good code briefly
- Number findings sequentially (WARNING #1, WARNING #2)
