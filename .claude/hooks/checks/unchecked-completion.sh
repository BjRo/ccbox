#!/bin/bash
# Sourced by pre-bash-dispatch.sh -- do not execute directly
# Provides: check_unchecked_completion(command)

check_unchecked_completion() {
    local command="$1"

    local bean_id
    bean_id=$(echo "$command" | grep -oP 'beans\s+update\s+\K(ccbox-[a-zA-Z0-9]+)')

    if [ -z "$bean_id" ]; then
        return 0
    fi

    local bean_json
    bean_json=$(timeout 5s beans query "{ bean(id: \"$bean_id\") { type body } }" --json 2>/dev/null || true)

    if [ -z "$bean_json" ]; then
        return 0
    fi

    local bean_type
    bean_type=$(echo "$bean_json" | jq -r '.bean.type // ""')
    local bean_body
    bean_body=$(echo "$bean_json" | jq -r '.bean.body // ""')

    if [ "$bean_type" = "epic" ] || [ "$bean_type" = "milestone" ]; then
        return 0
    fi

    local unchecked
    unchecked=$(echo "$bean_body" | grep -P '^\- \[ \] ' || true)

    if [ -z "$unchecked" ]; then
        return 0
    fi

    local count
    count=$(echo "$unchecked" | wc -l)

    local item_list
    item_list=$(echo "$unchecked" | head -10 | sed 's/^- \[ \] /  - /')

    echo "" >&2
    echo "========================================" >&2
    echo "BLOCKED: Unchecked checklist items" >&2
    echo "========================================" >&2
    echo "" >&2
    echo "Cannot mark $bean_id as completed -- $count unchecked checklist item(s) remain:" >&2
    echo "$item_list" >&2
    if [ "$count" -gt 10 ]; then
        echo "  ... and $((count - 10)) more" >&2
    fi
    echo "" >&2
    echo "Check off all items before completing." >&2
    exit 2
}
