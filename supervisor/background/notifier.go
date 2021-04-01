package background

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/zllovesuki/G14Manager/cxx/osd"
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
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Println("[notifier] starting notify loop")
	display, err := osd.NewOSD(450, 50, 22)
	if err != nil {
		log.Printf("[notifier] OSD not available: %s\n", err)
		display = nil
	}

	for {
		select {
		case msg := <-n.C:
			if msg.Delay == time.Duration(0) {
				msg.Delay = time.Millisecond * 2500
			}
			if display == nil {
				continue
			}
			display.Show(msg.Message, msg.Delay)
		case <-haltCtx.Done():
			log.Println("[notifier] existing notify loop")
			return nil
		}
	}
}
