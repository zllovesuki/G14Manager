package atkacpi

import (
	"github.com/zllovesuki/G14Manager/system/device"
)

// Defines the byte index for setting behavior
const (
	KeyPressControlByteIndex           = 12
	BatteryChargeLimitControlByteIndex = 12
	ThrottlePlanControlByteIndex       = 12
	// Fan curve is a little different, DeviceControlByteIndex sets CPU/GPU, and Start Index defines the curve
	FanCurveDeviceControlByteIndex = 8
	FanCurveControlByteStartIndex  = 12
)

// Defines the buffer size when writing to ATKACPI
const (
	KeyPressControlBufferLength         = 16
	BatteryChargeLimitInputBufferLength = 16
	ThrottlePlanInputBufferLength       = 16
	FanCurveInputBufferLength           = 28
)

// Defines the buffer size when reading from ATKACPI
const (
	KeyPressControlOutputBufferLength    = 4
	BatteryChargeLimitOutputBufferLength = 1024
	ThrottlePlanOutputBufferLength       = 1024
	FanCurveOutputBufferLength           = 1024
)

// Defines the template control buffer. Note: You must not change this and must copy() to a new []byte
var (
	KeyPressControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, 0x08, 0x00, 0x00, 0x00,
		0x21, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	BatteryChargeLimitControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, 0x08, 0x00, 0x00, 0x00,
		0x57, 0x00, 0x12, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	ThrottlePlanControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, 0x08, 0x00, 0x00, 0x00,
		0x75, 0x00, 0x12, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	FanCurveControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, 0x14, 0x00, 0x00,
		0x00, 0xFF, 0x00, 0x11, 0x00, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
)

const devicePath = `\\.\ATKACPI`

type ATKControl struct {
	device *device.Control
}

func NewAtkControl(controlCode uint32) (*ATKControl, error) {
	device, err := device.NewControl(devicePath, controlCode)
	if err != nil {
		return nil, err
	}
	return &ATKControl{
		device: device,
	}, nil
}

func (a *ATKControl) Write(buf []byte) (result int, err error) {
	result, err = a.device.Write(buf)
	if err != nil {
		return
	}
	return
}

func (a *ATKControl) Read(buf []byte) (result int, err error) {
	result, err = a.device.Read(buf)
	if err != nil {
		return
	}
	return
}

func (a *ATKControl) Close() error {
	return a.device.Close()
}
