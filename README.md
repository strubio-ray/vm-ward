# vm-ward

Auto-halt daemon for forgotten Vagrant VMs.

Discovers running VMs via Vagrant's machine index, assigns time-limited leases, warns users before expiry via tmux, and halts expired VMs automatically.

## Architecture

Single binary (`vmw`), platform-detected behavior:

- **macOS** (host): daemon that discovers VMs, manages leases, and enforces expiry
- **Linux** (guest): warning agent that displays tmux messages when the host pushes alerts

## Install

```bash
brew install strubio-ray/tap/vm-ward
```

Or clone and use directly:

```bash
git clone https://github.com/strubio-ray/vm-ward.git
cd vm-ward
./bin/vmw version
```

### Dependencies

- `jq` — JSON processing
- `vagrant` — VM management (host only)
- `VBoxManage` — VM state detection (host only)
- `tmux` — warning display (guest only)

## Usage

### Host commands (macOS)

```bash
vmw status              # Show all VMs and lease status
vmw status --json       # Machine-readable output
vmw extend . 8h         # Extend current project's VM by 8 hours
vmw extend . overnight  # Extend until tomorrow morning
vmw extend . indefinite # No expiry
vmw halt .              # Halt current project's VM now
vmw exempt .            # Exempt from auto-halt
vmw sweep               # Run enforcement loop (called by launchd)
vmw install             # Install launchd daemon
vmw uninstall           # Remove launchd daemon
vmw tmux-status         # Print tmux status bar segment
```

### Guest commands (Linux)

```bash
vmw warn "2h 13m remaining"          # Display warning in tmux
vmw warn --urgent "15m remaining"    # Urgent warning (longer display)
```

### Duration formats

| Format       | Meaning                |
|-------------|------------------------|
| `4h`        | 4 hours                |
| `30m`       | 30 minutes             |
| `overnight` | 14 hours (configurable)|
| `weekend`   | 48 hours               |
| `indefinite`| No expiry              |

## Daemon setup

```bash
vmw install    # Install and start launchd daemon (sweeps every 5 minutes)
vmw uninstall  # Stop and remove daemon
```

The daemon runs `vmw sweep` every 5 minutes, which:

1. Discovers running VMs via Vagrant machine index
2. Creates retroactive leases for VMs without one (default: 4h)
3. Sends T1 warning at 50% elapsed (via `vagrant ssh -c "vmw warn"`)
4. Sends T2 urgent warning at 87.5% elapsed
5. Halts expired VMs via `vagrant halt`

## Configuration

Optional config file at `~/.config/vm-ward/config.json`:

```json
{
  "default_duration": "4h",
  "presets": {
    "overnight": "14h",
    "weekend": "48h"
  },
  "warnings": {
    "t1_ratio": 0.5,
    "t2_ratio": 0.875
  }
}
```

## State

Lease data is stored in `~/.local/state/vm-ward/leases.json`. Daemon logs go to `~/.local/state/vm-ward/sweep.log`.

## tmux integration

Add to your `tmux.conf`:

```
set -g status-right '#(vmw tmux-status) | %H:%M'
```

## License

MIT
