package osd

// #cgo CXXFLAGS: -std=c++17
// #cgo LDFLAGS: -lgdi32 -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
// #include "binding.h"
import "C"
import (
	"fmt"
	"unsafe"
)

type OSD struct {
	fontSize int
	cache    map[string]*C.char
}

// NewOSD is not goroutine safe. This should be instantiated from a single OS thread
func NewOSD(height int, width int, fSize int) (*OSD, error) {
	ret := C.NewWindow(C.int(height), C.int(width))
	if int(ret) != 1 {
		return nil, fmt.Errorf("osd: failed to initialize window")
	}
	return &OSD{
		fontSize: fSize,
		cache:    make(map[string]*C.char),
	}, nil
}

func (o *OSD) Show(m string) {
	if _, ok := o.cache[m]; !ok {
		o.cache[m] = C.CString(m)
	}
	C.ShowText(o.cache[m], C.int(o.fontSize))
}

func (o *OSD) Hide() {
	C.Hide()
}

func (o *OSD) Release() {
	for _, v := range o.cache {
		C.free(unsafe.Pointer(v))
	}
}
