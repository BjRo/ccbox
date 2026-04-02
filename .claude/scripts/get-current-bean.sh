#!/bin/bash
# Extracts bean ID from current branch name
# Branch format: <type>/<bean-id>-<description>

set -e

BRANCH=$(git branch --show-current)

if [[ "$BRANCH" == "main" ]] || [[ "$BRANCH" == "master" ]]; then
    echo "Error: On main/master branch, no bean ID to extract" >&2
    exit 1
fi

if [[ "$BRANCH" =~ ^[a-z]+/(ccbox-[a-zA-Z0-9]+)-.* ]]; then
    echo "${BASH_REMATCH[1]}"
elif [[ "$BRANCH" =~ ^[a-z]+/(beans-[a-zA-Z0-9]+)-.* ]]; then
    echo "${BASH_REMATCH[1]}"
else
    echo "Error: Could not extract bean ID from branch: $BRANCH" >&2
    echo "Expected format: <type>/<bean-id>-<description>" >&2
    exit 1
fi
