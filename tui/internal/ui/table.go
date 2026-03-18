package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/strubio-ray/vm-ward/tui/internal/vmw"
)

// Column widths
const (
	colName     = 20
	colProject  = 25
	colState    = 10
	colLease    = 12
	colTimeLeft = 16
	colActivity = 10
	colLastAct  = 14
	colTemplate = 10
)

func tableWidth() int {
	return colName + colProject + colState + colLease + colTimeLeft + colActivity + colLastAct + colTemplate + 7 // 7 for spacing
}

// RenderHeader returns the bold header row.
func RenderHeader() string {
	return HeaderStyle.Render(fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		colName, "VM NAME",
		colProject, "PROJECT",
		colState, "STATE",
		colLease, "LEASE",
		colTimeLeft, "TIME LEFT",
		colActivity, "ACTIVITY",
		colLastAct, "LAST ACTIVE",
		colTemplate, "TEMPLATE",
	))
}

// RenderSeparator returns a line of ─ characters.
func RenderSeparator(width int) string {
	w := width
	if w <= 0 {
		w = tableWidth()
	}
	return Dim.Render(strings.Repeat("─", w))
}

// RenderRow renders a single VM row with appropriate coloring.
func RenderRow(vm vmw.VM, now time.Time, selected bool) string {
	name := truncate(vm.Name, colName)
	project := truncate(filepath.Base(vm.Path), colProject)
	state := truncate(vm.State, colState)
	lease := truncate(vm.Lease, colLease)
	timeLeft := truncate(TimeLeft(vm, now), colTimeLeft)
	activity := truncate(formatActivity(vm.LastActivity), colActivity)
	lastActive := truncate(FormatAgo(lastActiveTime(vm), now), colLastAct)
	tplVer := "—"
	if vm.TemplateVersion != nil {
		tplVer = *vm.TemplateVersion
	}
	template := truncate(tplVer, colTemplate)

	line := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		colName, name,
		colProject, project,
		colState, state,
		colLease, lease,
		colTimeLeft, timeLeft,
		colActivity, activity,
		colLastAct, lastActive,
		colTemplate, template,
	)

	style := rowStyle(vm, now)
	if selected {
		style = style.Reverse(true)
	}
	return style.Render(line)
}

// RenderSectionHeader renders a dim italic section label.
func RenderSectionHeader(label string) string {
	return Dim.Italic(true).Render(label)
}

// TimeLeft calculates remaining time from ExpiresAt, falling back to the
// pre-formatted Remaining string for non-timed leases.
func TimeLeft(vm vmw.VM, now time.Time) string {
	if vm.ExpiresAt == nil {
		return vm.Remaining
	}
	secs := *vm.ExpiresAt - now.Unix()
	if secs <= 0 {
		return "expired"
	}

	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60

	var timeStr string
	if h > 0 {
		timeStr = fmt.Sprintf("%dh %dm", h, m)
	} else if m > 0 {
		timeStr = fmt.Sprintf("%dm %ds", m, s)
	} else {
		timeStr = fmt.Sprintf("%ds", s)
	}

	if vm.Duration != nil {
		return fmt.Sprintf("%s (%s)", timeStr, *vm.Duration)
	}
	return timeStr
}

// FormatAgo formats an epoch timestamp as a relative time string.
func FormatAgo(epoch *int64, now time.Time) string {
	if epoch == nil {
		return "—"
	}
	diff := now.Unix() - *epoch
	if diff < 0 {
		return "just now"
	}
	switch {
	case diff < 60:
		return "just now"
	case diff < 3600:
		return fmt.Sprintf("%dm ago", diff/60)
	case diff < 86400:
		return fmt.Sprintf("%dh ago", diff/3600)
	default:
		return fmt.Sprintf("%dd ago", diff/86400)
	}
}

func lastActiveTime(vm vmw.VM) *int64 {
	if vm.HaltedAt != nil {
		return vm.HaltedAt
	}
	return vm.LastActive
}

func formatActivity(activity *string) string {
	if activity == nil {
		return "pending"
	}
	return *activity
}

// ActivityStyle returns the lipgloss style for the activity value.
func ActivityStyle(activity *string) lipgloss.Style {
	if activity == nil {
		return Yellow
	}
	switch *activity {
	case "active":
		return Green
	case "idle":
		return Dim
	default:
		return Dim
	}
}

func rowStyle(vm vmw.VM, now time.Time) lipgloss.Style {
	if !vm.Managed {
		return Dim
	}
	switch vm.Lease {
	case "halted":
		return Dim
	case "exempt", "indefinite":
		return Cyan
	case "expired":
		return Red
	case "pending":
		return Yellow
	case "none":
		if vm.State == "poweroff" {
			return Dim
		}
		return lipgloss.NewStyle()
	case "active":
		return leaseProgressStyle(vm, now)
	default:
		return lipgloss.NewStyle()
	}
}

func leaseProgressStyle(vm vmw.VM, now time.Time) lipgloss.Style {
	if vm.ExpiresAt == nil || vm.LastActive == nil || vm.Duration == nil {
		return Green
	}
	secsLeft := *vm.ExpiresAt - now.Unix()
	if secsLeft <= 0 {
		return Red
	}

	// Calculate total duration from the Duration string is complex;
	// use expires_at - created_at approach via last_active as proxy.
	// Simpler: use ratio of remaining / total.
	// We approximate total from ExpiresAt and the lease creation.
	// For accurate ratio, use the remaining vs original duration.
	// Since we have Duration as a string like "4h", we need to parse it.
	// For now, use simple thresholds based on absolute remaining time.
	totalSecs := parseDurationSecs(*vm.Duration)
	if totalSecs <= 0 {
		return Green
	}
	elapsed := totalSecs - secsLeft
	ratio := float64(elapsed) / float64(totalSecs)

	switch {
	case ratio >= 0.875:
		return Red
	case ratio >= 0.5:
		return Yellow
	default:
		return Green
	}
}

func parseDurationSecs(d string) int64 {
	var total int64
	var num int64
	for _, c := range d {
		switch {
		case c >= '0' && c <= '9':
			num = num*10 + int64(c-'0')
		case c == 'h':
			total += num * 3600
			num = 0
		case c == 'm':
			total += num * 60
			num = 0
		case c == 's':
			total += num
			num = 0
		}
	}
	return total
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
