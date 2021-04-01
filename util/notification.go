package util

import (
	"time"
)

// Notification constructs the title and message for the toast notification
type Notification struct {
	Message string
	Delay   time.Duration
}
