package atkacpi

import (
	"github.com/zllovesuki/G14Manager/system/device"
)

// Defines the byte index for setting behavior
const (
	HardwareControlByteIndex           = 12
	BatteryChargeLimitControlByteIndex = 12
	ThrottlePlanControlByteIndex       = 12
	// Fan curve is a little different, DeviceControlByteIndex sets CPU/GPU, and Start Index defines the curve
	FanCurveDeviceControlByteIndex = 8
	FanCurveControlByteStartIndex  = 12
)

// Defines the buffer size when writing to ATKACPI
const (
	HardwareControlBufferLength         = 16
	BatteryChargeLimitInputBufferLength = 16
	ThrottlePlanInputBufferLength       = 16
	FanCurveInputBufferLength           = 28
)

// Defines the buffer size when reading from ATKACPI
const (
	HardwareControlOutputBufferLength    = 4
	BatteryChargeLimitOutputBufferLength = 1024
	ThrottlePlanOutputBufferLength       = 1024
	FanCurveOutputBufferLength           = 1024
)

// Defines the template control buffer. Note: You must not change this and must copy() to a new []byte
// These buffers will be used to instruct atkwmiacpi64.sys to invoke WMI functions, the control code is IOCTL_ATK_ACPI_WMIFUNCTION.
// WMI method for setting device is DEVS (Stands for DEVice Set)
// (for adventure of WMI, see reverse_eng/wmi.txt)
// Unfortunately, DEVS only announces itself having 2 paremeters in WMI (g14-dsdt.dsl),
// So we cannot control the fan curve via WMI, and have to invoke ACPI method (which we cannot do from userspace).
// However, atkwmiacpi64.sys will be our bridge to success.
// The ID for DEVS is 0x53564544, and because of endianess difference, they are reversed in the buffer template in the first 4 bytes.
// Length of argument is in 4th-7th bytes
// Remaining buffer is argument
// TODO: Refactor this into a helper function
var (
	HardwareControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x21, 0x00, 0x10, 0x00, // IIA0
		0x00, 0x00, 0x00, 0x00, // IIA1
	}
	BatteryChargeLimitControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x57, 0x00, 0x12, 0x00, // IIA0
		0x00, 0x00, 0x00, 0x00, // PCI0.SBRG.EC0.SRSC (IIA1)
	}
	ThrottlePlanControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x75, 0x00, 0x12, 0x00, // IIA0
		0x00, 0x00, 0x00, 0x00, // Calls PCI0.SBRG.EC0.STCD according to 0, 1, or 2
	}
	FanCurveControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x14, 0x00, 0x00, 0x00, // 20 bytes of argument
		0xFF, 0x00, 0x11, 0x00, // IIA0: 0x001100XX, where XX could be CPU (24) or GPU (25)
		// PCI0.SBRG.EC0.SUFC (IIA1, IIA2, IIA3, IIA4, 0x40/0x44)
		0xFF, 0xFF, 0xFF, 0xFF, // IIA1
		0xFF, 0xFF, 0xFF, 0xFF, // IIA2
		0xFF, 0xFF, 0xFF, 0xFF, // IIA3
		0xFF, 0xFF, 0xFF, 0xFF, // IIA4
	}
	InitializationBuffer = []byte{
		0x49, 0x4e, 0x49, 0x54, // INIT, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x00, 0x00, 0x00, 0x00, // IIA0, value doesn't matter
		0x00, 0x00, 0x00, 0x00, // IIA1, unused
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
