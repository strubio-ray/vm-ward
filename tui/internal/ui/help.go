package ui

import (
	"strings"

	"charm.land/bubbles/v2/help"
)

// RenderHelpOverlay renders a full-screen help overlay with all keybindings.
func RenderHelpOverlay(h help.Model, km help.KeyMap, width, height int) string {
	var b strings.Builder

	title := Bold.Render("  Keybindings")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(Dim.Render(strings.Repeat("─", width)))
	b.WriteString("\n\n")

	// Render full help using a copy with ShowAll=true
	fullHelp := h
	fullHelp.ShowAll = true
	fullHelp.SetWidth(width - 4)

	rendered := fullHelp.View(km)
	for _, line := range strings.Split(rendered, "\n") {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(FooterDim.Render("  ? or Esc to close"))

	return b.String()
}
