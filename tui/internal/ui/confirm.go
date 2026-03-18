package ui

import "fmt"

// RenderConfirm renders the confirmation bar for a destructive action.
func RenderConfirm(action, vmName string) string {
	prompt := fmt.Sprintf("%s '%s'? [y/n]", capitalize(action), vmName)
	return ConfirmStyle.Render("  " + prompt)
}

// RenderProvisionConfirm renders the post-update provisioning prompt.
func RenderProvisionConfirm() string {
	return ConfirmStyle.Render("  Reprovision VMs after update? [y/n/esc to cancel]")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}
