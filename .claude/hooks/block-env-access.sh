#!/bin/bash
# Claude Code PreToolUse hook: Blocks access to .env files containing secrets
#
# Input: JSON via stdin with tool_name and tool_input
# Output: Exit 0 to allow, Exit 2 to block (message via stderr)
#
# Note: No `set -e` — we use explicit error handling to avoid exit code 1

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/checks/env-access.sh"

block_with_message() {
    block_env_with_message "$@"
}

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // ""' 2>/dev/null || echo "")

if [ "$TOOL_NAME" = "Grep" ]; then
    GREP_PATH=$(echo "$INPUT" | jq -r '.tool_input.path // ""' 2>/dev/null || echo "")
    if [ -z "$GREP_PATH" ]; then
        exit 0
    fi
    if is_protected_env_file "$GREP_PATH"; then
        block_with_message "Grep on '$GREP_PATH' was blocked because .env files contain secrets."
    fi
    exit 0
fi

if [ "$TOOL_NAME" != "Bash" ]; then
    exit 0
fi

COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""' 2>/dev/null || echo "")
if [ -z "$COMMAND" ]; then
    exit 0
fi

check_env_access_bash "$COMMAND"
exit 0
