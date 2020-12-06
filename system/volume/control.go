package volume

// #cgo LDFLAGS: -lole32 -loleaut32
// #include "volume.h"
import "C"

import (
	"fmt"
	"log"
)

type Control struct {
	dryRun  bool
	isMuted bool
}

func NewVolumeControl(dryRun bool) (*Control, error) {
	return &Control{
		dryRun: dryRun,
	}, nil
}

func (c *Control) CheckMicrophoneMute() error {
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

func (c *Control) ToggleMicrophoneMute() error {
	if c.dryRun {
		return nil
	}

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
