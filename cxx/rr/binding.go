package rr

// #cgo CXXFLAGS: -std=c++17
// #cgo LDFLAGS: -lsetupapi -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
// #include "binding.h"
import "C"
import (
	"fmt"
	"unsafe"
)

type Display struct {
	pDisplay unsafe.Pointer
}

func NewDisplayRR() (*Display, error) {
	pDisplay := C.GetDisplay()
	if pDisplay == nil {
		return nil, fmt.Errorf("No active display attached to integrated graphics")
	}
	return &Display{
		pDisplay: pDisplay,
	}, nil
}

func (d *Display) CycleRefreshRate() int {
	return int(C.CycleRefreshRate(d.pDisplay))
}

func (d *Display) GetCurrent() int {
	return int(C.GetCurrentRefreshRate(d.pDisplay))
}
