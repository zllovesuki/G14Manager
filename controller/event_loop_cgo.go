// Copyright 2019 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Modification by @zllovesuki

// +build windows,use_cgo

package controller

import (
	"fmt"
	"unsafe"
)

// #include "event_loop.h"
import "C"

func (c *controller) eventLoop() int {
	fmt.Println("Using C event_loop")
	return int(C.event_loop(C.uintptr_t(uintptr(unsafe.Pointer(&c.hWnd)))))
}

/*
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC_FOR_TARGET=x86_64-w64-mingw32-gcc go build -ldflags="-H=windowsgui -s -w" github.com/zllovesuki/ROGManager
*/
