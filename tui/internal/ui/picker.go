package ui

import (
	"fmt"
	"strings"
)

// DurationPresets are the available choices when extending a lease.
var DurationPresets = []string{"1h", "2h", "4h", "8h", "overnight", "weekend", "indefinite"}

// DefaultPresetIndex is the index of the default duration (4h).
const DefaultPresetIndex = 2

// RenderPicker renders the horizontal duration selector.
func RenderPicker(cursor int, vmName string) string {
	label := fmt.Sprintf("  Extend '%s': ", vmName)
	return renderPickerRow(label, DurationPresets, cursor)
}

// renderPickerRow renders a labeled horizontal preset selector with navigation hints.
func renderPickerRow(label string, presets []string, cursor int) string {
	parts := make([]string, len(presets))
	for i, preset := range presets {
		if i == cursor {
			parts[i] = PickerActive.Render(" " + preset + " ")
		} else {
			parts[i] = PickerInactive.Render(" " + preset + " ")
		}
	}
	return ConfirmStyle.Render(label) + strings.Join(parts, " ") + FooterDim.Render("  ←/→ select  Enter confirm  Esc cancel")
}

