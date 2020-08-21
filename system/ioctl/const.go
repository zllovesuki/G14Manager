package ioctl

// Defines control code for write/read operations to ATKACPI
const (
	ATK_ACPI_WMIFUNCTION     = 0x22240c
	ATK_ACPI_FUNCTION        = 0x222404
	ATK_ACPI_GET_NOTIFY_CODE = 0x222408
	ATK_ACPI_ASSIGN_EVENT    = 0x222400
)

// Defines control code for HidD. http://www.ioctls.net/
const (
	HID_SET_FEATURE = 0xb0191
	HID_GET_FEATURE = 0xb0192
)
