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

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/zllovesuki/ROGManager/system/device"
	"github.com/zllovesuki/ROGManager/system/persist"

	"github.com/karalabe/usb"
)

const (
	persistKey = "KeyboardBrightness"
)

const (
	writeControlCode = uint32(721297)
)

const (
	brightnessControlByteIndex = 4
)

const (
	brightnessControlBufferLength = 64
)

const (
	kbBrightnessDevice = "mi_02&col01"
)

var (
	brightnessControlBuffer = []byte{
		0x5a, 0xba, 0xc5, 0xc4, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
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

// Brightness allows you to set the keyboard brightness directly
type Brightness struct {
	deviceCtrl        *device.Control
	currentBrightness Level
}

// NewBrightnessControl checks if the computer has the brightness control interface, and returns a control interface if it does
func NewBrightnessControl() (*Brightness, error) {
	devices, err := usb.EnumerateHid(0, 0)
	if err != nil {
		return nil, err
	}
	var path string
	for _, device := range devices {
		if strings.Contains(device.Path, kbBrightnessDevice) {
			path = device.Path
		}
	}
	if path == "" {
		return nil, fmt.Errorf("Keyboard control interface not found")
	}
	// I could technically use usb.Device.Write() here
	ctrl, err := device.NewControl(path, writeControlCode)
	if err != nil {
		return nil, err
	}
	return &Brightness{
		deviceCtrl:        ctrl,
		currentBrightness: OFF,
	}, nil
}

func (b *Brightness) set(v Level) error {
	inputBuf := make([]byte, brightnessControlBufferLength)
	copy(inputBuf, brightnessControlBuffer)
	inputBuf[brightnessControlByteIndex] = byte(v)

	_, err := b.deviceCtrl.Write(inputBuf)
	if err != nil {
		return err
	}

	b.currentBrightness = v

	return nil
}

// Up increases the keyboard brightness by one level
// TODO: use a FSM
func (b *Brightness) Up() error {
	var targetLevel Level
	switch b.currentBrightness {
	case OFF:
		targetLevel = LOW
	case LOW:
		targetLevel = MEDIUM
	case MEDIUM:
		targetLevel = HIGH
	default:
		return nil
	}
	return b.set(targetLevel)
}

// Down decreases the keyboard brightness by one level
// TODO: use a FSM
func (b *Brightness) Down() error {
	var targetLevel Level
	switch b.currentBrightness {
	case HIGH:
		targetLevel = MEDIUM
	case MEDIUM:
		targetLevel = LOW
	case LOW:
		targetLevel = OFF
	default:
		return nil
	}
	return b.set(targetLevel)
}

var _ persist.Registry = &Brightness{}

// Name satisfies persist.Registry
func (b *Brightness) Name() string {
	return persistKey
}

// Value satisfies persist.Registry
func (b *Brightness) Value() []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(b.currentBrightness))
	return buf
}

// Load satisfies persist.Registry
// TODO: check if the input is actually valid
func (b *Brightness) Load(v []byte) error {
	if len(v) == 0 {
		return nil
	}
	b.currentBrightness = Level(binary.LittleEndian.Uint16(v))
	return nil
}

// Apply satisfies persist.Registry
func (b *Brightness) Apply() error {
	return b.set(b.currentBrightness)
}

// Close satisfied persist.Registry
func (b *Brightness) Close() error {
	return b.deviceCtrl.Close()
}
