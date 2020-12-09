package util

import "gopkg.in/toast.v1"

// Notification constructs the title and message for the toast notification
type Notification struct {
	Title   string
	Message string
	Icon    string
}

// SendToastNotification will notify the user via toast
func SendToastNotification(appName string, n Notification) error {
	notification := toast.Notification{
		AppID:    appName,
		Title:    n.Title,
		Message:  n.Message,
		Icon:     n.Icon,
		Duration: toast.Short,
		Audio:    "silent",
	}
	if err := notification.Push(); err != nil {
		return err
	}
	return nil
}
