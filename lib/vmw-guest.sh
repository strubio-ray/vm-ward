#!/usr/bin/env bash
# =============================================================================
# vmw-guest.sh — Linux guest agent for vm-ward
# =============================================================================

cmd_warn() {
  local urgent=false
  local message=""
  local display_duration=10000

  while [ $# -gt 0 ]; do
    case "$1" in
      --urgent) urgent=true; shift ;;
      *)        message="$1"; shift ;;
    esac
  done

  if [ -z "$message" ]; then
    die "Usage: vmw warn [--urgent] <message>"
  fi

  if [ "$urgent" = true ]; then
    display_duration=30000
    message="⚠ URGENT: ${message}"
  fi

  local prefix="vm-ward"
  local full_message="${prefix}: ${message}"

  # Try tmux first
  local sessions
  sessions=$(tmux list-sessions -F '#{session_name}' 2>/dev/null) || true

  if [ -n "$sessions" ]; then
    while IFS= read -r session; do
      tmux display-message -t "$session" -d "$display_duration" "$full_message" 2>/dev/null || true
    done <<< "$sessions"
  else
    # Fallback: log to file
    local logfile="/tmp/vmw-warnings.log"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $full_message" >> "$logfile"
    log "No tmux sessions found, logged to $logfile"
  fi
}

cmd_guest_help() {
  cat <<'EOF'
vm-ward guest agent

Usage: vmw <command>

Commands:
  warn [--urgent] <message>  Display lease warning in tmux sessions
  version                    Print version
  help                       Show this help
EOF
}

main() {
  local cmd="${1:-help}"
  shift || true

  case "$cmd" in
    warn)    cmd_warn "$@" ;;
    version) vmw_version ;;
    help|--help|-h) cmd_guest_help ;;
    *)       die "Unknown command: $cmd (guest mode). Host commands are only available on macOS." ;;
  esac
}
