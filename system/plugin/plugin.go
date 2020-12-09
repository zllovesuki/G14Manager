package plugin

import "context"

// Notification facilitates the hardware event to be sent to plugins
type Notification struct {
	Event Event
	Value interface{}
}

type Callback struct {
	Event Event
	Value interface{}
}

// Plugin will receive hardware events from controller
type Plugin interface {
	Initialize() error
	Run(haltCtx context.Context, cb chan<- Callback) <-chan error
	Notify(t Notification)
}
