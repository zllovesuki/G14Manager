package volume

// #cgo LDFLAGS: -lole32 -loleaut32
// #include "volume.h"
import "C"

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/plugin"
)

type Control struct {
	dryRun  bool
	isMuted bool
	mu      sync.Mutex

	queue   chan plugin.Notification
	errChan chan error
}

var _ plugin.Plugin = &Control{}

// NewVolumeControl returns a controller for toggling default input device's muted status.
// The controller is safe for multiple goroutines.
func NewVolumeControl(dryRun bool) (*Control, error) {
	return &Control{
		dryRun:  dryRun,
		queue:   make(chan plugin.Notification),
		errChan: make(chan error),
	}, nil
}

// Initialize satisfies system/plugin.Plugin
func (c *Control) Initialize() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return c.doCheckMute()
}

func (c *Control) loop(haltCtx context.Context, cb chan<- plugin.Callback) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("volCtrl: loop panic %+v\n", err)
			c.errChan <- err.(error)
		}
	}()

	for {
		select {
		case t := <-c.queue:
			keycode, ok := t.Value.(uint32)
			if !ok {
				continue
			}
			switch keycode {
			case keyboard.KeyMuteMic:
				c.errChan <- c.ToggleMuted()
			}
		case <-haltCtx.Done():
			return
		}
	}
}

// Run satisfies system/plugin.Plugin
func (c *Control) Run(haltCtx context.Context, cb chan<- plugin.Callback) <-chan error {
	log.Println("volCtrl: Starting queue loop")

	go c.loop(haltCtx, cb)

	return c.errChan
}

// Notify satisfies system/plugin.Plugin
func (c *Control) Notify(t plugin.Notification) {
	if c.dryRun {
		return
	}

	if t.Event != plugin.EvtKeyboardFn {
		return
	}

	c.queue <- t
}

func (c *Control) doCheckMute() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ret := C.SetMicrophoneMute(1, 0)
	switch ret {
	case -1:
		return fmt.Errorf("Cannot check microphone muted status")
	default:
		c.isMuted = ret == 1
		log.Printf("wca: current microphone mute is %v\n", c.isMuted)
		return nil
	}
}

// CheckMuted returns the default recording device's muted status
func (c *Control) CheckMuted() (bool, error) {
	if err := c.doCheckMute(); err != nil {
		return false, err
	}
	return c.isMuted, nil
}

// ToggleMuted toggles the default recording device's muted status.
// Note: This should be called after CheckMuted() has been called once.
func (c *Control) ToggleMuted() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
