#!/bin/bash
# Claude Code PostToolUse hook: Validates Definition of Done on bean creation
#
# Note: PostToolUse hooks cannot hard-block. The JSON output is injected as
# context for Claude, which will see the message and act on it (soft enforcement).
# Output: Exit 0 with JSON {"decision":"block","reason":"..."} if DoD missing
#         Exit 0 silently if DoD present or command is not beans create

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')

if [[ ! "$COMMAND" =~ ^beans[[:space:]]+create[[:space:]] ]]; then
    exit 0
fi

TOOL_RESPONSE=$(echo "$INPUT" | jq -r '.tool_response // ""')
BEAN_ID=$(echo "$TOOL_RESPONSE" | grep -oP 'agentbox-[a-zA-Z0-9]+' | head -1)

if [ -z "$BEAN_ID" ]; then
    exit 0
fi

BEAN_JSON=$(beans query "{ bean(id: \"$BEAN_ID\") { type body } }" --json 2>/dev/null)
BEAN_TYPE=$(echo "$BEAN_JSON" | jq -r '.bean.type // ""')
BEAN_BODY=$(echo "$BEAN_JSON" | jq -r '.bean.body // ""')

if [ "$BEAN_TYPE" = "epic" ] || [ "$BEAN_TYPE" = "milestone" ]; then
    exit 0
fi

TEMPLATE_DIR="${CLAUDE_PROJECT_DIR:-.}/.claude/templates"
TEMPLATE_FILE="$TEMPLATE_DIR/definition-of-done.md"

if [ ! -f "$TEMPLATE_FILE" ]; then
    exit 0
fi

MISSING_ITEMS=()
while IFS= read -r line; do
    phrase=$(echo "$line" | sed 's/^- \[ \] //')
    if [ -n "$phrase" ] && [ "$phrase" != "$line" ]; then
        if ! echo "$BEAN_BODY" | grep -qi "$phrase"; then
            MISSING_ITEMS+=("$phrase")
        fi
    fi
done < "$TEMPLATE_FILE"

if [ ${#MISSING_ITEMS[@]} -gt 0 ]; then
    REASON="Bean $BEAN_ID is missing the Definition of Done checklist. Read .claude/templates/definition-of-done.md and append it to the bean body using beans update."
    echo "{\"decision\":\"block\",\"reason\":\"$REASON\"}"
    exit 0
fi

exit 0
