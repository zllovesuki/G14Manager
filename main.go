package main

import (
	"log"
	"os/exec"
	"unsafe" // what is type safety anyway ¯\_(ツ)_/¯

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	lpString   = "ACPI Notification through ATKHotkey from BIOS"
	className  = "ROGKeyRebind"
	windowName = "ROGKeyRebind"
)

var (
	pClassName  = windows.StringToUTF16Ptr(className)
	pWindowName = windows.StringToUTF16Ptr(windowName)
	pAPCI       = win.RegisterWindowMessage(windows.StringToUTF16Ptr(lpString))
)

// If you just want it to launch whatever program, change this
var commandWithArgs = []string{"Taskmgr.exe"}

func handleSystemControlInterface(wParam uintptr) {
	// received all the control key presses (e.g. volume up, down, etc)
	// we are only interested in "56", which is the ROG key
	switch wParam {
	case 56:
		// ROG Key pressed
		log.Println("ROG Key Pressed")
		cmd := exec.Command(commandWithArgs[0], commandWithArgs[1:]...)
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
	/*
		case 107:
			// Fn + F10: disable/enable trackpad
		case 124:
			// microphone mute/unmute
		case 174:
			// Fn + F5: change profile
		case 196:
			// brightness up
		case 197:
			// brightness down
	*/
	default:
		log.Printf("Unknown keypress: %d\n", wParam)
	}
}

// Below is uninteresting even to developers

func wndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
	case pAPCI:
		handleSystemControlInterface(wParam)
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func main() {

	wc := win.WNDCLASSEX{}

	hInst := win.GetModuleHandle(nil)
	if hInst == 0 {
		log.Fatal("cannot acquire instance")
	}

	wc.LpfnWndProc = windows.NewCallback(wndProc)
	wc.HInstance = hInst
	wc.LpszClassName = pClassName
	wc.CbSize = uint32(unsafe.Sizeof(wc))

	if atom := win.RegisterClassEx(&wc); atom == 0 {
		log.Fatal("cannot register class")
	}

	hwnd := win.CreateWindowEx(
		0,
		wc.LpszClassName,
		pWindowName,
		win.WS_VISIBLE,
		-1, -1, 1, 1,
		0,
		0,
		hInst,
		nil,
	)
	if hwnd == 0 {
		log.Fatal("cannot create window", win.GetLastError())
	}

	win.ChangeWindowMessageFilterEx(
		hwnd,
		pAPCI,
		win.MSGFLT_ALLOW,
		nil,
	)

	win.ShowWindow(hwnd, win.SW_HIDE)
	msg := win.MSG{}
	for win.GetMessage(&msg, 0, 0, 0) != 0 {
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
}
