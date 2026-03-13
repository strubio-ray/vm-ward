package ui

import "charm.land/lipgloss/v2"

var (
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	Cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	Dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	Bold   = lipgloss.NewStyle().Bold(true)

	HeaderStyle  = lipgloss.NewStyle().Bold(true)
	FooterDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ToastStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	ToastErrStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	ConfirmStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))

	PickerActive   = lipgloss.NewStyle().Bold(true).Reverse(true)
	PickerInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	SelectedRow = lipgloss.NewStyle().Bold(true)
)
