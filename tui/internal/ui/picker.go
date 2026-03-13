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
	var parts []string
	for i, preset := range DurationPresets {
		if i == cursor {
			parts = append(parts, PickerActive.Render(" "+preset+" "))
		} else {
			parts = append(parts, PickerInactive.Render(" "+preset+" "))
		}
	}
	label := fmt.Sprintf("  Extend '%s': ", vmName)
	return ConfirmStyle.Render(label) + strings.Join(parts, " ") + FooterDim.Render("  ←/→ select  Enter confirm  Esc cancel")
}
