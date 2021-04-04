package background

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/zllovesuki/G14Manager/cxx/osd"
	"github.com/zllovesuki/G14Manager/util"
)

const (
	defaultDelay = time.Millisecond * 2500
	minimumDelay = time.Millisecond * 500
	qSize        = 10
)

type Notifier struct {
	C    chan util.Notification
	show chan string
	hide chan struct{}
}

func NewNotifier() *Notifier {
	return &Notifier{
		C:    make(chan util.Notification, qSize),
		show: make(chan string, 1),
		hide: make(chan struct{}),
	}
}

func (n *Notifier) Serve(haltCtx context.Context) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Println("[notifier] starting notify loop")
	display, err := osd.NewOSD(480, 45, 24)
	if err != nil {
		log.Printf("[notifier] OSD not available: %s\n", err)
		display = nil
		// empty loop to consume channel to avoid blocking
		for {
			select {
			case <-n.C:
			case <-haltCtx.Done():
				return nil
			}
		}
	}

	go func() {
		var hideTimer <-chan time.Time
		s := make(chan util.Notification, 1)
		q := make(chan util.Notification, qSize)
		inflight := false

		for {
			select {
			case msg := <-n.C:
				if msg.Delay == time.Duration(0) {
					msg.Delay = defaultDelay
				} else if msg.Delay < minimumDelay {
					msg.Delay = minimumDelay
				}
				if msg.Immediate || !inflight {
					s <- msg
				} else {
					q <- msg
				}
			case msg := <-s:
				n.show <- msg.Message
				hideTimer = time.After(msg.Delay)
				inflight = true
			case <-hideTimer:
				n.hide <- struct{}{}
				hideTimer = nil
				inflight = false
				if len(q) > 0 {
					// amazing (/s) syntax btw
					s <- <-q
				}
			case <-haltCtx.Done():
				return
			}
		}
	}()

	for {
		select {
		case msg := <-n.show:
			display.Show(msg)
		case <-n.hide:
			display.Hide()
		case <-haltCtx.Done():
			log.Println("[notifier] existing notify loop")
			display.Release()
			return nil
		}
	}
}
