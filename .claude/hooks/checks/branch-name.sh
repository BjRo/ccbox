#!/bin/bash
# Sourced by pre-bash-dispatch.sh -- do not execute directly
# Provides: check_branch_name(command, first_line)

check_branch_name() {
    local command="$1"
    local first_line="$2"

    if ! echo "$first_line" | grep -qP '(^|&&\s*|;\s*)git\s+(checkout|switch)\s'; then
        return 0
    fi

    local branch_name=""

    if echo "$first_line" | grep -qP '(^|&&\s*|;\s*)git\s+checkout\s+(-\S+\s+)*-[bB]\s+'; then
        branch_name=$(echo "$first_line" | grep -oP '(^|&&\s*|;\s*)git\s+checkout\s+(-\S+\s+)*-[bB]\s+\K[a-zA-Z0-9/_.-]+' | tail -1)
    fi

    if [ -z "$branch_name" ] && echo "$first_line" | grep -qP '(^|&&\s*|;\s*)git\s+switch\s+(-\S+\s+)*-[cC]\s+'; then
        branch_name=$(echo "$first_line" | grep -oP '(^|&&\s*|;\s*)git\s+switch\s+(-\S+\s+)*-[cC]\s+\K[a-zA-Z0-9/_.-]+' | tail -1)
    fi

    if [ -z "$branch_name" ]; then
        return 0
    fi

    branch_name=$(echo "$branch_name" | sed "s/^[\"']//;s/[\"']$//")

    local valid_pattern='^(feat|fix|refactor|chore|docs)/(ccbox-[a-zA-Z0-9]+|beans-[a-zA-Z0-9]+)-.+$'

    if echo "$branch_name" | grep -qP "$valid_pattern"; then
        return 0
    fi

    echo "" >&2
    echo "========================================" >&2
    echo "BLOCKED: Invalid branch name" >&2
    echo "========================================" >&2
    echo "" >&2
    echo "Branch name '${branch_name}' does not match the required pattern." >&2
    echo "" >&2
    echo "Expected: <type>/<bean-id>-<description>" >&2
    echo "  Types: feat, fix, refactor, chore, docs" >&2
    echo "  Example: feat/ccbox-abc1-add-stack-detection" >&2
    echo "" >&2
    echo "Use the start-work script instead:" >&2
    echo "  .claude/scripts/start-work.sh <bean-id>" >&2
    exit 2
}
