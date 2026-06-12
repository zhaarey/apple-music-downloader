#!/usr/bin/env bash
# Clear Apple Music artwork cache directories (admin context). Usage:
#   sync-repair-macos.sh /path/to/log.txt "/path/one" "/path/two"
set -euo pipefail

LOG_PATH="${1:?log path required}"
shift

echo "Aura sync repair started $(date -Iseconds)" >"$LOG_PATH"

for p in "$@"; do
  if [[ -z "$p" ]]; then
    continue
  fi
  if [[ ! -e "$p" ]]; then
    echo "Skip (missing): $p" >>"$LOG_PATH"
    continue
  fi
  if rm -rf "$p" 2>>"$LOG_PATH"; then
    echo "Cleared: $p" >>"$LOG_PATH"
  else
    echo "Error: $p" >>"$LOG_PATH"
  fi
done

echo "Aura sync repair finished $(date -Iseconds)" >>"$LOG_PATH"
