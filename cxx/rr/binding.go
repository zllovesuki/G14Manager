package rr

// #cgo CXXFLAGS: -std=c++17
// #cgo LDFLAGS: -lsetupapi -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
// #include "binding.h"
import "C"
import (
	"fmt"
)

type Display struct {
}

func NewDisplayRR() (*Display, error) {
	ret := C.GetDisplay()
	if int(ret) != 1 {
		return nil, fmt.Errorf("no active display attached to integrated graphics")
	}
	return &Display{}, nil
}

func (d *Display) CycleRefreshRate() int {
	return int(C.CycleRefreshRate())
}

func (d *Display) GetCurrent() int {
	return int(C.GetCurrentRefreshRate())
}

func (d *Display) Release() {
	C.ReleaseDisplay()
}
