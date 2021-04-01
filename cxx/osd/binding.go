package osd

// #cgo CXXFLAGS: -std=c++17
// #cgo LDFLAGS: -lgdi32 -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
// #include <stdlib.h>
// #include "binding.h"
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

type OSD struct {
	fontSize int
	queue    chan string
}

func NewOSD(height int, width int, fSize int) (*OSD, error) {
	ret := C.NewWindow(C.int(height), C.int(width))
	if int(ret) != 1 {
		return nil, fmt.Errorf("osd: failed to initialize window")
	}
	return &OSD{
		fontSize: fSize,
		queue:    make(chan string, 5),
	}, nil
}

func (o *OSD) Show(m string, delay time.Duration) {
	msg := C.CString(m)
	C.ShowText(msg, C.int(o.fontSize))
	time.Sleep(delay)
	C.Hide()
	C.free(unsafe.Pointer(msg))
}