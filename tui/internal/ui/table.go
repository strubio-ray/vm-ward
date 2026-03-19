package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/strubio-ray/vm-ward/tui/internal/vmw"
)

// Column describes a table column with responsive layout metadata.
type Column struct {
	ID       string // key for cellValue switch
	Label    string // header text
	Width    int    // preferred width
	MinWidth int    // minimum before hiding (same as Width = not shrinkable)
	Priority int    // 0 = essential/never hide, 1 = hide first, 4 = hide last
}

// columns defines the table layout in display order.
var columns = []Column{
	{ID: "project", Label: "PROJECT", Width: 25, MinWidth: 12, Priority: 0},
	{ID: "state", Label: "STATE", Width: 10, MinWidth: 10, Priority: 0},
	{ID: "lease", Label: "LEASE", Width: 12, MinWidth: 12, Priority: 4},
	{ID: "timeleft", Label: "TIME LEFT", Width: 16, MinWidth: 16, Priority: 0},
	{ID: "activity", Label: "ACTIVITY", Width: 10, MinWidth: 10, Priority: 3},
	{ID: "lastactive", Label: "LAST ACTIVE", Width: 14, MinWidth: 14, Priority: 2},
	{ID: "template", Label: "TEMPLATE", Width: 10, MinWidth: 10, Priority: 1},
}

// LayoutResult holds the computed column layout for a given terminal width.
type LayoutResult struct {
	Cols   []Column
	Hidden bool
}

// VisibleColumns computes which columns to show and their widths for the
// given available width. It shrinks flexible columns first, then hides
// non-essential columns by priority (lowest priority hidden first).
func VisibleColumns(availableWidth int) LayoutResult {
	cols := make([]Column, len(columns))
	copy(cols, columns)

	total := func(cc []Column) int {
		w := 0
		for _, c := range cc {
			w += c.Width
		}
		if len(cc) > 1 {
			w += len(cc) - 1 // one space between columns
		}
		return w
	}

	// Phase 1: shrink flexible columns to MinWidth.
	if total(cols) > availableWidth {
		for i := range cols {
			if cols[i].MinWidth < cols[i].Width {
				cols[i].Width = cols[i].MinWidth
			}
		}
	}

	// Phase 2: hide non-essential columns, lowest priority first.
	hidden := false
	for pri := 1; pri <= 4; pri++ {
		if total(cols) <= availableWidth {
			break
		}
		var kept []Column
		for _, c := range cols {
			if c.Priority == pri {
				hidden = true
				continue
			}
			kept = append(kept, c)
		}
		cols = kept
	}

	// If columns were hidden, reserve space for the " …" indicator and
	// re-check — drop another column if needed.
	if hidden && total(cols)+2 > availableWidth {
		for pri := 1; pri <= 4; pri++ {
			if total(cols)+2 <= availableWidth {
				break
			}
			var kept []Column
			for _, c := range cols {
				if c.Priority == pri {
					continue
				}
				kept = append(kept, c)
			}
			cols = kept
		}
	}

	return LayoutResult{Cols: cols, Hidden: hidden}
}

func tableWidth() int {
	w := 0
	for _, c := range columns {
		w += c.Width
	}
	return w + len(columns) - 1
}

// cellValue extracts and formats a cell value from a VM by column ID.
func cellValue(id string, vm vmw.VM, now time.Time) string {
	switch id {
	case "project":
		return filepath.Base(vm.Path)
	case "state":
		return vm.State
	case "lease":
		return vm.Lease
	case "timeleft":
		return TimeLeft(vm, now)
	case "activity":
		return formatActivity(vm.LastActivity)
	case "lastactive":
		return FormatAgo(lastActiveTime(vm), now)
	case "template":
		if vm.TemplateVersion != nil {
			return *vm.TemplateVersion
		}
		return "—"
	default:
		return ""
	}
}

// RenderHeader returns the bold header row, responsive to terminal width.
func RenderHeader(width int) string {
	layout := VisibleColumns(width)
	parts := make([]string, len(layout.Cols))
	for i, col := range layout.Cols {
		parts[i] = fmt.Sprintf("%-*s", col.Width, col.Label)
	}
	line := strings.Join(parts, " ")
	rendered := HeaderStyle.Render(line)
	if layout.Hidden {
		rendered += " " + Dim.Render("…")
	}
	return rendered
}

// RenderSeparator returns a line of ─ characters.
func RenderSeparator(width int) string {
	w := width
	if w <= 0 {
		w = tableWidth()
	}
	return Dim.Render(strings.Repeat("─", w))
}

// RenderRow renders a single VM row with appropriate coloring, responsive to terminal width.
func RenderRow(vm vmw.VM, now time.Time, selected bool, width int) string {
	layout := VisibleColumns(width)
	parts := make([]string, len(layout.Cols))
	for i, col := range layout.Cols {
		val := truncate(cellValue(col.ID, vm, now), col.Width)
		parts[i] = fmt.Sprintf("%-*s", col.Width, val)
	}
	line := strings.Join(parts, " ")

	style := rowStyle(vm, now)
	if selected {
		style = style.Reverse(true)
	}
	rendered := style.Render(line)
	if layout.Hidden {
		rendered += " " + Dim.Render("…")
	}
	return rendered
}

// RenderSectionHeader renders a dim italic section label.
func RenderSectionHeader(label string) string {
	return Dim.Italic(true).Render(label)
}

// RenderCollapsedSummary renders a dim one-line summary for a collapsed section.
func RenderCollapsedSummary(count int, sectionLabel string) string {
	noun := "VM"
	if count != 1 {
		noun = "VMs"
	}
	return Dim.Italic(true).Render(fmt.Sprintf("  %d %s %s", count, sectionLabel, noun))
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
