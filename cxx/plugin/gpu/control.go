package gpu

// #cgo CXXFLAGS: -std=c++17
// #cgo LDFLAGS: -lsetupapi -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
// #include "device.h"
import "C"

import (
	"context"
	"fmt"
	"log"

	"github.com/zllovesuki/G14Manager/system/plugin"
	"github.com/zllovesuki/G14Manager/util"
)

type Control struct {
	dryRun  bool
	queue   chan plugin.Notification
	errChan chan error
}

var _ plugin.Plugin = &Control{}

func NewGPUControl(dryRun bool) (*Control, error) {
	return &Control{
		dryRun:  dryRun,
		queue:   make(chan plugin.Notification),
		errChan: make(chan error),
	}, nil
}

func (c *Control) RestartGPU() error {
	ret := C.restartGPU()
	if int(ret) == 0 {
		return fmt.Errorf("gpu: Cannot restart GPU")
	}
	return nil
}

func (c *Control) Initialize() error {
	return nil
}

func (c *Control) loop(haltCtx context.Context, cb chan<- plugin.Callback) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("gpu: loop panic %+v\n", err)
			c.errChan <- err.(error)
		}
	}()

	for {
		select {
		case <-c.queue:
			if c.dryRun {
				log.Println("gpu: dry run, not restarting GPU")
				continue
			}
			cb <- plugin.Callback{
				Event: plugin.CbNotifyToast,
				Value: util.Notification{
					Title:   "GPU Control",
					Message: "Restarting GPU...",
				},
			}
			err := c.RestartGPU()
			n := util.Notification{
				Title: "GPU Control",
			}
			if err != nil {
				n.Message = "Unable to restart GPU. Please check Device Manager."
				cb <- plugin.Callback{
					Event: plugin.CbNotifyToast,
					Value: n,
				}
				c.errChan <- err
				continue
			}
			n.Message = "GPU Restarted"
			cb <- plugin.Callback{
				Event: plugin.CbNotifyToast,
				Value: n,
			}
		case <-haltCtx.Done():
			log.Println("gpu: exiting Plugin run loop")
			return
		}
	}
}

func (c *Control) Run(haltCtx context.Context, cb chan<- plugin.Callback) <-chan error {
	log.Println("gpu: Starting queue loop")

	go c.loop(haltCtx, cb)

	return c.errChan
}

func (c *Control) Notify(t plugin.Notification) {
	if t.Event != plugin.EvtSentinelRestartGPU {
		return
	}

	c.queue <- t
}
