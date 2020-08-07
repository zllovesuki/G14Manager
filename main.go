package main

import (
	"fmt"
	"log"
	"os/exec"
	"unsafe" // what is type safety anyway ¯\_(ツ)_/¯

	"github.com/zllovesuki/ROGManager/system/thermal"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"gopkg.in/toast.v1"
)

const (
	lpString   = "ACPI Notification through ATKHotkey from BIOS"
	className  = "ROGManager"
	windowName = "ROGManager"
)

var (
	pClassName  = windows.StringToUTF16Ptr(className)
	pWindowName = windows.StringToUTF16Ptr(windowName)
	pAPCI       = win.RegisterWindowMessage(windows.StringToUTF16Ptr(lpString))
)

// If you just want it to launch whatever program, change this
var commandWithArgs = []string{"Taskmgr.exe"}

type controller struct {
	thermal *thermal.Thermal
}

func (c *controller) handleSystemControlInterface(wParam uintptr) {
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
	case 174:
		log.Println("Fn + F5 Pressed")
		next, err := c.thermal.NextProfile()
		notification := toast.Notification{
			AppID:    className,
			Title:    "Toggle Thermal Plan",
			Message:  fmt.Sprintf("Thermal plan changed to %s", next),
			Duration: toast.Short,
			Audio:    "silent",
		}
		if err != nil {
			notification.Message = err.Error()
		}
		err = notification.Push()
		if err != nil {
			log.Println(err)
		}
	/*
		case 87:
			// Not sure what this is
		case 107:
			// Fn + F10: disable/enable trackpad
		case 123:
			// Power input change (unplug/plug in)
		case 124:
			// microphone mute/unmute
		case 196:
			// brightness up
		case 197:
			// brightness down
	*/
	default:
		log.Printf("Unknown keypress: %d\n", wParam)
	}
}

func (c *controller) wndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
	case pAPCI:
		c.handleSystemControlInterface(wParam)
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (c *controller) Run() {
	wc := win.WNDCLASSEX{}

	hInst := win.GetModuleHandle(nil)
	if hInst == 0 {
		log.Fatal("cannot acquire instance")
	}

	wc.LpfnWndProc = windows.NewCallback(c.wndProc)
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

func main() {
	profile, err := thermal.NewThermal()
	if err != nil {
		log.Fatalln(err)
	}
	profile.Default()

	// TODO: consider persistent states to Registry
	control := &controller{
		thermal: profile,
	}

	// TODO: consider adding signal handling for safe shutdown
	control.Run()
}
