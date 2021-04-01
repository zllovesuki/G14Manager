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

const notifyDelay time.Duration = time.Second * 3

var _ plugin.Plugin = &Control{}

func NewRRControl(dryRun bool) (*Control, error) {
	return &Control{
		dryRun:   dryRun,
		pDisplay: nil,
		queue:    make(chan plugin.Notification),
		errChan:  make(chan error),
	}, nil
}

// Initialize satisfies system/plugin.Plugin
func (c *Control) Initialize() error {
	var err error
	c.pDisplay, err = rr.NewDisplayRR()
	if err != nil {
		log.Printf("rr: internal display not active/primary")
		c.pDisplay = nil
	}
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

	if c.pDisplay != nil {
		cb <- plugin.Callback{
			Event: plugin.CbNotifyToast,
			Value: util.Notification{
				Message: fmt.Sprintf("Current Refresh Rate: %d Hz", c.pDisplay.GetCurrent()),
				Delay:   notifyDelay,
			},
		}
	}

	for {
		select {
		case <-c.queue:
			if c.dryRun {
				log.Println("rr: dry run, not changing refresh rate")
				continue
			}
			n := util.Notification{
				Delay: notifyDelay,
			}
			if c.pDisplay == nil {
				// try again, in case the laptop now has internal display as primary
				c.Initialize()
				if c.pDisplay == nil {
					cb <- plugin.Callback{
						Event: plugin.CbNotifyToast,
						Value: util.Notification{
							Message: "Internal display is not primary, will not change refresh rate",
							Delay:   notifyDelay,
						},
					}
					continue
				}
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
			if c.pDisplay != nil {
				c.pDisplay.Release()
			}
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
