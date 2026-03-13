package model

import "charm.land/bubbletea/v2"

// matchKey checks if a key message matches the given string.
func matchKey(msg tea.KeyPressMsg, keys ...string) bool {
	for _, k := range keys {
		if msg.String() == k {
			return true
		}
	}
	return false
}
