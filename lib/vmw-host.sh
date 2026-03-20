#!/usr/bin/env bash
# =============================================================================
# vmw-host.sh — macOS host daemon for vm-ward
# =============================================================================

VMW_STATE_DIR="${VMW_STATE_DIR:-$HOME/.local/state/vm-ward}"
VMW_CONFIG_DIR="${VMW_CONFIG_DIR:-$HOME/.config/vm-ward}"
LEASES_FILE="${VMW_STATE_DIR}/leases.json"
LOCK_DIR="${VMW_STATE_DIR}/sweep.lock"
EVENTS_FILE="${VMW_STATE_DIR}/events.jsonl"
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

get_activity_enabled() {
  config_get '.activity_detection.enabled' "true"
}

get_activity_cpu_threshold() {
  config_get '.activity_detection.cpu_threshold' "5"
}

daemon_status() {
  local plist_dst="$HOME/Library/LaunchAgents/com.strubio.vm-ward.plist"
  if [ ! -f "$plist_dst" ]; then
    echo "not-installed"
    return
  fi
  local line
  line=$(launchctl list 2>/dev/null | grep 'com.strubio.vm-ward' || true)
  if [ -z "$line" ]; then
    echo "not-running"
    return
  fi
  local pid
  pid=$(echo "$line" | awk '{print $1}')
  if [ "$pid" != "-" ] && [ -n "$pid" ]; then
    echo "running $pid"
  else
    echo "loaded"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Vagrant machine index helpers
# ─────────────────────────────────────────────────────────────────────────────

# Read machine index and output machine entries as JSON
remove_from_machine_index() {
  local machine_id="$1"
  if [ ! -f "$MACHINE_INDEX" ]; then
    return
  fi
  # Only remove if the entry exists
  if ! jq -e --arg id "$machine_id" '.machines[$id]' "$MACHINE_INDEX" >/dev/null 2>&1; then
    return
  fi
  local tmp
  tmp=$(mktemp "${MACHINE_INDEX}.XXXXXX")
  jq --arg id "$machine_id" '.machines |= del(.[$id])' "$MACHINE_INDEX" > "$tmp" && mv "$tmp" "$MACHINE_INDEX"
}

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

# Get copier template version for a project path
get_template_version() {
  local vfp="$1"
  local answers="${vfp}/.vm/.copier-answers.yml"
  [ -f "$answers" ] && grep '^_commit:' "$answers" | awk '{print $2}'
}

# Get list of running VMs from VBoxManage
get_running_vms() {
  if ! command -v VBoxManage >/dev/null 2>&1; then
    return
  fi
  perl -e 'alarm(shift); exec @ARGV' 5 bash -c 'VBoxManage list runningvms 2>/dev/null' 2>/dev/null | sed -n 's/^"[^"]*" {\([^}]*\)}/\1/p'
}

# Resolve a Vagrant machine entry to its VirtualBox UUID
resolve_vbox_uuid() {
  local vfp="$1" name="$2"
  local id_file="$vfp/.vagrant/machines/$name/virtualbox/id"
  if [ -f "$id_file" ]; then
    cat "$id_file"
  fi
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

  # Try as a direct machine ID first
  local machines
  machines=$(read_machine_index)
  if echo "$machines" | jq -e --arg id "$name" 'has($id)' >/dev/null 2>&1; then
    echo "$name"
    return
  fi

  # Try as a vagrantfile path
  local machine_id
  machine_id=$(echo "$machines" | jq -r --arg path "$name" '
    to_entries[] | select(.value.vagrantfile_path == $path) | .key
  ' 2>/dev/null | head -1)
  if [ -n "$machine_id" ]; then
    echo "$machine_id"
    return
  fi

  # Search by vm_name in machine index
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

lease_set_halted() {
  local machine_id="$1"
  local now
  now=$(epoch_now)
  local tmp
  tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
  jq --arg id "$machine_id" --argjson now "$now" \
    '.[$id].mode = "halted" | .[$id].expires_at = null | .[$id].halted_at = $now' \
    "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
}

event_log() {
  local type="$1" vm_name="$2" machine_id="$3" detail="${4:-}"
  local now
  now=$(epoch_now)
  printf '%s\n' "$(jq -nc --argjson ts "$now" --arg type "$type" \
    --arg vm "$vm_name" --arg mid "$machine_id" --arg det "$detail" \
    '{ts: $ts, type: $type, vm_name: $vm, machine_id: $mid, detail: $det}')" >> "$EVENTS_FILE"
  # Trim to 500 lines
  if [ -f "$EVENTS_FILE" ]; then
    local lc; lc=$(wc -l < "$EVENTS_FILE")
    if [ "$lc" -gt 500 ]; then
      local tmp; tmp=$(mktemp "${EVENTS_FILE}.XXXXXX")
      tail -n 500 "$EVENTS_FILE" > "$tmp" && mv "$tmp" "$EVENTS_FILE"
    fi
  fi
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
       warned_t2: false,
       last_active: $cat
     }' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
}

# ─────────────────────────────────────────────────────────────────────────────
# Activity detection
# ─────────────────────────────────────────────────────────────────────────────

ensure_metrics_setup() {
  local vbox_uuid="$1"
  # Check if metrics are actively collecting by looking for percentage values in query output.
  # VBoxManage metrics list always shows CPU/Load/User even when collection is not enabled,
  # so we must check query output for actual data instead.
  local query
  query=$(VBoxManage metrics query "$vbox_uuid" CPU/Load/User 2>/dev/null) || true
  if echo "$query" | grep -qE '[0-9]+(\.[0-9]+)?%'; then
    return 0
  fi
  # No data — enable metrics collection
  VBoxManage metrics setup --period 10 --samples 6 "$vbox_uuid" CPU/Load >/dev/null 2>&1 || true
}

check_vm_activity() {
  local vbox_uuid="$1"
  local cpu_threshold="$2"

  local output
  output=$(VBoxManage metrics query "$vbox_uuid" CPU/Load/User 2>/dev/null) || {
    echo "idle -1"
    return
  }

  # Prefer the :avg row (present when --samples > 1), fall back to raw row
  local cpu_pct
  cpu_pct=$(echo "$output" | grep "CPU/Load/User:avg" | sed -E 's/.*[[:space:]]([0-9]+)(\.[0-9]+)?%.*/\1/')
  if [ -z "$cpu_pct" ]; then
    cpu_pct=$(echo "$output" | grep "CPU/Load/User" | head -1 | sed -E 's/.*[[:space:]]([0-9]+)(\.[0-9]+)?%.*/\1/')
  fi

  # No data yet (metrics just set up, need one sampling period to populate)
  if [ -z "$cpu_pct" ]; then
    echo "idle -1"
    return
  fi

  if [ "$cpu_pct" -ge "$cpu_threshold" ] 2>/dev/null; then
    echo "active $cpu_pct"
  else
    echo "idle $cpu_pct"
  fi
}

reset_lease_activity() {
  local machine_id="$1"
  local now
  now=$(epoch_now)

  local duration
  duration=$(lease_get "$machine_id" "duration")
  if [ -z "$duration" ]; then
    log "Warning: no duration found for $machine_id, skipping activity reset"
    return 1
  fi

  local duration_secs
  duration_secs=$(parse_duration "$duration")
  if [ "$duration_secs" = "indefinite" ]; then
    return 0  # indefinite leases don't need reset
  fi

  local new_expires=$(( now + duration_secs ))
  local tmp
  tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
  jq --arg id "$machine_id" \
     --argjson now "$now" \
     --argjson eat "$new_expires" \
     '.[$id].created_at = $now |
      .[$id].expires_at = $eat |
      .[$id].warned_t1 = false |
      .[$id].warned_t2 = false |
      .[$id].last_active = $now' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
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

  local known_uuids=""

  if [ "$json_output" = true ]; then
    # JSON output mode
    local daemon_st
    daemon_st=$(daemon_status)
    local daemon_json
    case "$daemon_st" in
      running\ *)
        local daemon_pid="${daemon_st#running }"
        daemon_json=$(jq -nc --arg state "running" --argjson pid "$daemon_pid" '{state: $state, pid: $pid}')
        ;;
      *)
        daemon_json=$(jq -nc --arg state "$daemon_st" '{state: $state, pid: null}')
        ;;
    esac

    local result="[]"
    for machine_id in $(echo "$machines" | jq -r 'keys[]' 2>/dev/null); do
      local vfp vm_name provider state machine_name vbox_uuid
      vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
      vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')
      provider=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].provider // "unknown"')
      machine_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].name // "default"')
      vbox_uuid=$(resolve_vbox_uuid "$vfp" "$machine_name")
      if [ -n "$vbox_uuid" ]; then
        known_uuids="$known_uuids $vbox_uuid"
      fi

      if [ -n "$vbox_uuid" ] && echo "$running_vms" | grep -qF "$vbox_uuid" 2>/dev/null; then
        state="running"
      else
        state="poweroff"
      fi

      local lease="none" remaining="n/a" halted_at_val="" expires_at_val=""
      if lease_exists "$machine_id"; then
        local mode expires_at
        mode=$(lease_get "$machine_id" "mode")
        expires_at=$(lease_get "$machine_id" "expires_at")
        expires_at_val="$expires_at"
        if [ "$mode" = "halted" ]; then
          lease="halted"
          remaining="n/a"
          halted_at_val=$(lease_get "$machine_id" "halted_at")
        elif [ "$mode" = "exempt" ] || [ "$mode" = "indefinite" ]; then
          lease="$mode"
          remaining="$mode"
        elif [ -n "$expires_at" ] && [ "$expires_at" != "null" ]; then
          local secs_left=$(( expires_at - now ))
          if [ "$secs_left" -le 0 ]; then
            lease="expired"
            remaining="expired"
          else
            lease="active"
            remaining=$(format_remaining "$secs_left")
          fi
        fi
      fi

      if [ "$lease" = "none" ] && [ "$state" = "running" ]; then
        local daemon_st; daemon_st=$(daemon_status)
        case "$daemon_st" in
          running\ *|loaded) lease="pending"; remaining="next sweep" ;;
        esac
      fi

      local last_active_ts="" duration_val="" last_activity_val="" cpu_percent_val=""
      if lease_exists "$machine_id"; then
        last_active_ts=$(lease_get "$machine_id" "last_active")
        duration_val=$(lease_get "$machine_id" "duration")
        last_activity_val=$(lease_get "$machine_id" "last_activity")
        cpu_percent_val=$(lease_get "$machine_id" "cpu_percent")
      fi

      local section="active"
      if [ "$lease" = "halted" ] || { [ "$state" = "poweroff" ] && { [ "$lease" = "expired" ] || [ "$lease" = "none" ]; }; }; then
        section="halted"
      fi

      local tpl_version
      tpl_version=$(get_template_version "$vfp")

      result=$(echo "$result" | jq \
        --arg id "$machine_id" \
        --arg name "$vm_name" \
        --arg path "$vfp" \
        --arg state "$state" \
        --arg lease "$lease" \
        --arg remaining "$remaining" \
        --arg last_active "$last_active_ts" \
        --arg halted_at "$halted_at_val" \
        --arg duration "$duration_val" \
        --arg section "$section" \
        --arg expires_at_val "$expires_at_val" \
        --arg last_activity "$last_activity_val" \
        --arg tpl_version "$tpl_version" \
        --arg cpu_pct "$cpu_percent_val" \
        '. + [{id: $id, name: $name, path: $path, state: $state, lease: $lease, remaining: $remaining, duration: (if $duration == "" then null else $duration end), last_active: (if $last_active == "" then null else ($last_active | tonumber) end), halted_at: (if $halted_at == "" then null else ($halted_at | tonumber) end), managed: true, section: $section, expires_at: (if $expires_at_val == "" or $expires_at_val == "null" then null else ($expires_at_val | tonumber) end), last_activity: (if $last_activity == "" then null else $last_activity end), template_version: (if $tpl_version == "" then null else $tpl_version end), cpu_percent: (if $cpu_pct == "" or $cpu_pct == "-1" then null else ($cpu_pct | tonumber) end)}]')
    done

    # Add unmanaged VBox VMs
    if command -v VBoxManage >/dev/null 2>&1; then
      local all_vbox_vms
      all_vbox_vms=$(VBoxManage list vms 2>/dev/null | sed -n 's/^"\([^"]*\)" {\([^}]*\)}/\1 \2/p')
      while IFS=' ' read -r vm_display_name vm_uuid; do
        [ -z "$vm_uuid" ] && continue
        if echo "$known_uuids" | grep -qF "$vm_uuid" 2>/dev/null; then
          continue
        fi
        local vm_state="poweroff"
        if echo "$running_vms" | grep -qF "$vm_uuid" 2>/dev/null; then
          vm_state="running"
        fi
        result=$(echo "$result" | jq \
          --arg name "$vm_display_name" \
          --arg id "$vm_uuid" \
          --arg state "$vm_state" \
          '. + [{id: $id, name: $name, path: "", state: $state, lease: "n/a", remaining: "n/a", managed: false}]')
      done <<< "$all_vbox_vms"
    fi

    local last_sweep_json="null"
    if [ -f "${VMW_STATE_DIR}/last-sweep" ]; then
      last_sweep_json=$(cat "${VMW_STATE_DIR}/last-sweep")
    fi

    local recent_events="[]"
    if [ -f "$EVENTS_FILE" ]; then
      recent_events=$(tail -n 5 "$EVENTS_FILE" | jq -sc '.')
    fi

    local cfg_cpu_threshold
    cfg_cpu_threshold=$(get_activity_cpu_threshold)
    local cfg_activity_enabled
    cfg_activity_enabled=$(get_activity_enabled)

    jq -nc --argjson daemon "$daemon_json" --argjson vms "$result" \
      --argjson last_sweep "$last_sweep_json" \
      --argjson events "$recent_events" \
      --argjson cpu_threshold "$cfg_cpu_threshold" \
      --argjson activity_enabled "$([ "$cfg_activity_enabled" = "true" ] && echo true || echo false)" \
      '{daemon: $daemon, last_sweep: $last_sweep, recent_events: $events, vms: $vms, cpu_threshold: $cpu_threshold, activity_enabled: $activity_enabled}'
    return
  fi

  # No --json flag: launch interactive TUI dashboard
  local tui_bin="${VMW_ROOT}/bin/vmw-tui"
  if [ ! -x "$tui_bin" ]; then
    die "TUI binary not found. Install via 'brew upgrade vm-ward' or build with 'cd tui && go build -o ../bin/vmw-tui'"
  fi
  VMW_PATH="${VMW_ROOT}/bin/vmw" exec "$tui_bin"
}

