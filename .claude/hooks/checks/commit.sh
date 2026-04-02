#!/bin/bash
# Sourced by pre-bash-dispatch.sh -- do not execute directly
# Provides: check_pre_commit(command, first_line)

check_pre_commit() {
    local command="$1"
    local first_line="$2"

    if ! echo "$first_line" | grep -qP '(^|&&\s*|;\s*)git\s+(-\S+\s+\S+\s+)*commit(\s|$)'; then
        return 0
    fi

    cd "${CLAUDE_PROJECT_DIR:-.}"

    # Only run lint when staged files include Go source changes
    if ! git diff --cached --name-only | grep -q '\.go$'; then
        return 0
    fi

    echo "Pre-commit hook: Running lint before commit..." >&2

    timeout 300 golangci-lint run ./... >&2
    local lint_exit=$?
    if [ "$lint_exit" -eq 124 ]; then
        echo "" >&2
        echo "========================================" >&2
        echo "COMMIT BLOCKED: Linter timed out after 5 minutes" >&2
        echo "========================================" >&2
        exit 2
    elif [ "$lint_exit" -ne 0 ]; then
        echo "" >&2
        echo "========================================" >&2
        echo "COMMIT BLOCKED: Linter failed" >&2
        echo "========================================" >&2
        echo "Please fix linting errors before committing." >&2
        echo "Run 'golangci-lint run ./...' to see issues." >&2
        exit 2
    fi

    echo "Pre-commit hook: Lint passed!" >&2
    return 0
}
