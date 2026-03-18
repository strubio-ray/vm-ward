package ui

import (
	"strings"
	"time"
)

// Toast represents a temporary feedback message.
type Toast struct {
	Message   string
	IsError   bool
	ExpiresAt time.Time
}

// NewToast creates a toast that auto-clears after the given duration.
func NewToast(msg string, isError bool, ttl time.Duration) *Toast {
	return &Toast{
		Message:   msg,
		IsError:   isError,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// IsVisible returns true if the toast hasn't expired yet.
func (t *Toast) IsVisible(now time.Time) bool {
	return now.Before(t.ExpiresAt)
}

// Lines returns the number of rendered lines (for layout calculations).
func (t *Toast) Lines() int {
	return strings.Count(t.Message, "\n") + 1
}

// Render returns the styled toast string.
func (t *Toast) Render() string {
	style := ToastStyle
	if t.IsError {
		style = ToastErrStyle
	}
	lines := strings.Split(t.Message, "\n")
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(style.Render("  " + line))
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
