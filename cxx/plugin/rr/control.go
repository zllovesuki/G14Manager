package rr

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/zllovesuki/G14Manager/cxx/rr"
	"github.com/zllovesuki/G14Manager/system/plugin"
	"github.com/zllovesuki/G14Manager/util"
)

type Control struct {
	dryRun   bool
	pDisplay *rr.Display

	queue   chan plugin.Notification
	errChan chan error
}

var _ plugin.Plugin = &Control{}

func NewRRControl(dryRun bool) (*Control, error) {
	display, err := rr.NewDisplayRR()
	if err != nil {
		return nil, err
	}
	return &Control{
		dryRun:   dryRun,
		pDisplay: display,
		queue:    make(chan plugin.Notification),
		errChan:  make(chan error),
	}, nil
}

// Initialize satisfies system/plugin.Plugin
func (c *Control) Initialize() error {
	return nil
}

func (c *Control) loop(haltCtx context.Context, cb chan<- plugin.Callback) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("rr: loop panic %+v\n", err)
			c.errChan <- err.(error)
		}
	}()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cb <- plugin.Callback{
		Event: plugin.CbNotifyToast,
		Value: util.Notification{
			Message: fmt.Sprintf("Current Refresh Rate: %d Hz", c.pDisplay.GetCurrent()),
			Delay:   time.Millisecond * 3000,
		},
	}

	for {
		select {
		case <-c.queue:
			if c.dryRun {
				log.Println("rr: dry run, not changing refresh rate")
				continue
			}
			n := util.Notification{
				Delay: time.Millisecond * 3000,
			}
			ret := c.pDisplay.CycleRefreshRate()
			log.Printf("rr: ret value %d\n", ret)
			if ret == 0 {
				n.Message = "Unable to change refresh rate"
			} else {
				n.Message = fmt.Sprintf("Refresh Rate changed to %d Hz", ret)
			}
			cb <- plugin.Callback{
				Event: plugin.CbNotifyToast,
				Value: n,
			}
		case <-haltCtx.Done():
			log.Println("rr: exiting Plugin run loop")
			return
		}
	}
}

// Run satisfies system/plugin.Plugin
func (c *Control) Run(haltCtx context.Context, cb chan<- plugin.Callback) <-chan error {
	log.Println("rr: Starting queue loop")

	go c.loop(haltCtx, cb)

	return c.errChan
}

// Notify satisfies system/plugin.Plugin
func (c *Control) Notify(t plugin.Notification) {
	if c.dryRun {
		return
	}

	if t.Event != plugin.EvtSentinelCycleRefreshRate {
		return
	}

	c.queue <- t
}
