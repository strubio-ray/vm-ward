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

// ThresholdPresets are the available choices for the CPU threshold picker.
var ThresholdPresets = []string{"1%", "2%", "5%", "10%", "15%", "off"}

// DefaultThresholdIndex is the index of the default threshold (5%).
const DefaultThresholdIndex = 2

// RenderThresholdPicker renders the horizontal CPU threshold selector.
func RenderThresholdPicker(cursor int) string {
	return renderPickerRow("  CPU threshold: ", ThresholdPresets, cursor)
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

// ThresholdValue converts a preset string to a threshold integer and enabled flag.
func ThresholdValue(preset string) (threshold int, enabled bool) {
	if preset == "off" {
		return 0, false
	}
	var pct int
	fmt.Sscanf(preset, "%d%%", &pct)
	return pct, true
}

// ThresholdPresetIndex returns the index of the preset matching the given threshold.
func ThresholdPresetIndex(threshold int) int {
	if threshold <= 0 {
		return len(ThresholdPresets) - 1 // "off"
	}
	target := fmt.Sprintf("%d%%", threshold)
	for i, p := range ThresholdPresets {
		if p == target {
			return i
		}
	}
	return DefaultThresholdIndex
}
