#!/bin/bash
# Claude Code PreToolUse dispatcher for Bash tool calls
#
# Parses JSON input once and dispatches to individual check functions.
# Exit codes: 0 = allow, 2 = block (message via stderr)
#
# Note: No `set -e` -- we use explicit error handling to avoid exit code 1
# which Claude Code treats as a hook error (allowing the tool call to proceed).

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
export PROJECT_DIR

# Source all check functions (only function definitions, no side effects)
source "$SCRIPT_DIR/checks/env-access.sh"
source "$SCRIPT_DIR/checks/commit.sh"
source "$SCRIPT_DIR/checks/push.sh"
source "$SCRIPT_DIR/checks/branch-name.sh"
source "$SCRIPT_DIR/checks/unchecked-completion.sh"

# Parse JSON input once
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""' 2>/dev/null || echo "")

if [ -z "$COMMAND" ]; then
    exit 0
fi

FIRST_LINE=$(echo "$COMMAND" | head -1)

# 1. Always run env-access check (applies to all Bash commands)
check_env_access_bash "$COMMAND"

# 2. Conditional checks based on pattern matches on the first line.
case "$FIRST_LINE" in
    *git\ commit*)
        check_pre_commit "$COMMAND" "$FIRST_LINE"
        ;;
    *git\ push*)
        check_pre_push "$COMMAND" "$FIRST_LINE"
        ;;
    *git\ checkout*|*git\ switch*)
        check_branch_name "$COMMAND" "$FIRST_LINE"
        ;;
esac

# 3. Bean completion checks
if [[ "$COMMAND" =~ beans[[:space:]]+update[[:space:]] ]] && [[ "$COMMAND" =~ --status[[:space:]]+completed ]]; then
    check_unchecked_completion "$COMMAND"
fi

exit 0
