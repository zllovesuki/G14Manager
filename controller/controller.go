package controller

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"github.com/zllovesuki/ROGManager/system/persist"
	"github.com/zllovesuki/ROGManager/system/thermal"
	"github.com/zllovesuki/ROGManager/util"

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

type Controller interface {
	Run() int
}

var _ Controller = &controller{}

type Config struct {
	Thermal  *thermal.Thermal
	Registry *persist.RegistryHelper
	ROGKey   []string
}

type keyedDebounce struct {
	noisy chan<- interface{}
	clean <-chan util.DebounceEvent
}

type controller struct {
	Config
	hWnd        win.HWND
	notifyQueue chan notification
	debounce    map[int]keyedDebounce
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
		debounce:    make(map[int]keyedDebounce),
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
	switch wParam {
	case 56:
		log.Println("ROG Key Pressed (debounced)")
		c.debounce[58].noisy <- struct{}{}
	case 174:
		log.Println("Fn + F5 Pressed (debounced)")
		c.debounce[174].noisy <- struct{}{}
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

func (c *controller) setupDebounce() {
	// TODO: revisit this
	keys := []int{
		58,  // ROG Key
		174, // Fn + F5
	}
	for _, key := range keys {
		// TODO: make debounce interval configurable, maybe
		in, out := util.Debounce(time.Millisecond * 500)
		c.debounce[key] = keyedDebounce{
			noisy: in,
			clean: out,
		}
	}
	go c.handleDebounce()
}

func (c *controller) handleDebounce() {
	for {
		select {
		case ev := <-c.debounce[58].clean:
			log.Printf("ROG Key pressed %d times\n", ev.Counter)
			if int(ev.Counter) <= len(c.Config.ROGKey) {
				cmd := exec.Command("cmd.exe", "/C", c.Config.ROGKey[ev.Counter-1])
				cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
				if err := cmd.Start(); err != nil {
					log.Println(err)
				}
			}
		case ev := <-c.debounce[174].clean:
			log.Printf("Fn + F5 pressed %d times\n", ev.Counter)
			next, err := c.Config.Thermal.NextProfile(int(ev.Counter))
			message := fmt.Sprintf("Thermal plan changed to %s", next)
			if err != nil {
				log.Println(err)
				message = err.Error()
			}
			c.notifyQueue <- notification{
				title:   "Toggle Thermal Plan",
				message: message,
			}
			if err := c.Config.Registry.Save(); err != nil {
				log.Println("error saving to registry", err)
			}
		}
	}
}

func (c *controller) Run() int {

	// TODO: revisit this
	c.notifyQueue <- notification{
		title:   "Settings Loaded from Registry",
		message: fmt.Sprintf("Current Thermal Plan: %s", c.Config.Thermal.CurrentProfile().Name),
	}

	c.initialize()

	c.setupDebounce()

	return c.eventLoop()
}
