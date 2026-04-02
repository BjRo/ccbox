#!/bin/bash
# Sourced by pre-bash-dispatch.sh -- do not execute directly
# Provides: check_pre_push(command, first_line)

check_pre_push() {
    local command="$1"
    local first_line="$2"

    if ! echo "$first_line" | grep -qP '(^|&&\s*|;\s*)git\s+(-\S+\s+\S+\s+)*push(\s|$)'; then
        return 0
    fi

    cd "${CLAUDE_PROJECT_DIR:-.}"

    _push_has_source_changes() {
        if git diff --name-only @{upstream}..HEAD 2>/dev/null | grep -q '\.go$'; then
            return 0
        fi
        if git rev-parse --verify @{upstream} 2>/dev/null >/dev/null; then
            return 1
        fi
        if git diff --name-only origin/main..HEAD 2>/dev/null | grep -q '\.go$'; then
            return 0
        fi
        if git rev-parse --verify origin/main 2>/dev/null >/dev/null; then
            return 1
        fi
        return 0
    }

    if ! _push_has_source_changes; then
        return 0
    fi

    echo "Pre-push hook: Running tests before push..." >&2

    timeout 300 go test ./... >&2
    local test_exit=$?
    if [ "$test_exit" -eq 124 ]; then
        echo "" >&2
        echo "========================================" >&2
        echo "PUSH BLOCKED: Tests timed out after 5 minutes" >&2
        echo "========================================" >&2
        exit 2
    elif [ "$test_exit" -ne 0 ]; then
        echo "" >&2
        echo "========================================" >&2
        echo "PUSH BLOCKED: Tests failed" >&2
        echo "========================================" >&2
        echo "Please fix failing tests before pushing." >&2
        echo "Run 'go test ./...' to see failures." >&2
        exit 2
    fi

    echo "Pre-push hook: Tests passed!" >&2
    return 0
}
