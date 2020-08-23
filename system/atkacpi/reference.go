package atkacpi

// Defines the byte index for setting behavior
const (
	hardwareControlByteIndex           = 12
	batteryChargeLimitControlByteIndex = 12
	throttlePlanControlByteIndex       = 12
	// Fan curve is a little different, DeviceControlByteIndex sets CPU/GPU, and Start Index defines the curve
	fanCurveDeviceControlByteIndex = 8
	fanCurveControlByteStartIndex  = 12
)

// Defines the buffer size when writing to ATKACPI
const (
	hardwareControlBufferLength         = 16
	batteryChargeLimitInputBufferLength = 16
	throttlePlanInputBufferLength       = 16
	fanCurveInputBufferLength           = 28
)

// Defines the buffer size when reading from ATKACPI
const (
	hardwareControlOutputBufferLength    = 4
	batteryChargeLimitOutputBufferLength = 1024
	throttlePlanOutputBufferLength       = 1024
	fanCurveOutputBufferLength           = 1024
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
	fardwareControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x21, 0x00, 0x10, 0x00, // IIA0
		0x00, 0x00, 0x00, 0x00, // IIA1
	}
	fatteryChargeLimitControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x57, 0x00, 0x12, 0x00, // IIA0
		0x00, 0x00, 0x00, 0x00, // PCI0.SBRG.EC0.SRSC (IIA1)
	}
	throttlePlanControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x75, 0x00, 0x12, 0x00, // IIA0
		0x00, 0x00, 0x00, 0x00, // Calls PCI0.SBRG.EC0.STCD according to 0, 1, or 2
	}
	fanCurveControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, // DEVS, Arg1
		// Arg2
		0x14, 0x00, 0x00, 0x00, // 20 bytes of argument
		0xFF, 0x00, 0x11, 0x00, // IIA0: 0x001100XX, where XX could be CPU (24) or GPU (25)
		// PCI0.SBRG.EC0.SUFC (IIA1, IIA2, IIA3, IIA4, 0x40/0x44)
		// (Set User Fan Curve)
		0xFF, 0xFF, 0xFF, 0xFF, // IIA1
		0xFF, 0xFF, 0xFF, 0xFF, // IIA2
		0xFF, 0xFF, 0xFF, 0xFF, // IIA3
		0xFF, 0xFF, 0xFF, 0xFF, // IIA4
	}
	initializationBuffer = []byte{
		0x49, 0x4e, 0x49, 0x54, // INIT, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0x00, 0x00, 0x00, 0x00, // IIA0, value doesn't matter
		0x00, 0x00, 0x00, 0x00, // IIA1, unused
	}

	// Unused buffer but here for reference
	getDefaultFanCurveControlBuffer = []byte{
		0x44, 0x53, 0x54, 0x53, // DSTS, Arg1
		// Arg2
		0x08, 0x00, 0x00, 0x00, // 8 bytes of argument
		0xFF, 0x00, 0x11, 0x00, // IIA0, specifies CPU (0x24) or GPU (0x25)
		0xFF, 0x00, 0x00, 0x00, // IIA1, specifies which thermal profile (0, 1, 2)
		// Output buffer is 16 bytes
	}
)
