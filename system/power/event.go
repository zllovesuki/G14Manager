package power

import (
	"context"
	"log"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// adapted from https://golang.org/src/runtime/os_windows.go

var (
	libPowrProf                              = windows.NewLazySystemDLL("powrprof.dll")
	powerRegisterSuspendResumeNotification   = libPowrProf.NewProc("PowerRegisterSuspendResumeNotification")
	powerUnregisterSuspendResumeNotification = libPowrProf.NewProc("PowerUnregisterSuspendResumeNotification")
)

// Defines the type of event
const (
	PBT_APMSUSPEND         uint32 = 4
	PBT_APMRESUMESUSPEND   uint32 = 7
	PBT_APMRESUMEAUTOMATIC uint32 = 18
)

// NewEventListener will listen for PowerSuspendResumeNotification and send events to the channel
func NewEventListener(haltCtx context.Context, eventCh chan uint32) error {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		const (
			_DEVICE_NOTIFY_CALLBACK = 2
		)
		type _DEVICE_NOTIFY_SUBSCRIBE_PARAMETERS struct {
			callback uintptr
			context  uintptr
		}

		var fn interface{} = func(context uintptr, changeType uint32, setting uintptr) uintptr {
			eventCh <- changeType
			return 0
		}

		params := _DEVICE_NOTIFY_SUBSCRIBE_PARAMETERS{
			callback: windows.NewCallback(fn),
		}
		handle := uintptr(0)

		log.Println("power: registering suspend/resume notification")
		powerRegisterSuspendResumeNotification.Call(
			_DEVICE_NOTIFY_CALLBACK,
			uintptr(unsafe.Pointer(&params)),
			uintptr(unsafe.Pointer(&handle)),
		)

		<-haltCtx.Done()
		log.Println("power: unregistering suspend/resume notification")
		powerUnregisterSuspendResumeNotification.Call(
			uintptr(unsafe.Pointer(&handle)),
		)

	}()

	return nil
}
