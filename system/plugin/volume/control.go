package volume

// #cgo LDFLAGS: -lole32 -loleaut32
// #include "volume.h"
import "C"

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/zllovesuki/G14Manager/system/plugin"
)

type Control struct {
	dryRun  bool
	isMuted bool

	queue   chan plugin.Task
	errChan chan error
}

var _ plugin.Plugin = &Control{}

func NewVolumeControl(dryRun bool) (*Control, error) {
	return &Control{
		dryRun:  dryRun,
		queue:   make(chan plugin.Task),
		errChan: make(chan error),
	}, nil
}

func (c *Control) Initialize() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return c.doCheckMute()
}

func (c *Control) Run(haltCtx context.Context) <-chan error {
	log.Println("volCtrl: Starting queue loop")

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		for {
			select {
			case t := <-c.queue:
				switch t.Event {
				case plugin.EvtVolToggleMute:
					c.errChan <- c.doToggleMute()
				}
			case <-haltCtx.Done():
				return
			}
		}
	}()

	return c.errChan
}

func (c *Control) Notify(t plugin.Task) {
	if c.dryRun {
		return
	}

	c.queue <- t
}

func (c *Control) doCheckMute() error {
	ret := C.SetMicrophoneMute(1, 0)
	switch ret {
	case -1:
		return fmt.Errorf("Cannot check microphone muted status")
	default:
		c.isMuted = ret == 0
		log.Printf("wca: current microphone mute is %v\n", c.isMuted)
		return nil
	}
}

func (c *Control) doToggleMute() error {
	var to int
	if c.isMuted {
		to = 1
	}
	log.Printf("wca: setting microphone mute to %t\n", c.isMuted)
	ret := C.SetMicrophoneMute(0, C.int(to))
	switch ret {
	case -1:
		return fmt.Errorf("Cannot set microphone muted status")
	default:
		c.isMuted = !c.isMuted
		return nil
	}
}
