#!/bin/bash
set -e

# Archive completed/scrapped beans and commit the deletions
# Usage: .claude/scripts/beans-archive.sh

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Archiving completed/scrapped beans...${NC}"

echo -e "\n${GREEN}[1/3]${NC} Running beans archive..."
beans archive --force

cd "$PROJECT_DIR"
DELETED_FILES=$(git diff --name-only --diff-filter=D -- .beans/ 2>/dev/null || true)

if [ -z "$DELETED_FILES" ]; then
    echo -e "\n${GREEN}No beans were archived (nothing to commit).${NC}"
    exit 0
fi

echo -e "\n${GREEN}[2/3]${NC} Committing archived beans..."
echo "$DELETED_FILES" | while IFS= read -r f; do
    echo "  - $f"
done

git add .beans/
git commit --no-gpg-sign -m "chore: Archive completed and scrapped beans"

echo -e "\n${GREEN}[3/3]${NC} Pushing to remote..."
CURRENT_BRANCH=$(git branch --show-current)
git push origin "$CURRENT_BRANCH"

echo -e "\n${GREEN}Beans archived, committed, and pushed!${NC}"
