# vm-ward

Auto-halt daemon for forgotten Vagrant VMs. Pure bash — no build system.

## Project Structure

```
bin/vmw              # Entry point — resolves symlinks, detects platform, dispatches subcommands
lib/vmw-common.sh    # Shared utilities (logging, JSON helpers, duration parsing)
lib/vmw-host.sh      # macOS host daemon (Vagrant/VBox integration, leases, sweep)
share/vm-ward/       # launchd plist template
```

## Key Concepts

- **Vagrant machine IDs** (hex hashes) ≠ **VirtualBox UUIDs** (dashed format). Use `resolve_vbox_uuid()` to bridge them.
- **Leases** track how long a VM is allowed to run. Stored in `~/.local/state/vm-ward/leases.json`.
- **Sweep** runs every 5 min via launchd — warns at T1 (50%) and T2 (87.5%), halts on expiry. Activity detection uses `VBoxManage metrics query` (host-side CPU%). First sweep after VM start returns "idle" (metrics need one sampling period to populate).
- **Version placeholder**: `bin/vmw` contains `VMW_VERSION="%%VERSION%%"` — injected by Homebrew formula at install time.

## Release Flow

1. Commit with conventional commit format (`feat:`, `fix:`, etc.)
2. Run `cog bump --patch` (or `--minor`/`--major`)
3. Cocogitto creates the version tag and post-bump hooks push tag + commits to origin
4. GitHub Actions (`bump-homebrew.yml`) detects the `v*` tag push
5. `mislav/bump-homebrew-formula-action` updates the formula in `strubio-ray/homebrew-tap`
6. Users get the update via `brew update && brew upgrade vm-ward` (`brew update` refreshes the tap index first)

## Dependencies

Runtime: `jq`, `vagrant`, `VBoxManage`

## Useful Commands

```bash
bash -n lib/vmw-host.sh          # Syntax check
vmw status                       # Show all VMs and lease status
vmw status --json                # JSON output
VBoxManage list runningvms       # Cross-check running VMs
cog bump --patch                 # Release a patch version
```