cmd_sweep() {
  init_state
  local skip_activity=false
  if [ "${1:-}" = "--no-activity" ]; then
    skip_activity=true
  fi

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
    local vfp vm_name provider machine_name vbox_uuid
    vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
    vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')
    provider=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].provider // "virtualbox"')
    machine_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].name // "default"')
    vbox_uuid=$(resolve_vbox_uuid "$vfp" "$machine_name")

    # Skip VMs not actually running
    if [ -z "$vbox_uuid" ] || ! echo "$running_vms" | grep -qF "$vbox_uuid" 2>/dev/null; then
      continue
    fi

    running_count=$(( running_count + 1 ))

    # Create retroactive lease if none exists
    if ! lease_exists "$machine_id"; then
      log "Creating retroactive lease for $vm_name ($machine_id) with duration $default_dur"
      create_lease "$machine_id" "$vfp" "$vm_name" "$provider" "$default_dur"
      event_log "lease_created" "$vm_name" "$machine_id" "retroactive, duration=$default_dur"
    fi

    local mode
    mode=$(lease_get "$machine_id" "mode")

    # Collect CPU metrics for all running VMs (including exempt/indefinite)
    if [ "$skip_activity" = false ] && [ "$(get_activity_enabled)" = "true" ]; then
      ensure_metrics_setup "$vbox_uuid"
      local activity_result
      activity_result=$(check_vm_activity "$vbox_uuid" "$(get_activity_cpu_threshold)")
      local activity="${activity_result%% *}"
      local cpu_pct="${activity_result##* }"
      local _tmp
      _tmp=$(mktemp "${LEASES_FILE}.XXXXXX")
      jq --arg id "$machine_id" --arg act "$activity" --arg cpu "$cpu_pct" \
        '.[$id].last_activity = $act | .[$id].cpu_percent = $cpu' \
        "$LEASES_FILE" > "$_tmp" && mv "$_tmp" "$LEASES_FILE"
    fi

    # Skip exempt and indefinite (no expiry/warning logic needed)
    if [ "$mode" = "exempt" ] || [ "$mode" = "indefinite" ]; then
      continue
    fi

    local expires_at
    expires_at=$(lease_get "$machine_id" "expires_at")
    if [ -z "$expires_at" ] || [ "$expires_at" = "null" ]; then
      continue
    fi

    # Activity-based lease reset
    if [ "$skip_activity" = false ] && [ "$(get_activity_enabled)" = "true" ]; then
      if [ "$activity" = "active" ]; then
        log "Activity detected on $vm_name, resetting lease"
        reset_lease_activity "$machine_id"
        event_log "lease_reset" "$vm_name" "$machine_id" "activity detected"
        expires_at=$(lease_get "$machine_id" "expires_at")
      fi
    fi

    local secs_left=$(( expires_at - now ))

    # Expired — halt the VM
    if [ "$secs_left" -le 0 ]; then
      log "Halting $vm_name (lease expired)"
      VAGRANT_CWD="$vfp" vagrant halt 2>/dev/null || {
        log "Warning: vagrant halt failed for $vm_name"
      }
      lease_set_halted "$machine_id"
      event_log "halted" "$vm_name" "$machine_id" "lease expired"
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
      lease_set_bool "$machine_id" "warned_t2" "true"
    fi

    # T1 warning check
    local warned_t1
    warned_t1=$(lease_get "$machine_id" "warned_t1")
    if [ "$warned_t1" != "true" ] && echo "$ratio >= $t1_ratio" | bc -l 2>/dev/null | grep -q 1; then
      log "T1 warning for $vm_name ($remaining remaining)"
      warning_count=$(( warning_count + 1 ))
      lease_set_bool "$machine_id" "warned_t1" "true"
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

  # Clean up halted leases older than 24h
  for lease_id in $(jq -r 'to_entries[] | select(.value.mode == "halted") | .key' "$LEASES_FILE" 2>/dev/null); do
    local halted_at
    halted_at=$(lease_get "$lease_id" "halted_at")
    if [ -n "$halted_at" ] && [ $(( now - halted_at )) -gt 86400 ]; then
      log "Removing stale halted lease for $lease_id (halted >24h ago)"
      lease_remove "$lease_id"
    fi
  done

  # Clean up expired standard leases for poweroff VMs
  for lease_id in $(jq -r --argjson now "$now" 'to_entries[] | select(.value.mode == "standard" and .value.expires_at != null and (.value.expires_at | tonumber) < $now) | .key' "$LEASES_FILE" 2>/dev/null); do
    # Check if VM is not running
    local lease_vfp lease_machine_name lease_vbox_uuid
    lease_vfp=$(lease_get "$lease_id" "vagrantfile_path")
    lease_machine_name=$(echo "$machines" | jq -r --arg id "$lease_id" '.[$id].name // "default"')
    lease_vbox_uuid=$(resolve_vbox_uuid "$lease_vfp" "$lease_machine_name")
    if [ -z "$lease_vbox_uuid" ] || ! echo "$running_vms" | grep -qF "$lease_vbox_uuid" 2>/dev/null; then
      log "Removing expired lease for $lease_id (poweroff VM with expired standard lease)"
      lease_remove "$lease_id"
    fi
  done

  update_tmux_cache "$running_count" "$warning_count"
  epoch_now > "${VMW_STATE_DIR}/last-sweep"
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
       --argjson now "$now" \
       '.[$id].expires_at = null |
        .[$id].duration = "indefinite" |
        .[$id].mode = "indefinite" |
        .[$id].warned_t1 = false |
        .[$id].warned_t2 = false |
        .[$id].last_active = $now' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
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
        .[$id].warned_t2 = false |
        .[$id].last_active = $now' "$LEASES_FILE" > "$tmp" && mv "$tmp" "$LEASES_FILE"
  fi

  local vm_name
  vm_name=$(lease_get "$machine_id" "vm_name")
  event_log "lease_extended" "$vm_name" "$machine_id" "$duration"
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
  lease_set_halted "$machine_id"
  event_log "halted" "$vm_name" "$machine_id" "manual"
  echo "Halted $vm_name and removed lease."
}

