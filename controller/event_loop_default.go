// Copyright 2019 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Modification by @zllovesuki

// +build windows,!use_cgo

package controller

import (
	"github.com/lxn/win"
)

func (c *controller) eventLoop() int {
	msg := &win.MSG{}
	for c.hWnd != 0 {
		switch win.GetMessage(msg, 0, 0, 0) {
		case 0:
			return int(msg.WParam)

		case -1:
			return -1
		}

		if !win.IsDialogMessage(c.hWnd, msg) {
			win.TranslateMessage(msg)
			win.DispatchMessage(msg)
		}
	}

	return 0
}
