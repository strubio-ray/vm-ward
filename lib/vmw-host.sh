#!/usr/bin/env bash
# =============================================================================
# vmw-host.sh — macOS host daemon for vm-ward
# =============================================================================

VMW_STATE_DIR="${VMW_STATE_DIR:-$HOME/.local/state/vm-ward}"
VMW_CONFIG_DIR="${VMW_CONFIG_DIR:-$HOME/.config/vm-ward}"
LEASES_FILE="${VMW_STATE_DIR}/leases.json"
LOCK_DIR="${VMW_STATE_DIR}/sweep.lock"
TMUX_CACHE="${VMW_STATE_DIR}/tmux-cache.txt"
MACHINE_INDEX="${VAGRANT_HOME:-$HOME/.vagrant.d}/data/machine-index/index"

# Default configuration values
DEFAULT_DURATION="4h"
T1_RATIO="0.5"
T2_RATIO="0.875"

# ─────────────────────────────────────────────────────────────────────────────
# State helpers
# ─────────────────────────────────────────────────────────────────────────────

init_state() {
  ensure_dirs
  if [ ! -f "$LEASES_FILE" ]; then
    echo '{}' > "$LEASES_FILE"
  fi
}

get_default_duration() {
  config_get '.default_duration' "$DEFAULT_DURATION"
}

get_t1_ratio() {
  config_get '.warnings.t1_ratio' "$T1_RATIO"
}

get_t2_ratio() {
  config_get '.warnings.t2_ratio' "$T2_RATIO"
}

# ─────────────────────────────────────────────────────────────────────────────
# Vagrant machine index helpers
# ─────────────────────────────────────────────────────────────────────────────

# Read machine index and output machine entries as JSON
read_machine_index() {
  if [ ! -f "$MACHINE_INDEX" ]; then
    echo '{}'
    return
  fi
  local machines
  machines=$(jq -r '.machines // {}' "$MACHINE_INDEX" 2>/dev/null) || {
    log "Warning: corrupt machine index at $MACHINE_INDEX"
    echo '{}'
    return
  }
  echo "$machines"
}

# Get list of running VMs from VBoxManage
get_running_vms() {
  if ! command -v VBoxManage >/dev/null 2>&1; then
    return
  fi
  VBoxManage list runningvms 2>/dev/null | sed -n 's/^".*"{\(.*\)}/\1/p'
}

# Resolve current directory to a machine ID
resolve_vm_by_path() {
  local target_path="$1"
  local machines
  machines=$(read_machine_index)
  echo "$machines" | jq -r --arg path "$target_path" '
    to_entries[] | select(.value.vagrantfile_path == $path) | .key
  ' 2>/dev/null | head -1
}