cmd_destroy() {
  init_state
  local name="${1:-.}"
  local machine_id
  machine_id=$(resolve_vm "$name")

  local vfp=""
  local vm_name=""
  local machine_name="default"

  # Gather VM info from lease
  if lease_exists "$machine_id"; then
    vfp=$(lease_get "$machine_id" "vagrantfile_path")
    vm_name=$(lease_get "$machine_id" "vm_name")
  fi

  # Fall back to machine index
  if [ -z "$vfp" ] || [ -z "$vm_name" ]; then
    local machines
    machines=$(read_machine_index)
    [ -z "$vfp" ] && vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
    [ -z "$vm_name" ] && vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')
    machine_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].name // "default"')
  fi

  if [ -z "$vfp" ]; then
    die "Cannot determine Vagrantfile path for $machine_id"
  fi

  echo "Destroying $vm_name..."

  if [ -d "$vfp" ]; then
    # Normal path: directory exists, use vagrant destroy
    VAGRANT_CWD="$vfp" vagrant destroy -f 2>&1 || die "Failed to destroy $vm_name"
  else
    # Directory is gone — try VBoxManage fallback
    echo "Vagrantfile directory no longer exists: $vfp"
    local vbox_uuid
    vbox_uuid=$(resolve_vbox_uuid "$vfp" "$machine_name")
    if [ -n "$vbox_uuid" ] && command -v VBoxManage >/dev/null 2>&1; then
      # Check if VM is still registered in VirtualBox
      if VBoxManage showvminfo "$vbox_uuid" >/dev/null 2>&1; then
        echo "Removing VM via VBoxManage (UUID: $vbox_uuid)..."
        VBoxManage unregistervm "$vbox_uuid" --delete 2>&1 || {
          echo "Warning: VBoxManage unregistervm failed, continuing with cleanup"
        }
      else
        echo "VM not registered in VirtualBox, cleaning up vm-ward state only."
      fi
    else
      echo "No VirtualBox UUID found, cleaning up vm-ward state only."
    fi
  fi

  # Remove lease if one exists
  if lease_exists "$machine_id"; then
    lease_remove "$machine_id"
  fi

  # Remove from Vagrant machine index
  remove_from_machine_index "$machine_id"

  event_log "destroyed" "$vm_name" "$machine_id" "manual"
  echo "Destroyed $vm_name and removed lease."
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
  event_log "lease_exempted" "$vm_name" "$machine_id" ""
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

  # Remove any existing service first (idempotent reinstall)
  launchctl bootout "gui/${uid}/com.strubio.vm-ward" 2>/dev/null || true

  # Enable the service before bootstrap (matches Homebrew's pattern)
  launchctl enable "gui/${uid}/com.strubio.vm-ward"

  # Bootstrap into the GUI domain
  local lctl_err
  lctl_err=$(launchctl bootstrap "gui/${uid}" "$plist_dst" 2>&1) || {
    die "Failed to load launchd plist: ${lctl_err:-unknown error}"
  }

  echo "vm-ward daemon installed and started."
  echo "  Plist: $plist_dst"
  echo "  Logs:  ${VMW_STATE_DIR}/sweep.log"
}

