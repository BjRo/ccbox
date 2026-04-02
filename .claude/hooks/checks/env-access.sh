#!/bin/bash
# Sourced by pre-bash-dispatch.sh -- do not execute directly
# Provides: check_env_access_bash(command)
#   Return 0 to allow, exit 2 to block
#
# Also provides shared helpers used by block-env-access.sh (Grep matcher):
#   is_protected_env_file(filepath)
#   block_env_with_message(detail)

block_env_with_message() {
    local detail="$1"
    echo "" >&2
    echo "========================================" >&2
    echo "BLOCKED: .env file access denied" >&2
    echo "========================================" >&2
    echo "" >&2
    echo "$detail" >&2
    echo "" >&2
    echo "Allowed alternatives:" >&2
    echo "  - .env.example (template without real values)" >&2
    echo "  - .env.template (template without real values)" >&2
    echo "  - test -f .env (existence check only)" >&2
    echo "  - ls .env (listing only)" >&2
    exit 2
}

# Returns 0 if the basename is a protected .env file, 1 if it's safe
is_protected_env_file() {
    local filepath="$1"
    local basename
    basename=$(basename "$filepath")

    # Not a .env file at all
    if ! echo "$basename" | grep -qP '^\.env' 2>/dev/null; then
        return 1
    fi

    # Safe variants: .env.example, .env.template, .env.sample, .envrc
    if echo "$basename" | grep -qP '^\.env\.(example|template|sample)' 2>/dev/null; then
        return 1
    fi
    if [ "$basename" = ".envrc" ]; then
        return 1
    fi

    # Protected: .env, .env.local, .env.production, .env.development, etc.
    return 0
}

check_env_access_bash() {
    local command="$1"

    # Strip quoted strings to avoid false positives (e.g., echo "check .env file")
    local stripped
    stripped=$(echo "$command" | sed -E "s/'[^']*'/__QUOTED__/g; s/\"[^\"]*\"/__QUOTED__/g")

    # Check if the stripped command contains any reference to a protected .env file.
    local env_refs
    env_refs=$(echo "$stripped" | grep -oP '(\S*/)?\.env(\.\w+)?(?=\s|$|;|\||&|>|<)' 2>/dev/null || echo "")

    # Also check for .env at end of line (no trailing delimiter)
    local env_refs_eol
    env_refs_eol=$(echo "$stripped" | grep -oP '(\S*/)?\.env(\.\w+)?$' 2>/dev/null || echo "")

    local all_refs="$env_refs $env_refs_eol"
    all_refs=$(echo "$all_refs" | tr ' ' '\n' | sort -u | grep -v '^$' || echo "")

    if [ -z "$all_refs" ]; then
        return 0  # No .env references found
    fi

    # Check if any reference is to a protected .env file
    local has_protected=false
    local ref
    for ref in $all_refs; do
        if is_protected_env_file "$ref"; then
            has_protected=true
            break
        fi
    done

    if ! $has_protected; then
        return 0  # All references are to safe .env files
    fi

    # We have a protected .env file reference. Now check if the command is
    # actually accessing it (vs. just checking existence or listing).

    # Safe commands that don't expose file contents: test, [, ls, stat, file
    local safe_pattern='(^|\s|;|&&|\|\|)\s*(test|ls|\[|stat|file)\s'
    if echo "$stripped" | grep -qP "$safe_pattern" 2>/dev/null; then
        local dangerous_pattern='(^|\s|;|&&|\|\||\|)\s*(cat|head|tail|less|more|grep|egrep|fgrep|rg|sed|awk|nano|vi|vim|nvim|cp|mv|rm|source|diff|sort|uniq|tee|xargs|wc|chmod|chown)\s'
        if ! echo "$stripped" | grep -qP "$dangerous_pattern" 2>/dev/null; then
            if ! echo "$stripped" | grep -qP '(^|\s|;|&&|\|\|)\.\s+' 2>/dev/null; then
                if ! echo "$stripped" | grep -qP '>+\s*(\S*/)?\.env(\.\w+)?(\s|$)' 2>/dev/null; then
                    return 0  # Only safe commands, allow
                fi
            fi
        fi
    fi

    # Check for redirect to a protected .env file (> .env, >> .env)
    if echo "$stripped" | grep -qP '>+\s*(\S*/)?\.env(\.\w+)?(\s|$)' 2>/dev/null; then
        local redir_target
        redir_target=$(echo "$stripped" | grep -oP '>+\s*\K(\S*/)?\.env(\.\w+)?' 2>/dev/null | head -1)
        if is_protected_env_file "$redir_target"; then
            block_env_with_message "Writing to .env files is blocked because they contain secrets."
        fi
    fi

    # Check for dangerous commands operating on the .env file
    local dangerous_cmds='cat|head|tail|less|more|grep|egrep|fgrep|rg|sed|awk|nano|vi|vim|nvim|cp|mv|rm|source|diff|sort|uniq|tee|xargs|wc|chmod|chown'
    if echo "$stripped" | grep -qP "(^|\s|;|&&|\|\||\|)\s*($dangerous_cmds)\s" 2>/dev/null; then
        block_env_with_message "Access to .env files was blocked because they contain secrets (API keys, credentials, etc.) that should never be exposed in the conversation context."
    fi

    # Check for dot-sourcing: . .env or . /path/.env
    if echo "$stripped" | grep -qP '(^|\s|;|&&|\|\|)\.\s+(\S*/)?\.env(\.\w+)?(\s|$)' 2>/dev/null; then
        block_env_with_message "Sourcing .env files was blocked because they contain secrets."
    fi

    # If we reach here with a protected .env ref but no dangerous command detected, allow
    return 0
}