# Resolve a VM name or "." to a machine ID
resolve_vm() {
  local name="$1"
  if [ "$name" = "." ]; then
    local machine_id
    machine_id=$(resolve_vm_by_path "$(pwd)")
    if [ -z "$machine_id" ]; then
      die "No VM found for current directory: $(pwd)"
    fi
    echo "$machine_id"
    return
  fi

  # Search by vm_name in machine index
  local machines
  machines=$(read_machine_index)
  local machine_id
  machine_id=$(echo "$machines" | jq -r --arg name "$name" '
    to_entries[] | select(.value.extra_data.box.name // .value.name // "" | test($name; "i")) | .key
  ' 2>/dev/null | head -1)

  if [ -z "$machine_id" ]; then
    die "No VM found matching: $name"
  fi
  echo "$machine_id"
}

# ─────────────────────────────────────────────────────────────────────────────
# Lease helpers
# ─────────────────────────────────────────────────────────────────────────────

lease_get() {
  local machine_id="$1"
  local field="$2"
  jq -r --arg id "$machine_id" --arg field "$field" '.[$id][$field] // empty' "$LEASES_FILE" 2>/dev/null
}

lease_exists() {
  local machine_id="$1"
  jq -e --arg id "$machine_id" 'has($id)' "$LEASES_FILE" >/dev/null 2>&1
}

lease_set() {
  local machine_id="$1"
  local field="$2"
  local value="$3"
  local tmp
  tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
  jq --arg id "$machine_id" --arg field "$field" --arg val "$value" \
    '.[$id][$field] = $val' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
}

lease_set_bool() {
  local machine_id="$1"
  local field="$2"
  local value="$3"
  local tmp
  tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
  jq --arg id "$machine_id" --arg field "$field" --argjson val "$value" \
    '.[$id][$field] = $val' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
}

lease_remove() {
  local machine_id="$1"
  local tmp
  tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
  jq --arg id "$machine_id" 'del(.[$id])' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
}

create_lease() {
  local machine_id="$1"
  local vagrantfile_path="$2"
  local vm_name="$3"
  local provider="$4"
  local duration="$5"
  local now
  now=$(epoch_now)

  local duration_secs
  duration_secs=$(parse_duration "$duration")

  local expires_at mode
  if [ "$duration_secs" = "indefinite" ]; then
    expires_at="null"
    mode="indefinite"
  else
    expires_at=$(( now + duration_secs ))
    mode="standard"
  fi

  local tmp
  tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
  jq --arg id "$machine_id" \
     --arg vfp "$vagrantfile_path" \
     --arg vn "$vm_name" \
     --arg prov "$provider" \
     --argjson cat "$now" \
     --argjson eat "$expires_at" \
     --arg dur "$duration" \
     --arg m "$mode" \
     '.[$id] = {
       vagrantfile_path: $vfp,
       vm_name: $vn,
       provider: $prov,
       created_at: $cat,
       expires_at: $eat,
       duration: $dur,
       mode: $m,
       warned_t1: false,
       warned_t2: false
     }' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
}

# ─────────────────────────────────────────────────────────────────────────────
# Commands
# ─────────────────────────────────────────────────────────────────────────────

cmd_status() {
  init_state
  local json_output=false
  if [ "${1:-}" = "--json" ]; then
    json_output=true
  fi

  local machines
  machines=$(read_machine_index)

  local running_vms
  running_vms=$(get_running_vms)

  local now
  now=$(epoch_now)

  if [ "$json_output" = true ]; then
    # JSON output mode
    local result="[]"
    for machine_id in $(echo "$machines" | jq -r 'keys[]' 2>/dev/null); do
      local vfp vm_name provider state
      vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
      vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')
      provider=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].provider // "unknown"')

      if echo "$running_vms" | grep -qF "$machine_id" 2>/dev/null; then
        state="running"
      else
        state="poweroff"
      fi

      local lease="none" remaining="n/a"
      if lease_exists "$machine_id"; then
        local mode expires_at
        mode=$(lease_get "$machine_id" "mode")
        expires_at=$(lease_get "$machine_id" "expires_at")
        if [ "$mode" = "exempt" ] || [ "$mode" = "indefinite" ]; then
          lease="$mode"
          remaining="$mode"
        elif [ -n "$expires_at" ] && [ "$expires_at" != "null" ]; then
          local secs_left=$(( expires_at - now ))
          lease="active"
          remaining=$(format_remaining "$secs_left")
        fi
      fi

      result=$(echo "$result" | jq \
        --arg id "$machine_id" \
        --arg name "$vm_name" \
        --arg path "$vfp" \
        --arg state "$state" \
        --arg lease "$lease" \
        --arg remaining "$remaining" \
        '. + [{id: $id, name: $name, path: $path, state: $state, lease: $lease, remaining: $remaining}]')
    done
    echo "$result" | jq .
    return
  fi

  # Table output
  local has_vms=false
  printf "\033[1m%-20s %-30s %-10s %-12s %-12s\033[0m\n" \
    "VM NAME" "PROJECT" "STATE" "LEASE" "TIME LEFT"
  printf "%s\n" "$(printf '─%.0s' {1..84})"

  for machine_id in $(echo "$machines" | jq -r 'keys[]' 2>/dev/null); do
    has_vms=true
    local vfp vm_name state
    vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
    vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')

    local project
    project=$(basename "$vfp" 2>/dev/null || echo "unknown")

    if echo "$running_vms" | grep -qF "$machine_id" 2>/dev/null; then
      state="running"
    else
      state="poweroff"
    fi

    local lease="none" remaining="n/a" color="\033[0m"
    if lease_exists "$machine_id"; then
      local mode expires_at
      mode=$(lease_get "$machine_id" "mode")
      expires_at=$(lease_get "$machine_id" "expires_at")
      if [ "$mode" = "exempt" ] || [ "$mode" = "indefinite" ]; then
        lease="$mode"
        remaining="$mode"
        color="\033[36m"  # cyan
      elif [ -n "$expires_at" ] && [ "$expires_at" != "null" ]; then
        local secs_left=$(( expires_at - now ))
        local duration_secs
        local created_at
        created_at=$(lease_get "$machine_id" "created_at")
        duration_secs=$(( expires_at - created_at ))

        if [ "$secs_left" -le 0 ]; then
          lease="expired"
          remaining="expired"
          color="\033[31m"  # red
        else
          lease="active"
          remaining=$(format_remaining "$secs_left")
          local elapsed=$(( now - created_at ))
          local ratio
          if [ "$duration_secs" -gt 0 ]; then
            ratio=$(echo "scale=3; $elapsed / $duration_secs" | bc 2>/dev/null || echo "0")
          else
            ratio="0"
          fi
          local t2
          t2=$(get_t2_ratio)
          local t1
          t1=$(get_t1_ratio)
          if echo "$ratio >= $t2" | bc -l 2>/dev/null | grep -q 1; then
            color="\033[31m"  # red
          elif echo "$ratio >= $t1" | bc -l 2>/dev/null | grep -q 1; then
            color="\033[33m"  # yellow
          else
            color="\033[32m"  # green
          fi
        fi
      fi
    fi

    if [ "$state" = "poweroff" ]; then
      color="\033[90m"  # gray
    fi

    printf "${color}%-20s %-30s %-10s %-12s %-12s\033[0m\n" \
      "$vm_name" "$project" "$state" "$lease" "$remaining"
  done

  if [ "$has_vms" = false ]; then
    echo "No VMs found in Vagrant machine index."
  fi
}

