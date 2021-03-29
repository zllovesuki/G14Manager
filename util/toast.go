package util

import "gopkg.in/toast.v1"

// Notification constructs the title and message for the toast notification
type Notification struct {
	AppName string
	Title   string
	Message string
	Icon    string
}

// SendToastNotification will notify the user via toast
func SendToastNotification(n Notification) error {
	notification := toast.Notification{
		AppID:    n.AppName,
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
