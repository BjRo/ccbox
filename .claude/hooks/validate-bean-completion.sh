#!/bin/bash
# Claude Code TaskCompleted hook: Validates bean checklist completion

INPUT=$(cat)
TASK_SUBJECT=$(echo "$INPUT" | jq -r '.task_subject // ""')
TASK_DESCRIPTION=$(echo "$INPUT" | jq -r '.task_description // ""')

BEAN_ID=""
if [ -n "$TASK_SUBJECT" ]; then
    BEAN_ID=$(echo "$TASK_SUBJECT" | grep -oP 'agentbox-[a-zA-Z0-9]+' | head -1 || true)
fi
if [ -z "$BEAN_ID" ] && [ -n "$TASK_DESCRIPTION" ]; then
    BEAN_ID=$(echo "$TASK_DESCRIPTION" | grep -oP 'agentbox-[a-zA-Z0-9]+' | head -1 || true)
fi
if [ -z "$BEAN_ID" ]; then
    BRANCH=$(git branch --show-current 2>/dev/null || true)
    if [ -n "$BRANCH" ] && [[ "$BRANCH" =~ ^[a-z]+/(agentbox-[a-zA-Z0-9]+)-.* ]]; then
        BEAN_ID="${BASH_REMATCH[1]}"
    fi
fi

if [ -z "$BEAN_ID" ]; then
    exit 0
fi

BEAN_JSON=$(timeout 5s beans query "{ bean(id: \"$BEAN_ID\") { id title status body } }" --json 2>/dev/null || true)
if [ -z "$BEAN_JSON" ]; then
    exit 0
fi

BEAN_BODY=$(echo "$BEAN_JSON" | jq -r '.bean.body // empty' 2>/dev/null || true)
if [ -z "$BEAN_BODY" ]; then
    exit 0
fi

UNCHECKED=$(echo "$BEAN_BODY" | grep -P '^\- \[ \] ' || true)
if [ -z "$UNCHECKED" ]; then
    exit 0
fi

COUNT=$(echo "$UNCHECKED" | wc -l)
echo "" >&2
echo "========================================" >&2
echo "TASK COMPLETION BLOCKED" >&2
echo "========================================" >&2
echo "" >&2
echo "Bean $BEAN_ID has $COUNT unchecked checklist item(s):" >&2
echo "$UNCHECKED" >&2
echo "" >&2
echo "Complete all checklist items before finishing this task." >&2
exit 2