cmd_sweep() {
  init_state

  # Acquire lock
  if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    # Check for stale lock
    local pid_file="${LOCK_DIR}/pid"
    if [ -f "$pid_file" ]; then
      local old_pid
      old_pid=$(cat "$pid_file")
      if ! kill -0 "$old_pid" 2>/dev/null; then
        log "Removing stale lock (PID $old_pid no longer running)"
        rm -rf "$LOCK_DIR"
        mkdir "$LOCK_DIR" 2>/dev/null || die "Could not acquire sweep lock"
      else
        log "Sweep already running (PID $old_pid), skipping"
        return 0
      fi
    else
      rm -rf "$LOCK_DIR"
      mkdir "$LOCK_DIR" 2>/dev/null || die "Could not acquire sweep lock"
    fi
  fi

  # Write PID and set trap for cleanup
  echo $$ > "${LOCK_DIR}/pid"
  trap 'rm -rf "$LOCK_DIR"' EXIT

  local machines
  machines=$(read_machine_index)
  if [ "$machines" = "{}" ]; then
    update_tmux_cache 0 0
    return 0
  fi

  local running_vms
  running_vms=$(get_running_vms)

  local now
  now=$(epoch_now)
  local default_dur
  default_dur=$(get_default_duration)
  local t1_ratio t2_ratio
  t1_ratio=$(get_t1_ratio)
  t2_ratio=$(get_t2_ratio)

  local running_count=0
  local warning_count=0

  for machine_id in $(echo "$machines" | jq -r 'keys[]' 2>/dev/null); do
    # Skip VMs not actually running
    if ! echo "$running_vms" | grep -qF "$machine_id" 2>/dev/null; then
      continue
    fi

    running_count=$(( running_count + 1 ))

    local vfp vm_name provider
    vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
    vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')
    provider=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].provider // "virtualbox"')

    # Create retroactive lease if none exists
    if ! lease_exists "$machine_id"; then
      log "Creating retroactive lease for $vm_name ($machine_id) with duration $default_dur"
      create_lease "$machine_id" "$vfp" "$vm_name" "$provider" "$default_dur"
    fi

    local mode
    mode=$(lease_get "$machine_id" "mode")

    # Skip exempt and indefinite
    if [ "$mode" = "exempt" ] || [ "$mode" = "indefinite" ]; then
      continue
    fi

    local expires_at
    expires_at=$(lease_get "$machine_id" "expires_at")
    if [ -z "$expires_at" ] || [ "$expires_at" = "null" ]; then
      continue
    fi

    local secs_left=$(( expires_at - now ))

    # Expired — halt the VM
    if [ "$secs_left" -le 0 ]; then
      log "Halting $vm_name (lease expired)"
      VAGRANT_CWD="$vfp" vagrant halt 2>/dev/null || {
        log "Warning: vagrant halt failed for $vm_name"
      }
      lease_remove "$machine_id"
      continue
    fi

    # Check warning thresholds
    local created_at duration_secs elapsed ratio
    created_at=$(lease_get "$machine_id" "created_at")
    duration_secs=$(( expires_at - created_at ))
    elapsed=$(( now - created_at ))

    if [ "$duration_secs" -gt 0 ]; then
      ratio=$(echo "scale=4; $elapsed / $duration_secs" | bc 2>/dev/null || echo "0")
    else
      ratio="0"
    fi

    local remaining
    remaining=$(format_remaining "$secs_left")

    # T2 warning check
    local warned_t2
    warned_t2=$(lease_get "$machine_id" "warned_t2")
    if [ "$warned_t2" != "true" ] && echo "$ratio >= $t2_ratio" | bc -l 2>/dev/null | grep -q 1; then
      log "T2 warning for $vm_name ($remaining remaining)"
      warning_count=$(( warning_count + 1 ))
      timeout 10 env VAGRANT_CWD="$vfp" vagrant ssh -c "vmw warn --urgent '$remaining remaining'" 2>/dev/null || {
        log "Warning: could not deliver T2 warning to $vm_name"
      }
      lease_set_bool "$machine_id" "warned_t2" "true"
    # T1 warning check
    else
      local warned_t1
      warned_t1=$(lease_get "$machine_id" "warned_t1")
      if [ "$warned_t1" != "true" ] && echo "$ratio >= $t1_ratio" | bc -l 2>/dev/null | grep -q 1; then
        log "T1 warning for $vm_name ($remaining remaining)"
        warning_count=$(( warning_count + 1 ))
        timeout 10 env VAGRANT_CWD="$vfp" vagrant ssh -c "vmw warn '$remaining remaining'" 2>/dev/null || {
          log "Warning: could not deliver T1 warning to $vm_name"
        }
        lease_set_bool "$machine_id" "warned_t1" "true"
      fi
    fi
  done

  # Clean up leases for VMs no longer in machine index
  local lease_ids
  lease_ids=$(jq -r 'keys[]' "$LEASES_FILE" 2>/dev/null)
  local machine_ids
  machine_ids=$(echo "$machines" | jq -r 'keys[]' 2>/dev/null)
  for lease_id in $lease_ids; do
    if ! echo "$machine_ids" | grep -qF "$lease_id"; then
      log "Removing stale lease for $lease_id (no longer in machine index)"
      lease_remove "$lease_id"
    fi
  done

  update_tmux_cache "$running_count" "$warning_count"
}

