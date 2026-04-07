#!/bin/bash
# Launch a headless Claude Code instance to deliver a bean in an isolated worktree.
#
# Each instance gets its own git worktree. The bean acts as the durable state
# machine — if the instance escalates or fails, the orchestrator can resolve
# the issue, update the bean, and relaunch with --resume.
#
# Usage:
#   .claude/scripts/launch-deliver.sh <bean-id> [--slot N] [--resume] [--base <branch>]
#
# Options:
#   --slot N          Agent slot (1-3). Optional for single delivery, required for parallel.
#   --resume          Resume in existing worktree instead of creating a new one.
#   --base <branch>   Base branch for worktree and PR target (default: main).
#
# Examples:
#   .claude/scripts/launch-deliver.sh agentbox-abc1
#   .claude/scripts/launch-deliver.sh agentbox-abc1 --slot 1
#   .claude/scripts/launch-deliver.sh agentbox-abc1 --slot 1 --resume

set -e

PROJECT_DIR="$(cd "${CLAUDE_PROJECT_DIR:-.}" && pwd)"
LOGS_DIR="${PROJECT_DIR}/.claude/logs"
WORKTREES_DIR="${PROJECT_DIR}/.claude/worktrees"
STATE_DIR="${PROJECT_DIR}/.claude/state/slots"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

usage() {
    echo "Usage: $0 <bean-id> [--slot N] [--resume] [--base <branch>]"
    echo ""
    echo "Arguments:"
    echo "  bean-id       The bean to deliver (e.g., agentbox-abc1)"
    echo ""
    echo "Options:"
    echo "  --slot N      Agent slot 1-3 (optional for single, required for parallel)"
    echo "  --resume      Resume in existing worktree"
    echo "  --base <branch>  Base branch (default: main)"
    echo ""
    echo "Examples:"
    echo "  $0 agentbox-abc1"
    echo "  $0 agentbox-abc1 --slot 1"
    echo "  $0 agentbox-abc1 --slot 1 --resume"
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

BEAN_ID="$1"
shift

SLOT=""
RESUME=false
BASE_BRANCH="main"

while [[ $# -gt 0 ]]; do
    case $1 in
        --slot) SLOT="$2"; shift 2 ;;
        --resume) RESUME=true; shift ;;
        --base) BASE_BRANCH="$2"; shift 2 ;;
        *) echo -e "${RED}Unknown option: $1${NC}"; usage ;;
    esac
done

# Validate slot if provided
if [ -n "$SLOT" ] && ! [[ "$SLOT" =~ ^[1-3]$ ]]; then
    echo -e "${RED}Error: Slot must be 1, 2, or 3 (got: ${SLOT})${NC}"
    exit 1
fi

# Verify bean exists
BEAN_CHECK=$(beans query "{ bean(id: \"${BEAN_ID}\") { id title } }" --json 2>&1)
BEAN_TITLE=$(echo "$BEAN_CHECK" | jq -r '.bean.title // empty')

if [ -z "$BEAN_TITLE" ]; then
    echo -e "${RED}Error: Bean '${BEAN_ID}' not found${NC}"
    exit 1
fi

echo -e "${YELLOW}Launching deliver pipeline for: ${BEAN_TITLE}${NC}"
echo -e "  Bean:   ${BEAN_ID}"
[ -n "$SLOT" ] && echo -e "  Slot:   ${SLOT}"
echo -e "  Resume: ${RESUME}"

# --- Slot management (only when --slot is provided) ---
if [ -n "$SLOT" ]; then
    mkdir -p "$STATE_DIR"

    SLOT_FILE="${STATE_DIR}/${SLOT}"
    if [ -f "$SLOT_FILE" ]; then
        OCCUPANT=$(cat "$SLOT_FILE")
        if [ "$OCCUPANT" = "$BEAN_ID" ] && [ "$RESUME" = true ]; then
            echo -e "${YELLOW}Slot ${SLOT} already held by ${BEAN_ID} (resuming)${NC}"
        else
            echo -e "${RED}Error: Slot ${SLOT} is already occupied by: ${OCCUPANT}${NC}"
            echo -e "Release it first or use a different slot."
            exit 1
        fi
    fi

    # Claim the slot
    echo "$BEAN_ID" > "$SLOT_FILE"

    # Cleanup slot on exit
    cleanup() {
        rm -f "$SLOT_FILE"
        echo -e "\n${GREEN}Slot ${SLOT} released.${NC}"
    }
    trap cleanup EXIT
fi

# --- Worktree setup ---
WORKTREE_NAME="deliver-${BEAN_ID}"
WORKTREE_PATH="${WORKTREES_DIR}/${WORKTREE_NAME}"

if [ "$RESUME" = true ]; then
    if [ ! -d "$WORKTREE_PATH" ]; then
        echo -e "${RED}Error: No worktree to resume at ${WORKTREE_PATH}${NC}"
        exit 1
    fi
    echo -e "\n${GREEN}Resuming in existing worktree: ${WORKTREE_PATH}${NC}"
else
    echo -e "\n${GREEN}Creating worktree...${NC}"
    mkdir -p "$WORKTREES_DIR"
    git worktree add -b "deliver-${BEAN_ID}" "$WORKTREE_PATH" "$BASE_BRANCH"
fi

# --- Log setup ---
mkdir -p "$LOGS_DIR"
LOG_FILE="${LOGS_DIR}/deliver-${BEAN_ID}.log"
echo -e "Log file: ${LOG_FILE}"

# --- Build the prompt ---
PROMPT="Deliver bean ${BEAN_ID}."

if [ "$BASE_BRANCH" != "main" ]; then
    PROMPT="${PROMPT} BASE_BRANCH=${BASE_BRANCH}. When creating the PR, use --base ${BASE_BRANCH} to target this branch instead of main."
fi

if [ "$RESUME" = true ]; then
    PROMPT="${PROMPT} RESUME MODE: Read the bean's Pipeline State section and continue from where the previous instance left off. Skip completed phases."
fi

# --- Launch Claude ---
echo -e "\n${GREEN}Launching Claude Code instance...${NC}"
echo -e "Started at: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo ""

cd "$WORKTREE_PATH"

echo "" | env \
    -u CLAUDECODE \
    -u CLAUDE_AGENT_SDK_VERSION \
    -u CLAUDE_CODE_ENTRYPOINT \
    -u CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING \
    claude -p \
        --agent deliver \
        "$PROMPT" \
    > "$LOG_FILE" 2>&1

EXIT_CODE=$?

echo -e "\nFinished at: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"

# --- Report result ---
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}Deliver completed successfully for ${BEAN_ID}${NC}"
    echo -e "Check log: ${LOG_FILE}"
else
    echo -e "${RED}Deliver exited with code ${EXIT_CODE} for ${BEAN_ID}${NC}"
    echo -e "Check log: ${LOG_FILE}"
    echo -e "\n--- Last 20 lines of log ---"
    tail -20 "$LOG_FILE"
fi

exit $EXIT_CODE
