package controller

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"github.com/zllovesuki/ROGManager/system/persist"
	"github.com/zllovesuki/ROGManager/system/thermal"
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

type Controller interface {
	Run() int
	Shutdown()
}

var _ Controller = &controller{}

type Config struct {
	Thermal  *thermal.Thermal
	Registry *persist.RegistryHelper
	ROGKey   []string // TODO: make this an interface for key remapping
}

type controller struct {
	Config
	hWnd        win.HWND
	notifyQueue chan notification
}

func NewController(conf Config) (Controller, error) {
	if conf.Thermal == nil {
		return nil, errors.New("nil Thermal is invalid")
	}
	if conf.Registry == nil {
		return nil, errors.New("nil Registry is invalid")
	}
	if len(conf.ROGKey) == 0 {
		return nil, errors.New("empty key remap is invalid")
	}
	return &controller{
		Config:      conf,
		notifyQueue: make(chan notification, 10),
	}, nil
}

type notification struct {
	title   string
	message string
}

func (c *controller) notify(n notification) error {
	notification := toast.Notification{
		AppID:    className,
		Title:    n.title,
		Message:  n.message,
		Duration: toast.Short,
		Audio:    "silent",
	}
	if err := notification.Push(); err != nil {
		return err
	}
	return nil
}

func (c *controller) handleSystemControlInterface(wParam uintptr) {
	// received all the control key presses (e.g. volume up, down, etc)
	// we are only interested in "56", which is the ROG key
	switch wParam {
	case 56:
		// ROG Key pressed
		log.Println("ROG Key Pressed")
		cmd := exec.Command(c.Config.ROGKey[0], c.Config.ROGKey[1:]...)
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
	case 174:
		log.Println("Fn + F5 Pressed")

		next, err := c.Config.Thermal.NextProfile()
		message := fmt.Sprintf("Thermal plan changed to %s", next)
		if err != nil {
			log.Println(err)
			message = err.Error()
		}
		c.notifyQueue <- notification{
			title:   "Toggle Thermal Plan",
			message: message,
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
		c.Shutdown()
	case pAPCI:
		c.handleSystemControlInterface(wParam)
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (c *controller) initialize() {
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

	c.hWnd = win.CreateWindowEx(
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
	if c.hWnd == 0 {
		log.Fatal("cannot create window", win.GetLastError())
	}

	win.ChangeWindowMessageFilterEx(
		c.hWnd,
		pAPCI,
		win.MSGFLT_ALLOW,
		nil,
	)

	win.ShowWindow(c.hWnd, win.SW_HIDE)

	// only now we can send toast notifications
	go func() {
		for {
			select {
			case msg := <-c.notifyQueue:
				log.Println("Sending toast notification")
				if err := c.notify(msg); err != nil {
					log.Printf("Error sending toast notification: %s\n", err)
				}
			}
		}
	}()
}

func (c *controller) Run() int {

	// TODO: revisit this
	c.notifyQueue <- notification{
		title:   "Settings Loaded from Registry",
		message: fmt.Sprintf("Current Thermal Plan: %s", c.Config.Thermal.CurrentProfile().Name),
	}

	c.initialize()

	return c.eventLoop()
}

func (c *controller) Shutdown() {
	// TODO: revisit this
	c.notifyQueue <- notification{
		title:   "Saving Settings to Registry",
		message: fmt.Sprintf("Thermal Plan: %s", c.Config.Thermal.CurrentProfile().Name),
	}
	time.Sleep(time.Millisecond * 50)

	if err := c.Config.Registry.Save(); err != nil {
		log.Fatalln(err)
	}
	os.Exit(0)
}