cmd_extend() {
  init_state
  local name="${1:-.}"
  local duration="${2:-$(get_default_duration)}"

  local machine_id
  machine_id=$(resolve_vm "$name")

  local now
  now=$(epoch_now)
  local duration_secs
  duration_secs=$(parse_duration "$duration")

  if [ "$duration_secs" = "indefinite" ]; then
    local tmp
    tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
    jq --arg id "$machine_id" \
       '.[$id].expires_at = null |
        .[$id].duration = "indefinite" |
        .[$id].mode = "indefinite" |
        .[$id].warned_t1 = false |
        .[$id].warned_t2 = false' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
  else
    local new_expires=$(( now + duration_secs ))
    local tmp
    tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
    jq --arg id "$machine_id" \
       --argjson eat "$new_expires" \
       --arg dur "$duration" \
       --argjson now "$now" \
       '.[$id].expires_at = $eat |
        .[$id].created_at = $now |
        .[$id].duration = $dur |
        .[$id].mode = "standard" |
        .[$id].warned_t1 = false |
        .[$id].warned_t2 = false' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
  fi

  local vm_name
  vm_name=$(lease_get "$machine_id" "vm_name")
  echo "Extended $vm_name lease: $duration from now"
}

cmd_halt() {
  init_state
  local name="${1:-.}"
  local machine_id
  machine_id=$(resolve_vm "$name")

  local vfp
  vfp=$(lease_get "$machine_id" "vagrantfile_path")
  local vm_name
  vm_name=$(lease_get "$machine_id" "vm_name")

  if [ -z "$vfp" ]; then
    # Try machine index if no lease
    local machines
    machines=$(read_machine_index)
    vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
    vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')
  fi

  echo "Halting $vm_name..."
  VAGRANT_CWD="$vfp" vagrant halt 2>&1 || die "Failed to halt $vm_name"
  lease_remove "$machine_id"
  echo "Halted $vm_name and removed lease."
}

