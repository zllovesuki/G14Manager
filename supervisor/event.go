package supervisor

import (
	"fmt"
	"log"

	"github.com/zllovesuki/G14Manager/util"

	"github.com/thejerf/suture/v4"
)

type EventHook struct {
	Notifier chan<- util.Notification
}

func (e *EventHook) Event(evt suture.Event) {
	log.Printf("[supervisor] event: %+v\n", evt)
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[supervisor] event hook panic: %+v\n", err)
		}
	}()
	m := evt.Map()
	switch evt.Type() {
	case suture.EventTypeServiceTerminate, suture.EventTypeServicePanic:
		e.Notifier <- util.Notification{
			Message: fmt.Sprintf("%s crashed unexpectedly, restarting...", m["service_name"].(string)),
		}
	}
}