cmd_uninstall() {
  local plist_dst="$HOME/Library/LaunchAgents/com.strubio.vm-ward.plist"

  local uid
  uid=$(id -u)
  launchctl bootout "gui/${uid}/com.strubio.vm-ward" 2>/dev/null || true

  rm -f "$plist_dst"
  echo "vm-ward daemon uninstalled."
}

cmd_peek() {
  init_state
  local name="${1:-.}"
  local machine_id
  machine_id=$(resolve_vm "$name")

  local machines
  machines=$(read_machine_index)
  local vfp vm_name machine_name vbox_uuid
  vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
  vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')
  machine_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].name // "default"')
  vbox_uuid=$(resolve_vbox_uuid "$vfp" "$machine_name")

  local running_vms
  running_vms=$(get_running_vms)
  if [ -z "$vbox_uuid" ] || ! echo "$running_vms" | grep -qF "$vbox_uuid" 2>/dev/null; then
    die "VM '$vm_name' is not running"
  fi

  local peek_output
  peek_output=$(perl -e 'alarm(shift); exec @ARGV' 10 bash -c "VAGRANT_CWD=\"$vfp\" vagrant ssh -c '
    echo \"===TERMINAL_LOG===\"
    if [ -f ~/.local/state/terminal-logs/current.log ]; then
      tail -c 32768 ~/.local/state/terminal-logs/current.log
    else
      echo \"(no terminal session log found)\"
    fi
    echo \"===PROCESSES===\"
    ps aux --sort=-%cpu | head -20
  ' -- -o StrictHostKeyChecking=accept-new -o BatchMode=yes -o LogLevel=ERROR 2>/dev/null") || {
    die "Failed to connect to VM '$vm_name' (timeout or SSH error)"
  }

  echo "$peek_output"
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
  status                     Interactive dashboard (default)
  status --json              Machine-readable JSON output
  extend <name|.> [duration] Extend lease (default: 4h from now)
  halt <name|.>              Immediately halt a VM and remove lease
  destroy <name|.>           Destroy a VM, delete its disk, and remove lease
  exempt <name|.>            Exempt a VM from auto-halt
  update [.|name|--all] [--provision]  Update copier templates
  peek [name|.]              Peek inside a running VM
  sweep [--no-activity]      Run enforcement loop (called by launchd)
  install                    Install launchd daemon
  uninstall                  Remove launchd daemon
  config get|set <key> [val]  Get or set config values
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
  vmw update .            Update current project's copier template
  vmw update --all        Update all copier-managed projects
  vmw peek .              Peek at current project's VM

