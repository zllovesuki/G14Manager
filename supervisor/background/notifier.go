package background

import (
	"context"
	"log"

	"github.com/zllovesuki/G14Manager/util"
)

type Notifier struct {
	C chan util.Notification
}

func NewNotifier() *Notifier {
	return &Notifier{
		C: make(chan util.Notification, 10),
	}
}

func (n *Notifier) Serve(haltCtx context.Context) error {
	log.Println("[notifier] starting notify loop")
	for {
		select {
		case msg := <-n.C:
			if err := util.SendToastNotification(msg); err != nil {
				log.Printf("[notifier] cannot send toast notification: %+v\n", err)
			}
		case <-haltCtx.Done():
			log.Println("[notifier] existing notify loop")
			return nil
		}
	}
}
