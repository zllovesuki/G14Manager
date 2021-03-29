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
	ret := C.disableGPU()
	if int(ret) == 0 {
		return fmt.Errorf("gpu: Cannot disable GPU")
	}
	ret = C.enableGPU()
	if int(ret) == 0 {
		return fmt.Errorf("gpu: Cannot re-enable GPU")
	}
	return nil
}

func (c *Control) DisableGPU() error {
	ret := C.disableGPU()
	switch int(ret) {
	case 0:
		return fmt.Errorf("gpu: Cannot disable GPU")
	case 2:
		return fmt.Errorf("gpu: GPU is already disabled")
	default:
		return nil
	}
}

func (c *Control) EnableGPU() error {
	ret := C.enableGPU()
	switch int(ret) {
	case 0:
		return fmt.Errorf("gpu: Cannot enable GPU")
	case 2:
		return fmt.Errorf("gpu: GPU is already enabled")
	default:
		return nil
	}
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
		case evt := <-c.queue:
			if c.dryRun {
				log.Println("gpu: dry run, not controlling GPU state")
				continue
			}
			var action string
			var err error
			switch evt.Event {
			case plugin.EvtSentinelEnableGPU:
				action = "enable"
				err = c.EnableGPU()
			case plugin.EvtSentinelDisableGPU:
				action = "disable"
				err = c.DisableGPU()
			}
			cb <- plugin.Callback{
				Event: plugin.CbNotifyToast,
				Value: util.Notification{
					Title:   "GPU Control",
					Message: fmt.Sprintf("Attempting to %s GPU...", action),
				},
			}
			n := util.Notification{
				Title: "GPU Control",
			}
			if err != nil {
				n.Message = fmt.Sprintf("Unable to %s GPU. Please check log for more details", action)
				cb <- plugin.Callback{
					Event: plugin.CbNotifyToast,
					Value: n,
				}
				c.errChan <- err
				continue
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
	if t.Event != plugin.EvtSentinelEnableGPU && t.Event != plugin.EvtSentinelDisableGPU {
		return
	}

	c.queue <- t
}
