package ui

import (
	"fmt"
	"strings"
	"time"
)

const keyLegend = "q quit  ↑↓ select  e extend  h halt  d destroy  x exempt  i indef  s sweep  r refresh"

// RenderFooter renders the three-section footer: key legend, sweep countdown, refresh age.
func RenderFooter(width int, lastSweep *int64, lastRefresh, now time.Time) string {
	left := FooterDim.Render(keyLegend)
	center := sweepCountdown(lastSweep, now)
	right := refreshAge(lastRefresh, now)

	leftW := lipglossWidth(left)
	centerW := lipglossWidth(center)
	rightW := lipglossWidth(right)

	gap := width - leftW - centerW - rightW
	if gap < 2 {
		// Narrow terminal: just show key legend
		return left
	}
	leftGap := gap / 2
	rightGap := gap - leftGap

	return left + strings.Repeat(" ", leftGap) + center + strings.Repeat(" ", rightGap) + right
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

// lipglossWidth counts visible characters (approximation — strips ANSI).
func lipglossWidth(s string) int {
	// Simple ANSI strip for width calculation
	inEsc := false
	w := 0
	for _, c := range s {
		if c == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				inEsc = false
			}
			continue
		}
		w++
	}
	return w
}
