package power

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// adapted from https://golang.org/src/runtime/os_windows.go

var (
	libPowrProf                            = windows.NewLazySystemDLL("powrprof.dll")
	powerRegisterSuspendResumeNotification = libPowrProf.NewProc("PowerRegisterSuspendResumeNotification")
)

// Defines the type of event
const (
	PBT_APMSUSPEND         uint32 = 4
	PBT_APMRESUMESUSPEND   uint32 = 7
	PBT_APMRESUMEAUTOMATIC uint32 = 18
)

// NewEventListener will listen for PowerSuspendResumeNotification and send events to the channel
func NewEventListener(eventCh chan uint32) error {
	const (
		_DEVICE_NOTIFY_CALLBACK = 2
	)
	type _DEVICE_NOTIFY_SUBSCRIBE_PARAMETERS struct {
		callback uintptr
		context  uintptr
	}

	// TODO: investgiate if this is safe to run in goroutines
	var fn interface{} = func(context uintptr, changeType uint32, setting uintptr) uintptr {
		eventCh <- changeType
		return 0
	}

	params := _DEVICE_NOTIFY_SUBSCRIBE_PARAMETERS{
		callback: windows.NewCallback(fn),
	}
	handle := uintptr(0)
	ret, _, err := powerRegisterSuspendResumeNotification.Call(
		_DEVICE_NOTIFY_CALLBACK,
		uintptr(unsafe.Pointer(&params)),
		uintptr(unsafe.Pointer(&handle)),
	)
	if ret != 0 {
		return err
	}
	return nil
}
