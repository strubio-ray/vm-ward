package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// RenderFooter renders the footer with help view on the left and status on the right.
// Status sections are progressively dropped as the terminal narrows.
func RenderFooter(width int, helpView string, lastSweep *int64, lastRefresh, now time.Time) string {
	left := helpView
	sweep := sweepCountdown(lastSweep, now)
	refresh := refreshAge(lastRefresh, now)

	leftW := lipgloss.Width(left)
	sweepW := lipgloss.Width(sweep)
	refreshW := lipgloss.Width(refresh)

	const minGap = 2

	// Wide: help + sweep + refresh
	if leftW+sweepW+refreshW+minGap*2 <= width {
		gap := width - leftW - sweepW - refreshW
		leftGap := gap / 2
		rightGap := gap - leftGap
		return left + strings.Repeat(" ", leftGap) + sweep + strings.Repeat(" ", rightGap) + refresh
	}

	// Medium: help + refresh (drop sweep)
	if leftW+refreshW+minGap <= width {
		gap := width - leftW - refreshW
		return left + strings.Repeat(" ", gap) + refresh
	}

	// Narrow: help only
	return left
}

func sweepCountdown(lastSweep *int64, now time.Time) string {
	if lastSweep == nil {
		return FooterDim.Render("Next sweep: unknown")
	}
	nextSweep := *lastSweep + 300 - now.Unix()
	if nextSweep <= 0 {
		return Yellow.Render("Next sweep: overdue")
	}
	m := nextSweep / 60
	s := nextSweep % 60
	return FooterDim.Render(fmt.Sprintf("Next sweep: ~%dm %ds", m, s))
}

func refreshAge(lastRefresh, now time.Time) string {
	age := now.Sub(lastRefresh)
	secs := int(age.Seconds())
	style := FooterDim
	if secs > 90 {
		style = Yellow
	}
	return style.Render(fmt.Sprintf("Refreshed %ds ago", secs))
}
