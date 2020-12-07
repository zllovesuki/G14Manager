package keyboard

/*
We are looking for a device that looks something like this:
\\?\hid#vid_0b05&pid_1866&mi_02&col01#8&1e16c781&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}
Everything after \hid is just plain old PnP DeviceID, but with "\" replaced with "#"
"vid_X" where X is the vendor ID
"pid_X" where X is the product ID
"mi_X" and "colY" indicates that this is a multi-function, multi-TLC device, and we are looking for a specific column
&1e16c... is the serial number and it *should* be different on each computer
the {uuid} part is generic GUID_DEVINTERFACE_HID: https://docs.microsoft.com/en-us/windows-hardware/drivers/install/guid-devinterface-hid
*/

// #include "virtual.h"
import "C"

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/zllovesuki/G14Manager/system/device"
	"github.com/zllovesuki/G14Manager/system/ioctl"
	kb "github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/plugin"

	"github.com/karalabe/usb"
)

const (
	persistKey = "KeyboardControl"
)

const (
	brightnessControlByteIndex = 4
)

const (
	brightnessControlBufferLength     = 64
	touchPadToggleControlBufferLength = 64
	initBufferLength                  = 64
)

const (
	kbControlDevice = "mi_02&col01"
)

// TODO: reverse engineer this as well
var (
	brightnessControlBuffer = []byte{
		0x5a, 0xba, 0xc5, 0xc4,
	}
	touchPadToggleControlBuffer = []byte{
		0x5a, 0xf4, 0x6b,
	}
	// initBufs will initialize the keyboard control interface
	// for backlight control and disabling/enabling touchpad.
	// Captured via API Monitor. A lot of HidD_ functions were called
	// (which becomes DeviceIoControl)
	// also referencing https://github.com/flukejones/rog-core/blob/master/kernel-patch/0001-HID-asus-add-support-for-ASUS-N-Key-keyboard-v5.8.patch
	initBufs = [][]byte{
		{
			0x5a, 0x89,
		},
		{
			0x5a, 0x41, 0x53, 0x55, 0x53, 0x20, 0x54, 0x65,
			0x63, 0x68, 0x2e, 0x49, 0x6e, 0x63, 0x2e,
		},
		{
			0x5a, 0x05, 0x20, 0x31, 0x00, 0x08,
		},
	}
)

// Level defines the different level of keybroad brightness
type Level byte

// Brightness level
const (
	OFF    Level = 0x00
	LOW          = 0x01
	MEDIUM       = 0x02
	HIGH         = 0x03
)

// Control allows you to set the hid related functionalities directly
type Control struct {
	deviceCtrl        *device.Control
	currentBrightness Level

	queue   chan plugin.Task
	errChan chan error
}

var _ plugin.Plugin = &Control{}

// NewControl checks if the computer has the hid control interface, and returns a control interface if it does
func NewControl(dryRun bool) (*Control, error) {
	devices, err := usb.EnumerateHid(kb.VendorID, kb.ProductID)
	if err != nil {
		return nil, err
	}
	var path string
	for _, device := range devices {
		if strings.Contains(device.Path, kbControlDevice) {
			path = device.Path
		}
	}
	if path == "" {
		return nil, fmt.Errorf("kbCtrl: Keyboard control interface not found")
	}

	ctrl, err := device.NewControl(device.Config{
		DryRun:      dryRun,
		Path:        path,
		ControlCode: ioctl.HID_SET_FEATURE,
	})
	if err != nil {
		return nil, err
	}

	return &Control{
		deviceCtrl:        ctrl,
		currentBrightness: OFF,
		queue:             make(chan plugin.Task),
		errChan:           make(chan error),
	}, nil
}

// Initialize will send initialization buffer to the keyboard control device
func (c *Control) Initialize() error {
	log.Println("kbCtrl: initializaing hid interface")
	for _, buf := range initBufs {
		initBuf := make([]byte, initBufferLength)
		copy(initBuf, buf)
		_, err := c.deviceCtrl.Write(initBuf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Control) Run(haltCtx context.Context) <-chan error {
	log.Println("kbCtrl: Starting queue loop")

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		for {
			select {
			case t := <-c.queue:
				switch t.Event {
				case plugin.EvtKbReInit:
					c.errChan <- c.Initialize()
				case plugin.EvtKbBrightnessDown:
					c.errChan <- c.brightnessDown()
				case plugin.EvtKbBrightnessUp:
					c.errChan <- c.brightnessUp()
				case plugin.EvtKbBrightnessOff:
					c.errChan <- c.setBrightness(OFF)
				case plugin.EvtKbToggleTouchpad:
					c.errChan <- c.toggleTouchPad()
				case plugin.EvtKbEmulateKeyPress:
					if v, ok := t.Value.(uint16); ok {
						c.errChan <- c.emulateKeyPress(v)
					}
				}
			case <-haltCtx.Done():
				return
			}
		}
	}()

	return c.errChan
}

func (c *Control) Notify(t plugin.Task) {
	c.queue <- t
}

func (c *Control) setBrightness(v Level) error {
	inputBuf := make([]byte, brightnessControlBufferLength)
	copy(inputBuf, brightnessControlBuffer)
	inputBuf[brightnessControlByteIndex] = byte(v)

	_, err := c.deviceCtrl.Write(inputBuf)
	if err != nil {
		return err
	}

	c.currentBrightness = v

	return nil
}

func (c *Control) brightnessUp() error {
	var targetLevel Level
	switch c.currentBrightness {
	case OFF:
		targetLevel = LOW
	case LOW:
		targetLevel = MEDIUM
	case MEDIUM:
		targetLevel = HIGH
	default:
		return nil
	}
	return c.setBrightness(targetLevel)
}

func (c *Control) brightnessDown() error {
	var targetLevel Level
	switch c.currentBrightness {
	case HIGH:
		targetLevel = MEDIUM
	case MEDIUM:
		targetLevel = LOW
	case LOW:
		targetLevel = OFF
	default:
		return nil
	}
	return c.setBrightness(targetLevel)
}

func (c *Control) toggleTouchPad() error {
	inputBuf := make([]byte, touchPadToggleControlBufferLength)
	copy(inputBuf, touchPadToggleControlBuffer)

	_, err := c.deviceCtrl.Write(inputBuf)
	if err != nil {
		return err
	}

	// I don't think we have a way of checking if the touchpad is disabled/enabled

	return nil
}

func (c *Control) emulateKeyPress(keyCode uint16) error {
	if C.SendKeyPress(C.ushort(keyCode)) != 0 {
		return fmt.Errorf("kbCtrl: cannot emulate key press")
	}

	return nil
}

var _ persist.Registry = &Control{}

// Name satisfies persist.Registry
func (c *Control) Name() string {
	return persistKey
}

// Value satisfies persist.Registry
func (c *Control) Value() []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(c.currentBrightness))
	return buf
}

// Load satisfies persist.Registry
// TODO: check if the input is actually valid
func (c *Control) Load(v []byte) error {
	if len(v) == 0 {
		return nil
	}
	c.currentBrightness = Level(binary.LittleEndian.Uint16(v))
	return nil
}

// Apply satisfies persist.Registry
func (c *Control) Apply() error {
	return c.setBrightness(c.currentBrightness)
}

// Close satisfied persist.Registry
func (c *Control) Close() error {
	return c.deviceCtrl.Close()
}
