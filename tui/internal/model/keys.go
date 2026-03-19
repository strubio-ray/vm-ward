package model

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"github.com/strubio-ray/vm-ward/tui/internal/vmw"
)

// Compile-time check that keyMap satisfies help.KeyMap.
var _ help.KeyMap = keyMap{}

type keyMap struct {
	// Tier 0 — always visible
	Help     key.Binding
	Quit     key.Binding
	Up       key.Binding
	Down     key.Binding
	Navigate key.Binding // display-only "↑↓ select" for help rendering

	// Tier 1 — context, high priority (running VM)
	Extend key.Binding
	Halt   key.Binding
	Peek   key.Binding

	// Tier 2 — context, medium priority
	Destroy    key.Binding
	Exempt     key.Binding
	Indefinite key.Binding

	// Tier 3 — context, low priority (dropped first on narrow terminals)
	Sweep     key.Binding
	Refresh   key.Binding
	Update    key.Binding
	UpdateAll key.Binding

	// Sub-state bindings (used for key.Matches only, not in short help)
	ConfirmYes  key.Binding
	ConfirmNo   key.Binding
	PickerLeft  key.Binding
	PickerRight key.Binding
	PickerEnter key.Binding
	Escape      key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Up:       key.NewBinding(key.WithKeys("up", "k")),
		Down:     key.NewBinding(key.WithKeys("down", "j")),
		Navigate: key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "select")),

		Extend: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "extend")),
		Halt:   key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "halt")),
		Peek:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "peek")),

		Destroy:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "destroy")),
		Exempt:     key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "exempt")),
		Indefinite: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "indef")),

		Sweep:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sweep")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Update:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update")),
		UpdateAll: key.NewBinding(key.WithKeys("shift+u"), key.WithHelp("U", "update-all")),

		ConfirmYes:  key.NewBinding(key.WithKeys("y")),
		ConfirmNo:   key.NewBinding(key.WithKeys("n")),
		PickerLeft:  key.NewBinding(key.WithKeys("left", "h")),
		PickerRight: key.NewBinding(key.WithKeys("right", "l")),
		PickerEnter: key.NewBinding(key.WithKeys("enter")),
		Escape:      key.NewBinding(key.WithKeys("esc")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit, k.Navigate,
		k.Extend, k.Halt, k.Peek,
		k.Destroy, k.Exempt, k.Indefinite,
		k.Sweep, k.Refresh, k.Update, k.UpdateAll,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Navigate, k.Help, k.Quit},
		{k.Extend, k.Halt, k.Peek},
		{k.Destroy, k.Exempt, k.Indefinite},
		{k.Update, k.UpdateAll, k.Sweep, k.Refresh},
	}
}

func (k *keyMap) updateKeyStates(vm *vmw.VM, hasPending bool, sweeping bool) {
	isRunning := vm != nil && vm.State == "running"
	hasVM := vm != nil
	hasTemplate := vm != nil && vm.TemplateVersion != nil
	actionable := hasVM && !hasPending

	k.Halt.SetEnabled(isRunning && actionable)
	k.Peek.SetEnabled(isRunning && actionable)
	k.Extend.SetEnabled(actionable)
	k.Destroy.SetEnabled(actionable)
	k.Exempt.SetEnabled(actionable)
	k.Indefinite.SetEnabled(actionable)
	k.Update.SetEnabled(hasTemplate && actionable)
	k.UpdateAll.SetEnabled(!hasPending)
	k.Sweep.SetEnabled(!sweeping)
}
