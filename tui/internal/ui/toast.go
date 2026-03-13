package ui

import "time"

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

// Render returns the styled toast string.
func (t *Toast) Render() string {
	if t.IsError {
		return ToastErrStyle.Render("  " + t.Message)
	}
	return ToastStyle.Render("  " + t.Message)
}
