package matrix

// #cgo CXXFLAGS: -std=c++17 -DGO_BINDINGS
// #cgo LDFLAGS: -lsetupapi -lhid -lwinusb -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
// #include "binding.h"
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

const (
	noController        = 99
	operationSuccessful = 0
	noHandle            = 1
	inputError          = 2
	drawError           = 3
)

type Controller struct {
	mu       sync.Mutex
	pWrapper unsafe.Pointer
	freed    bool
}

func NewController() (*Controller, error) {
	var pWrapper unsafe.Pointer
	pWrapper = C.NewController()
	if pWrapper == unsafe.Pointer(nil) {
		return nil, fmt.Errorf("[matrix] Cannot obtain controller")
	}
	return &Controller{
		pWrapper: pWrapper,
		freed:    false,
	}, nil
}

func (c *Controller) Draw(buf []byte) error {
	if len(buf) != 1815 {
		return fmt.Errorf("[matrix] Invalid buffer size")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cRet := C.PrepareDraw(c.pWrapper, (*C.uchar)(unsafe.Pointer(&buf[0])), C.ulonglong(1815))
	ret := int(cRet)
	if ret != operationSuccessful {
		return fmt.Errorf("[matrix] failed to prepare draw buffer")
	}

	cRet = C.DrawMatrix(c.pWrapper)
	ret = int(cRet)
	if ret != operationSuccessful {
		return fmt.Errorf("[matrix] Draw error")
	}
	return nil
}

func (c *Controller) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cRet := C.ClearMatrix(c.pWrapper)
	ret := int(cRet)
	if ret != operationSuccessful {
		return fmt.Errorf("[matrix] Clear error")
	}
	return nil
}

func (c *Controller) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.freed {
		return
	}

	C.DeleteController(c.pWrapper)
	c.freed = true
}
