#!/usr/bin/env bash
# Claude Code status line: model | dir | branch | ctx used/total (%)
# Docs: https://code.claude.com/docs/en/statusline

set -u

input=$(cat)

j() { printf '%s' "$input" | jq -r "$1" 2>/dev/null; }

MODEL=$(j '.model.display_name // "Claude"')
CWD=$(j '.workspace.current_dir // .cwd // "."')
DIR="${CWD##*/}"

IN_TOK=$(j '.context_window.total_input_tokens // 0')
OUT_TOK=$(j '.context_window.total_output_tokens // 0')
CTX_SIZE=$(j '.context_window.context_window_size // 200000')
PCT=$(j '.context_window.used_percentage // empty')

fmt() {
  awk -v n="$1" 'BEGIN{
    if (n >= 1000000) printf "%.1fM", n/1000000
    else if (n >= 1000) printf "%.0fk", n/1000
    else printf "%d", n
  }'
}

TOTAL_TOK=$((IN_TOK + OUT_TOK))
CTX="$(fmt "$TOTAL_TOK")/$(fmt "$CTX_SIZE")"
if [ -n "$PCT" ] && [ "$PCT" != "null" ]; then
  PCT_INT=$(printf '%.0f' "$PCT")
  CTX="$CTX (${PCT_INT}%)"
fi

BRANCH=""
if git -C "$CWD" rev-parse --git-dir >/dev/null 2>&1; then
  BRANCH=$(git -C "$CWD" branch --show-current 2>/dev/null)
fi

if [ -n "$BRANCH" ]; then
  printf '%s | %s | %s | ctx %s\n' "$MODEL" "$DIR" "$BRANCH" "$CTX"
else
  printf '%s | %s | ctx %s\n' "$MODEL" "$DIR" "$CTX"
fi
