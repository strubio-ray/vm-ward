# ─────────────────────────────────────────────────────────────────────────────
# vmw-update.sh — Copier template update support for vm-ward
# ─────────────────────────────────────────────────────────────────────────────

require_copier() {
  if ! command -v copier >/dev/null 2>&1; then
    die "copier is not installed. Install with: pipx install copier (or pip install copier)"
  fi
}

# Parse _commit and _src_path from a .copier-answers.yml file
# Usage: read_copier_answers "/path/to/.copier-answers.yml"
# Outputs two lines: _commit (line 1) and _src_path (line 2)
read_copier_answers() {
  local file="$1"
  local commit src_path
  commit=$(grep '^_commit:' "$file" 2>/dev/null | awk '{print $2}')
  src_path=$(grep '^_src_path:' "$file" 2>/dev/null | awk '{print $2}')
  echo "$commit"
  echo "$src_path"
}

# Run copier update on a single project directory
# Arguments: machine_id vagrantfile_path vm_name provision(true/false)
# Returns: 0 = success, 1 = failure, 2 = skipped (no template)
update_single_project() {
  local machine_id="$1" vfp="$2" vm_name="$3" provision="$4"
  local answers_file="${vfp}/.vm/.copier-answers.yml"

  if [ ! -d "$vfp" ]; then
    log "Warning: project directory does not exist: $vfp"
    return 1
  fi

  if [ ! -f "$answers_file" ]; then
    return 2
  fi

  local old_commit
  old_commit=$(grep '^_commit:' "$answers_file" 2>/dev/null | awk '{print $2}')

  log "Updating template for $vm_name ($vfp)..."
  local update_output
  if update_output=$(cd "$vfp" && copier update --defaults --trust .vm 2>&1); then
    local new_commit
    new_commit=$(grep '^_commit:' "$answers_file" 2>/dev/null | awk '{print $2}')
    event_log "template_updated" "$vm_name" "$machine_id" "${old_commit:-unknown} -> ${new_commit:-unknown}"
    log "Updated $vm_name: ${old_commit:-unknown} -> ${new_commit:-unknown}"

    if [ "$provision" = true ]; then
      local running_vms vbox_uuid machine_name
      running_vms=$(get_running_vms)
      local machines
      machines=$(read_machine_index)
      machine_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].name // "default"')
      vbox_uuid=$(resolve_vbox_uuid "$vfp" "$machine_name")

      if [ -n "$vbox_uuid" ] && echo "$running_vms" | grep -qF "$vbox_uuid" 2>/dev/null; then
        log "Reloading VM $vm_name with provisioning..."
        if ! VAGRANT_CWD="$vfp" vagrant reload --provision 2>&1; then
          log "Warning: vagrant reload --provision failed for $vm_name"
          return 1
        fi
      else
        log "Skipping provision for $vm_name (not running)"
      fi
    fi
    return 0
  else
    local copier_exit=$?
    event_log "template_update_failed" "$vm_name" "$machine_id" "copier exit code $copier_exit"
    log "Failed to update template for $vm_name"
    echo "$update_output" >&2
    return 1
  fi
}

cmd_update() {
  require_copier
  init_state

  local target="" provision=false
  while [ $# -gt 0 ]; do
    case "$1" in
      --provision) provision=true ;;
      --all)       target="--all" ;;
      *)           target="$1" ;;
    esac
    shift
  done

  [ -z "$target" ] && target="."

  local machines
  machines=$(read_machine_index)

  if [ "$target" != "--all" ]; then
    # Single target mode
    local machine_id vfp vm_name
    machine_id=$(resolve_vm "$target")
    vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
    vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')

    if [ -z "$vfp" ]; then
      die "No vagrantfile path found for VM: $target"
    fi

    local result
    update_single_project "$machine_id" "$vfp" "$vm_name" "$provision"
    result=$?
    case $result in
      0) echo "Successfully updated $vm_name" ;;
      2) die "Not a copier-managed project: $vfp (no .vm/.copier-answers.yml)" ;;
      *) exit 1 ;;
    esac
    return
  fi

  # Batch mode (--all)
  local updated=0 skipped=0 failed=0
  local failed_names=()

  for machine_id in $(echo "$machines" | jq -r 'keys[]' 2>/dev/null); do
    local vfp vm_name
    vfp=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].vagrantfile_path // ""')
    vm_name=$(echo "$machines" | jq -r --arg id "$machine_id" '.[$id].extra_data.box.name // .[$id].name // "unknown"')

    [ -z "$vfp" ] && continue

    local result
    update_single_project "$machine_id" "$vfp" "$vm_name" "$provision"
    result=$?
    case $result in
      0) updated=$((updated + 1)) ;;
      2) skipped=$((skipped + 1)) ;;
      *)
        failed=$((failed + 1))
        failed_names+=("$vm_name")
        ;;
    esac
  done

  if [ "$updated" -eq 0 ] && [ "$failed" -eq 0 ] && [ "$skipped" -gt 0 ]; then
    echo "No copier-managed VMs found."
    return 0
  fi

  echo ""
  echo "Update summary: $updated updated, $skipped skipped (no template), $failed failed"

  if [ "$failed" -gt 0 ]; then
    echo ""
    echo "Failed to update:"
    for name in "${failed_names[@]}"; do
      echo "  - $name"
    done
    exit 1
  fi
}
