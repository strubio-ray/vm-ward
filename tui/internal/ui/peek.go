package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/vt"
)

// ParsePeekOutput splits raw peek output into terminal log and processes sections.
func ParsePeekOutput(raw string) (termLog string, processes string) {
	parts := strings.SplitN(raw, "===PROCESSES===", 2)
	if len(parts) == 2 {
		termLog = parts[0]
		processes = strings.TrimSpace(parts[1])
	} else {
		termLog = raw
	}
	termLog = strings.TrimPrefix(termLog, "===TERMINAL_LOG===\n")
	termLog = strings.TrimSpace(termLog)
	return
}

// RenderTerminalLog processes raw script output through a virtual terminal
// emulator and returns clean SGR-attributed text.
func RenderTerminalLog(raw string, cols, rows int) string {
	term := vt.NewEmulator(cols, rows)
	term.Write([]byte(raw))
	return term.Render()
}

// RenderPeekOverlay renders the full-screen peek view with header, terminal
// output, process list, and footer.
func RenderPeekOverlay(vmName, termRendered, processes string, scroll, width, height int, loading bool) string {
	var b strings.Builder

	// Header
	header := fmt.Sprintf("  Peek: %s", vmName)
	if loading {
		header += "  (refreshing...)"
	}
	b.WriteString(Bold.Render(header))
	b.WriteString("\n")
	b.WriteString(Dim.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Terminal log section
	b.WriteString(Bold.Render("  Terminal Output"))
	b.WriteString("\n\n")

	termLines := strings.Split(termRendered, "\n")

	// Process section
	b.WriteString(Dim.Render(strings.Repeat("─", width)))
	b.WriteString("\n")
	b.WriteString(Bold.Render("  Top Processes"))
	b.WriteString("\n\n")

	procLines := strings.Split(processes, "\n")

	// Combine all content lines for unified scrolling
	var allLines []string
	for _, line := range termLines {
		allLines = append(allLines, "  "+line)
	}
	allLines = append(allLines, "")
	allLines = append(allLines, Dim.Render(strings.Repeat("─", width)))
	allLines = append(allLines, Bold.Render("  Top Processes"))
	allLines = append(allLines, "")
	for _, line := range procLines {
		allLines = append(allLines, "  "+line)
	}

	// Reset builder and rebuild with scrolling
	b.Reset()

	// Fixed header (always visible)
	b.WriteString(Bold.Render(header))
	b.WriteString("\n")
	b.WriteString(Dim.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Scrollable viewport
	const headerHeight = 3 // header + separator + blank
	const footerHeight = 2 // blank + footer
	viewportHeight := height - headerHeight - footerHeight
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Clamp scroll
	maxScroll := len(allLines) - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	// Render visible lines
	end := scroll + viewportHeight
	if end > len(allLines) {
		end = len(allLines)
	}
	for _, line := range allLines[scroll:end] {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	scrollInfo := fmt.Sprintf("[%d/%d]", scroll+1, len(allLines))
	footerText := fmt.Sprintf("  ↑↓ scroll  r refresh  Esc close  %s", scrollInfo)
	b.WriteString(FooterDim.Render(footerText))

	return b.String()
}