Activity detection:
  Sweep checks running VMs for CPU activity via VBoxManage metrics
  and resets leases for active VMs. Disable via config: activity_detection.enabled = false
EOF
}

cmd_config() {
  case "${1:-}" in
    get) config_get "${2:?key required}" "${3:-}" ;;
    set) config_set "${2:?key required}" "${3:?value required}" ;;
    *)   die "Usage: vmw config get|set <jq_path> [value]" ;;
  esac
}

main() {
  local cmd="${1:-status}"
  shift || true

  case "$cmd" in
    status)      cmd_status "$@" ;;
    extend)      cmd_extend "$@" ;;
    halt)        cmd_halt "$@" ;;
    destroy)     cmd_destroy "$@" ;;
    exempt)      cmd_exempt "$@" ;;
    sweep)       cmd_sweep "$@" ;;
    install)     cmd_install "$@" ;;
    uninstall)   cmd_uninstall "$@" ;;
    tui)
      local tui_bin="${VMW_ROOT}/bin/vmw-tui"
      if [ ! -x "$tui_bin" ]; then
        die "TUI binary not found. Install via 'brew upgrade vm-ward' or build with 'cd tui && go build -o ../bin/vmw-tui'"
      fi
      VMW_PATH="${VMW_ROOT}/bin/vmw" exec "$tui_bin" "$@"
      ;;
    update)
      source "${VMW_ROOT}/lib/vmw-update.sh"
      cmd_update "$@"
      ;;
    peek)        cmd_peek "$@" ;;
    config)      cmd_config "$@" ;;
    tmux-status) cmd_tmux_status "$@" ;;
    version)     vmw_version ;;
    help|--help|-h) cmd_help ;;
    *)           die "Unknown command: $cmd. Run 'vmw help' for usage." ;;
  esac
}