cmd_exempt() {
  init_state
  local name="${1:-.}"
  local machine_id
  machine_id=$(resolve_vm "$name")

  local tmp
  tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
  jq --arg id "$machine_id" \
     '.[$id].mode = "exempt" |
      .[$id].expires_at = null' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"

  local vm_name
  vm_name=$(lease_get "$machine_id" "vm_name")
  echo "Exempted $vm_name from auto-halt."
}

cmd_install() {
  local plist_src="${VMW_ROOT}/share/vm-ward/com.strubio.vm-ward.plist"
  local plist_dst="$HOME/Library/LaunchAgents/com.strubio.vm-ward.plist"

  if [ ! -f "$plist_src" ]; then
    die "Plist template not found at $plist_src"
  fi

  # Resolve absolute path to vmw binary
  local vmw_path
  vmw_path="$(cd -P "$(dirname "$0")" && pwd)/$(basename "$0")"
  if [ -L "$0" ]; then
    vmw_path="$(readlink -f "$0" 2>/dev/null || realpath "$0" 2>/dev/null || echo "$0")"
  fi

  ensure_dirs
  mkdir -p "$HOME/Library/LaunchAgents"

  # Substitute placeholders
  sed -e "s|{{VMW_PATH}}|${vmw_path}|g" \
      -e "s|{{HOME}}|${HOME}|g" \
      "$plist_src" > "$plist_dst"

  # Load with modern launchctl API
  local uid
  uid=$(id -u)
  launchctl bootstrap "gui/${uid}" "$plist_dst" 2>/dev/null || {
    # Try legacy API as fallback
    launchctl load "$plist_dst" 2>/dev/null || die "Failed to load launchd plist"
  }

  echo "vm-ward daemon installed and started."
  echo "  Plist: $plist_dst"
  echo "  Logs:  ${VMW_STATE_DIR}/sweep.log"
}

cmd_uninstall() {
  local plist_dst="$HOME/Library/LaunchAgents/com.strubio.vm-ward.plist"

  local uid
  uid=$(id -u)
  launchctl bootout "gui/${uid}/com.strubio.vm-ward" 2>/dev/null || {
    launchctl unload "$plist_dst" 2>/dev/null || true
  }

  rm -f "$plist_dst"
  echo "vm-ward daemon uninstalled."
}

cmd_tmux_status() {
  if [ -f "$TMUX_CACHE" ]; then
    cat "$TMUX_CACHE"
  fi
}

update_tmux_cache() {
  local running="$1"
  local warnings="$2"
  if [ "$running" -eq 0 ]; then
    echo -n "" > "$TMUX_CACHE"
  else
    echo -n "VM: ${running}↑ ${warnings}⚠" > "$TMUX_CACHE"
  fi
}

cmd_help() {
  cat <<'EOF'
vm-ward — Auto-halt daemon for forgotten Vagrant VMs

Usage: vmw <command> [options]

Commands:
  status [--json]            Show all VMs and lease status
  extend <name|.> [duration] Extend lease (default: 4h from now)
  halt <name|.>              Immediately halt a VM and remove lease
  exempt <name|.>            Exempt a VM from auto-halt
  sweep                      Run enforcement loop (called by launchd)
  install                    Install launchd daemon
  uninstall                  Remove launchd daemon
  tmux-status                Print tmux status bar segment
  version                    Print version
  help                       Show this help

Duration formats:
  4h          Hours
  30m         Minutes
  overnight   14 hours (configurable)
  weekend     48 hours
  indefinite  No expiry

Examples:
  vmw status              Show all VMs
  vmw extend . 8h         Extend current project's VM by 8 hours
  vmw extend . overnight  Extend until tomorrow morning
  vmw halt .              Halt current project's VM now
EOF
}

main() {
  local cmd="${1:-status}"
  shift || true

  case "$cmd" in
    status)      cmd_status "$@" ;;
    extend)      cmd_extend "$@" ;;
    halt)        cmd_halt "$@" ;;
    exempt)      cmd_exempt "$@" ;;
    sweep)       cmd_sweep "$@" ;;
    install)     cmd_install "$@" ;;
    uninstall)   cmd_uninstall "$@" ;;
    tmux-status) cmd_tmux_status "$@" ;;
    version)     vmw_version ;;
    help|--help|-h) cmd_help ;;
    *)           die "Unknown command: $cmd. Run 'vmw help' for usage." ;;
  esac
}
