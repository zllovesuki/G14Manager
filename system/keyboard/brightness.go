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
	"errors"
	"fmt"
	"strings"

	"github.com/StackExchange/wmi"
	"github.com/zllovesuki/ROGManager/system/device"
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

// Level defines the different leval of keyboad brightness
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
	devicePath        string
	currentBrightness Level
}

// NewBrightnessControl checks if the computer has the brightness control interface, and returns a control interface if it does
func NewBrightnessControl() (*Brightness, error) {
	type Win32_PnPEntity struct {
		DeviceID string
	}
	var dst []Win32_PnPEntity
	// multi-function, multi-TLC USB interface by ASUS to control keyboard brightness
	q := wmi.CreateQuery(&dst, `WHERE DeviceID LIKE "HID\\VID_0B05&PID_1866&MI_02&COL01%"`)
	err := wmi.Query(q, &dst)
	if err != nil {
		return nil, err
	}
	if len(dst) != 1 {
		return nil, errors.New("cannot find brightness control interface")
	}
	return &Brightness{
		devicePath: fmt.Sprintf(`\\?\%s#{4d1e55b2-f16f-11cf-88cb-001111000030}`, strings.ReplaceAll(dst[0].DeviceID, "\\", "#")),
	}, nil
}

// Set will change the keyboard brightness level
func (b *Brightness) Set(v Level) error {
	inputBuf := make([]byte, brightnessControlBufferLength)
	copy(inputBuf, brightnessControlBuffer)
	inputBuf[brightnessControlByteIndex] = byte(v)

	ctrl, err := device.NewControl(b.devicePath, writeControlCode)
	if err != nil {
		return err
	}

	_, err = ctrl.Write(inputBuf)
	if err != nil {
		return err
	}

	b.currentBrightness = v

	return nil
}
