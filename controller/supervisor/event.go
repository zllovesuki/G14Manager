package supervisor

import (
	"fmt"
	"log"

	"github.com/zllovesuki/G14Manager/box"
	"github.com/zllovesuki/G14Manager/system/shared"
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
			AppName: shared.AppName,
			Icon:    box.GetAssetExtractor().Get("/dead.png"),
			Title:   fmt.Sprintf("%s crashed unexpectedly", m["service_name"].(string)),
			Message: "The service will be restarted",
		}
	}
}
