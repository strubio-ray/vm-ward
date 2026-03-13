package model

// ViewState represents the current UI mode.
type ViewState int

const (
	StateNormal  ViewState = iota // Table view, accepting commands
	StateConfirm                  // Awaiting y/n for destructive action
	StatePicker                   // Duration preset selector for extend
)
