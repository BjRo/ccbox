#!/bin/bash
set -e

# Post-merge cleanup: verify merge, delete branches, complete bean
# Usage: .claude/scripts/post-merge.sh <bean-id>

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
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

BEAN_ID="$1"

echo -e "${YELLOW}Running post-merge cleanup for ${BEAN_ID}...${NC}"

CURRENT_BRANCH=$(git branch --show-current)
echo -e "\n${GREEN}[1/7]${NC} Current branch: ${CURRENT_BRANCH}"

if [ "$CURRENT_BRANCH" = "main" ]; then
    echo -e "${RED}Error: Already on main branch.${NC}"
    exit 1
fi

echo -e "\n${GREEN}[2/7]${NC} Checking PR status..."
PR_INFO=$(gh pr view --json state,headRefName,mergedAt 2>/dev/null || echo '{"error": true}')

if echo "$PR_INFO" | grep -q '"error"'; then
    echo -e "${RED}Error: No PR found for branch '${CURRENT_BRANCH}'${NC}"
    exit 1
fi

PR_STATE=$(echo "$PR_INFO" | jq -r '.state')
if [ "$PR_STATE" != "MERGED" ]; then
    echo -e "${RED}Error: PR is not merged (state: ${PR_STATE})${NC}"
    exit 1
fi

echo -e "PR state: ${GREEN}MERGED${NC}"

echo -e "\n${GREEN}[3/7]${NC} Switching to main and pulling latest..."
git checkout main
git pull origin main

echo -e "\n${GREEN}[4/7]${NC} Deleting local branch '${CURRENT_BRANCH}'..."
if ! git branch -d "$CURRENT_BRANCH" 2>/dev/null; then
    echo -e "${YELLOW}Warning: Branch has unmerged changes. Force deleting...${NC}"
    git branch -D "$CURRENT_BRANCH"
fi

echo -e "\n${GREEN}[5/7]${NC} Deleting remote branch..."
git push origin --delete "$CURRENT_BRANCH" 2>/dev/null || echo "Remote branch already deleted or doesn't exist"

echo -e "\n${GREEN}[6/7]${NC} Marking bean as completed..."
beans update "$BEAN_ID" --status completed

echo -e "\n${GREEN}[7/7]${NC} Committing and pushing bean status change..."
cd "$PROJECT_DIR"
git add .beans/
git commit -m "chore: Mark ${BEAN_ID} as completed"
git push origin main

echo -e "\n${GREEN}Post-merge cleanup complete!${NC}"
echo ""
echo "Summary:"
echo "  - Branch '${CURRENT_BRANCH}' deleted (local and remote)"
echo "  - Bean '${BEAN_ID}' marked as completed"
echo "  - Changes pushed to main"
