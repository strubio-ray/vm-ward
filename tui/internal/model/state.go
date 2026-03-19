package model

// ViewState represents the current UI mode.
type ViewState int

const (
	StateNormal           ViewState = iota // Table view, accepting commands
	StateConfirm                           // Awaiting y/n for destructive action
	StateConfirmProvision                  // Awaiting y/n for post-update provisioning
	StatePicker                            // Duration preset selector for extend
	StatePeek                              // Full-screen peek overlay
)
