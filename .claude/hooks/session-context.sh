#!/bin/bash
# Claude Code SessionStart hook: Injects current work context

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"
cd "$PROJECT_DIR" || exit 0

echo "## Current Work Context"

BRANCH=$(git branch --show-current 2>/dev/null)
if [ -z "$BRANCH" ]; then
    echo "- Branch: (detached HEAD)"
else
    echo "- Branch: $BRANCH"
fi

BEAN_ID=""
if [ -n "$BRANCH" ] && [ "$BRANCH" != "main" ] && [ "$BRANCH" != "master" ]; then
    if [[ "$BRANCH" =~ ^[a-z]+/(agentbox-[a-zA-Z0-9]+)-.* ]]; then
        BEAN_ID="${BASH_REMATCH[1]}"
    elif [[ "$BRANCH" =~ ^[a-z]+/(beans-[a-zA-Z0-9]+)-.* ]]; then
        BEAN_ID="${BASH_REMATCH[1]}"
    fi
fi

if [ -n "$BEAN_ID" ]; then
    BEAN_JSON=$(timeout 2s beans query "{ bean(id: \"${BEAN_ID}\") { title status body } }" --json 2>/dev/null || echo "")
    if [ -n "$BEAN_JSON" ]; then
        BEAN_TITLE=$(echo "$BEAN_JSON" | jq -r '.bean.title // empty' 2>/dev/null)
        BEAN_STATUS=$(echo "$BEAN_JSON" | jq -r '.bean.status // empty' 2>/dev/null)
        BEAN_BODY=$(echo "$BEAN_JSON" | jq -r '.bean.body // empty' 2>/dev/null)

        if [ -n "$BEAN_TITLE" ]; then
            echo "- Active bean: ${BEAN_ID} — \"${BEAN_TITLE}\" (${BEAN_STATUS})"
            TOTAL=$(echo "$BEAN_BODY" | grep -c '^\- \[[ x]\]' 2>/dev/null || echo "0")
            UNCHECKED=$(echo "$BEAN_BODY" | grep -c '^\- \[ \]' 2>/dev/null || echo "0")
            if [ "$TOTAL" -gt 0 ]; then
                echo "- Unchecked items: ${UNCHECKED} of ${TOTAL}"
            fi
        else
            echo "- Active bean: ${BEAN_ID} (could not fetch details)"
        fi
    else
        echo "- Active bean: ${BEAN_ID} (beans query timed out)"
    fi
else
    if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
        echo "- No active bean (on ${BRANCH} branch)"
    elif [ -z "$BRANCH" ]; then
        echo "- No active bean"
    else
        echo "- No active bean (branch does not follow naming convention)"
    fi
fi

COMMITS=$(git log --oneline -5 2>/dev/null || echo "")
if [ -n "$COMMITS" ]; then
    echo "- Recent commits:"
    echo "$COMMITS" | while IFS= read -r line; do
        echo "  - $line"
    done
fi
