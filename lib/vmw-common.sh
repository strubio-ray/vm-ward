#!/usr/bin/env bash
# =============================================================================
# vmw-common.sh — Shared utilities for vm-ward
# =============================================================================

die() {
  echo "vmw: error: $*" >&2
  exit 1
}

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

vmw_version() {
  echo "vm-ward ${VMW_VERSION}"
}

epoch_now() {
  date +%s
}

json_get() {
  local file="$1"
  local query="$2"
  jq -r "$query" "$file" 2>/dev/null
}

# Atomic JSON write: read from source, transform with jq, write safely
# Usage: json_write <file> <jq_filter> [input_file]
# If input_file is omitted, reads from <file> itself
json_write() {
  local target="$1"
  local filter="$2"
  local input="${3:-$target}"
  local tmp
  tmp=$(mktemp "${target}.XXXXXX")
  if jq "$filter" "$input" > "$tmp"; then
    mv "$tmp" "$target"
  else
    rm -f "$tmp"
    return 1
  fi
}

# Convert duration string to seconds
# Supports: Nh, Nm, overnight, weekend, indefinite
parse_duration() {
  local dur="$1"
  case "$dur" in
    *h)
      local num="${dur%h}"
      echo $(( num * 3600 ))
      ;;
    *m)
      local num="${dur%m}"
      echo $(( num * 60 ))
      ;;
    overnight)
      echo $(( 14 * 3600 ))
      ;;
    weekend)
      echo $(( 48 * 3600 ))
      ;;
    indefinite)
      echo "indefinite"
      ;;
    *)
      die "Unknown duration format: $dur (expected Nh, Nm, overnight, weekend, indefinite)"
      ;;
  esac
}

# Convert seconds to human-readable "Xh Ym"
format_remaining() {
  local secs="$1"
  if [ "$secs" -le 0 ]; then
    echo "expired"
    return
  fi
  local hours=$(( secs / 3600 ))
  local mins=$(( (secs % 3600) / 60 ))
  if [ "$hours" -gt 0 ]; then
    echo "${hours}h ${mins}m"
  else
    echo "${mins}m"
  fi
}

# Ensure state and config directories exist
ensure_dirs() {
  mkdir -p "${VMW_STATE_DIR:-$HOME/.local/state/vm-ward}"
  mkdir -p "${VMW_CONFIG_DIR:-$HOME/.config/vm-ward}"
}

# Load config value with fallback to default
config_get() {
  local key="$1"
  local default="$2"
  local config_file="${VMW_CONFIG_DIR:-$HOME/.config/vm-ward}/config.json"
  if [ -f "$config_file" ]; then
    local val
    val=$(jq -r "$key // empty" "$config_file" 2>/dev/null)
    if [ -n "$val" ]; then
      echo "$val"
      return
    fi
  fi
  echo "$default"
}
