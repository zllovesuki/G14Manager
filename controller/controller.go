package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/zllovesuki/ROGManager/system/atkacpi"
	"github.com/zllovesuki/ROGManager/system/keyboard"
	"github.com/zllovesuki/ROGManager/system/persist"
	"github.com/zllovesuki/ROGManager/system/thermal"
	"github.com/zllovesuki/ROGManager/system/volume"
	"github.com/zllovesuki/ROGManager/util"

	"gopkg.in/toast.v1"
)

const (
	appName = "ROGManager"
)

type Controller interface {
	Run(haltCtx context.Context)
}

var _ Controller = &controller{}

type Config struct {
	VolumeControl   *volume.Control
	KeyboardControl *keyboard.Control
	Thermal         *thermal.Thermal
	Registry        *persist.RegistryHelper

	ROGKey []string
}

type keyedDebounce struct {
	noisy chan<- interface{}
	clean <-chan util.DebounceEvent
}

type controller struct {
	Config

	notifyQueueCh chan notification
	debounceCh    map[uint32]keyedDebounce
	keyCodeCh     chan uint32
	wmiCh         chan uint32

	keyCtrl *atkacpi.ATKControl
}

func NewController(conf Config) (Controller, error) {
	if conf.VolumeControl == nil {
		return nil, errors.New("nil volume.Control is invalid")
	}
	if conf.KeyboardControl == nil {
		return nil, errors.New("nil keyboard.Control is invalid")
	}
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
		Config:        conf,
		notifyQueueCh: make(chan notification, 10),
		debounceCh:    make(map[uint32]keyedDebounce),
		keyCodeCh:     make(chan uint32, 1),
		wmiCh:         make(chan uint32, 1),
	}, nil
}

type notification struct {
	title   string
	message string
}

