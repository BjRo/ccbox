#!/bin/bash
# Start work on a bean: auto-derive branch name, create branch, mark in-progress, commit
#
# Usage: .claude/scripts/start-work.sh <bean-id>

set -e

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

usage() {
    echo "Usage: $0 <bean-id>"
    echo ""
    echo "Arguments:"
    echo "  bean-id  The bean ID (e.g., ccbox-abc1)"
    echo ""
    echo "Example:"
    echo "  $0 ccbox-abc1"
    echo "  # Creates: feat/ccbox-abc1-add-stack-detection"
    exit 1
}

map_type_to_prefix() {
    case "$1" in
        feature)  echo "feat" ;;
        bug)      echo "fix" ;;
        task)     echo "chore" ;;
        milestone) echo "chore" ;;
        epic)     echo "chore" ;;
        *)        echo "chore" ;;
    esac
}

slugify() {
    echo "$1" \
        | tr '[:upper:]' '[:lower:]' \
        | sed 's/[_ ]/-/g' \
        | sed 's/[^a-z0-9-]//g' \
        | sed 's/-\+/-/g' \
        | sed 's/^-//;s/-$//'
}

if [ $# -lt 1 ]; then
    usage
fi

BEAN_ID="$1"

echo -e "${YELLOW}Starting work on ${BEAN_ID}...${NC}"

echo -e "\n${GREEN}[1/5]${NC} Ensuring main is up-to-date..."
git checkout main
git pull origin main

echo -e "\n${GREEN}[2/5]${NC} Querying bean metadata..."
BEAN_JSON=$(beans query "{ bean(id: \"${BEAN_ID}\") { id title status type } }" --json 2>&1)

BEAN_TITLE=$(echo "$BEAN_JSON" | jq -r '.bean.title // empty')
BEAN_TYPE=$(echo "$BEAN_JSON" | jq -r '.bean.type // empty')
BEAN_STATUS=$(echo "$BEAN_JSON" | jq -r '.bean.status // empty')

if [ -z "$BEAN_TITLE" ]; then
    echo -e "${RED}Error: Bean '${BEAN_ID}' not found${NC}"
    exit 1
fi

if [ "$BEAN_STATUS" = "completed" ]; then
    echo -e "${RED}Error: Bean '${BEAN_ID}' is already completed${NC}"
    exit 1
fi

echo -e "  Title:  ${BEAN_TITLE}"
echo -e "  Type:   ${BEAN_TYPE}"
echo -e "  Status: ${BEAN_STATUS}"

PREFIX=$(map_type_to_prefix "$BEAN_TYPE")
SLUG=$(slugify "$BEAN_TITLE")

FIXED_PART="${PREFIX}/${BEAN_ID}-"
MAX_SLUG_LEN=$((72 - ${#FIXED_PART}))
if [ ${#SLUG} -gt $MAX_SLUG_LEN ]; then
    SLUG="${SLUG:0:$MAX_SLUG_LEN}"
    SLUG="${SLUG%-}"
fi

BRANCH_NAME="${PREFIX}/${BEAN_ID}-${SLUG}"

echo -e "\n${GREEN}[3/5]${NC} Creating branch '${BRANCH_NAME}'..."
git checkout -b "$BRANCH_NAME"

echo -e "\n${GREEN}[4/5]${NC} Marking bean as in-progress..."
beans update "$BEAN_ID" --status in-progress

echo -e "\n${GREEN}[5/5]${NC} Committing bean status change..."
cd "$PROJECT_DIR"
git add .beans/
git commit --no-gpg-sign -m "chore: Start work on ${BEAN_ID}"

echo -e "\n${GREEN}Ready to work!${NC}"
echo -e "Branch: ${YELLOW}${BRANCH_NAME}${NC}"
echo -e "Bean:   ${YELLOW}${BEAN_ID}${NC} (in-progress)"
echo ""
echo "Next steps:"
echo "  1. Implement the feature using TDD"
echo "  2. Update bean checklist as you go"
echo "  3. Run: golangci-lint run ./... && go test ./..."
echo "  4. Push and create PR: git push -u origin ${BRANCH_NAME}"
