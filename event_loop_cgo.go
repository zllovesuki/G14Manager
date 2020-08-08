// Copyright 2019 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Modification by @zllovesuki

// +build windows,use_cgo

package main

import (
	"unsafe"
)

// #include <windows.h>
//
// static int event_loop(uintptr_t handle_ptr)
// {
//     HANDLE *hwnd = (HANDLE *)handle_ptr;
//     MSG m;
//     int r;
//
//     while (*hwnd) {
//         r = GetMessage(&m, NULL, 0, 0);
//         if (!r)
//             return m.wParam;
//         else if (r < 0)
//             return -1;
//         if (!IsDialogMessage(*hwnd, &m)) {
//             TranslateMessage(&m);
//             DispatchMessage(&m);
//         }
//     }
//     return 0;
// }
import "C"

func (c *controllerl) eventLoop() int {
	return int(C.event_loop(C.uintptr_t(uintptr(unsafe.Pointer(&c.hWnd)))))
}

/*
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC_FOR_TARGET=x86_64-w64-mingw32-gcc go build -ldflags="-H=windowsgui -s -w" github.com/zllovesuki/ROGManager
*/