func (c *controller) notify(n notification) error {
	notification := toast.Notification{
		AppID:    appName,
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

func (c *controller) initialize(haltCtx context.Context) {

	// initialize the ATKACPI interface
	// TODO: figure out how to use go-ole to do it
	run("powershell", "-command", `"(Get-WmiObject -Namespace root/WMI -Class AsusAtkWmi_WMNB).INIT(0)"`)

	devices, err := keyboard.NewHidListener(haltCtx, c.keyCodeCh)
	if err != nil {
		log.Fatalln("error initializing hidListener", err)
	}
	log.Printf("hid devices: %+v\n", devices)

	err = atkacpi.NewWMIListener(haltCtx, c.wmiCh)
	if err != nil {
		log.Fatalln("error initializing wmiListener", err)
	}

	c.keyCtrl, err = atkacpi.NewAtkControl(atkacpi.WriteControlCode)
	if err != nil {
		log.Fatalln("error initializing atk control for keyboard emulation", err)
	}

	// TODO: revisit this
	keys := []uint32{
		58,  // ROG Key
		174, // Fn + F5
		0,   // for debouncing persisting to Registry
	}
	for _, key := range keys {
		// TODO: make debounce interval configurable, maybe
		in, out := util.Debounce(haltCtx, time.Millisecond*500)
		c.debounceCh[key] = keyedDebounce{
			noisy: in,
			clean: out,
		}
	}
}

func (c *controller) handleWMI(haltCtx context.Context) {
	for {
		select {
		case wmi := <-c.wmiCh:
			switch wmi {
			case 87:
				log.Println("wmi: On battery")
			case 123:
				log.Println("wmi: Power input changed")
			default:
				log.Printf("wmi: Unknown %d\n", wmi)
			}
		case <-haltCtx.Done():
			log.Println("Exiting handleWMI")
			return
		}
	}
}

func keyEmulation(ctrl *atkacpi.ATKControl, keyCode uint32) {
	switch keyCode {
	case
		16,  // screen brightness down
		32,  // screen brightness up
		108, // sleep
		136, // RF kill toggle
		0:   // noop
		log.Printf("Forwarding %d to ATKACPI\n", keyCode)

		inputBuf := make([]byte, atkacpi.KeyPressControlBufferLength)
		copy(inputBuf, atkacpi.KeyPressControlBuffer)
		inputBuf[atkacpi.KeyPressControlByteIndex] = byte(keyCode)

		_, err := ctrl.Write(inputBuf)
		if err != nil {
			log.Fatalln("error sending key code to ATKACPI", err)
		}
	}
}

func (c *controller) handleKeyPress(haltCtx context.Context) {
	for {
		select {
		case keyCode := <-c.keyCodeCh:
			// also forward some special key combo to the ATK interface
			go keyEmulation(c.keyCtrl, keyCode)

			switch keyCode {
			case 56:
				log.Println("hid: ROG Key Pressed (debounced)")
				c.debounceCh[58].noisy <- struct{}{}
			case 174:
				log.Println("hid: Fn + F5 Pressed (debounced)")
				c.debounceCh[174].noisy <- struct{}{}

			case 197: // keyboard brightness down (Fn + Arrow Down)
				log.Println("hid: Fn + Arrow Down Pressed")
				c.Config.KeyboardControl.BrightnessDown()
				c.debounceCh[0].noisy <- struct{}{}
			case 196: // keyboard brightness up (Fn + Arrow Up)
				log.Println("hid: Fn + Arrow Up Pressed")
				c.Config.KeyboardControl.BrightnessUp()
				c.debounceCh[0].noisy <- struct{}{}

			case 107:
				log.Println("hid: Fn + F10 Pressed")
				c.Config.KeyboardControl.ToggleTouchPad()

			case 124:
				log.Println("hid: mute/unmute micrphone Pressed")
				c.Config.VolumeControl.ToggleMicrophoneMute()

			// TODO: Handle keyboard brightness up and down via wmi
			/*
				case 32:
					// screen brightness up
				case 16:
					// screen brightness down
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
				// ...etc
			*/
			default:
				log.Printf("hid: Unknown %d\n", keyCode)
			}
		case <-haltCtx.Done():
			log.Println("Exiting handleKeyPress")
			return
		}
	}
}

func (c *controller) handleNotify(haltCtx context.Context) {
	for {
		select {
		case msg := <-c.notifyQueueCh:
			if err := c.notify(msg); err != nil {
				log.Printf("Error sending toast notification: %s\n", err)
			}
		case <-haltCtx.Done():
			log.Println("Exiting handleNotify")
			return
		}
	}
}

func (c *controller) handleDebounce(haltCtx context.Context) {
	for {
		select {

		case ev := <-c.debounceCh[58].clean:
			log.Printf("ROG Key pressed %d times\n", ev.Counter)
			if int(ev.Counter) <= len(c.Config.ROGKey) {
				if err := run("cmd.exe", "/C", c.Config.ROGKey[ev.Counter-1]); err != nil {
					log.Println(err)
				}
			}

		case ev := <-c.debounceCh[174].clean:
			log.Printf("Fn + F5 pressed %d times\n", ev.Counter)
			next, err := c.Config.Thermal.NextProfile(int(ev.Counter))
			message := fmt.Sprintf("Thermal plan changed to %s", next)
			if err != nil {
				log.Println(err)
				message = err.Error()
			}
			c.notifyQueueCh <- notification{
				title:   "Toggle Thermal Plan",
				message: message,
			}
			c.debounceCh[0].noisy <- struct{}{}

		case <-c.debounceCh[0].clean:
			if err := c.Config.Registry.Save(); err != nil {
				log.Println("error saving to registry", err)
			}

		case <-haltCtx.Done():
			log.Println("Exiting handleDebounce")
			return
		}
	}
}

func (c *controller) Run(haltCtx context.Context) {

	log.Println("Loading configuration from Registry")
	// load configs from registry and try to reapply
	if err := c.Config.Registry.Load(); err != nil {
		log.Fatalln(err)
	}
	if err := c.Config.Registry.Apply(); err != nil {
		log.Fatalln(err)
	}

	c.notifyQueueCh <- notification{
		title:   "Settings Loaded from Registry",
		message: fmt.Sprintf("Current Thermal Plan: %s", c.Config.Thermal.CurrentProfile().Name),
	}

	c.initialize(haltCtx)

	go c.handleNotify(haltCtx)
	go c.handleDebounce(haltCtx)
	go c.handleKeyPress(haltCtx)
	go c.handleWMI(haltCtx)

	<-haltCtx.Done()
	c.Config.Registry.Close()
	c.Config.VolumeControl.Close()
}

func run(commands ...string) error {
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Start()
}
