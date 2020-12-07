package plugin

import "context"

type Task struct {
	Event Event
	Value interface{}
}

type Plugin interface {
	Initialize() error
	Run(haltCtx context.Context) <-chan error
	Notify(t Task)
}
