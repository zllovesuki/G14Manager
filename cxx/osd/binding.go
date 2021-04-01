package osd

// #cgo CXXFLAGS: -std=c++17
// #cgo LDFLAGS: -lgdi32 -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
// #include <stdlib.h>
// #include "binding.h"
import "C"
import (
	"time"
	"unsafe"
)

type OSD struct {
	pWindow  unsafe.Pointer
	fontSize int
	queue    chan string
}

func NewOSD(height int, width int, fSize int) (*OSD, error) {
	return &OSD{
		pWindow:  C.NewWindow(C.int(height), C.int(width)),
		fontSize: fSize,
		queue:    make(chan string, 5),
	}, nil
}

func (o *OSD) Show(m string, delay time.Duration) {
	msg := C.CString(m)
	C.ShowText(o.pWindow, msg, C.int(o.fontSize))
	time.Sleep(delay)
	C.Hide(o.pWindow)
	C.free(unsafe.Pointer(msg))
}