package controller

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/system/volume"
	"github.com/zllovesuki/G14Manager/util"

	"gopkg.in/toast.v1"
)

const (
	appName = "G14Manager"
)

const (
	fnPersistConfigs = iota // for debouncing persisting to Registry
	fnCheckCharger          // for debouncing power input change acpi event
	fnApplyConfigs          // for loading and re-applying configurations
	fnKbCtrl                // for controlling keyboard behaviors
	fnToggleTouchPad        // for toggling touchpad enable/disable
	fnVolCtrl               // for mute/unmute microphone
	fnHwCtrl                // for notifying atkacpi
	fnBeforeSuspend         // for doing work before suspend
)

type Config struct {
	EnableExperimental bool

	VolumeControl   *volume.Control
	KeyboardControl *keyboard.Control
	Thermal         *thermal.Control
	Registry        *persist.RegistryHelper

	ROGKey []string
}

type workQueue struct {
	noisy chan<- interface{}
	clean <-chan util.DebounceEvent
}

type Controller struct {
	Config

	notifyQueueCh chan notification

	workQueueCh map[uint32]workQueue

	keyCodeCh chan uint32
	acpiCh    chan uint32
	powerEvCh chan uint32

	wmi      atkacpi.WMI
	isDryRun bool
}

func NewController(conf Config) (*Controller, error) {
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
	return &Controller{
		Config: conf,

		notifyQueueCh: make(chan notification, 10),
		workQueueCh:   make(map[uint32]workQueue),

		keyCodeCh: make(chan uint32, 1),
		acpiCh:    make(chan uint32, 1),
		powerEvCh: make(chan uint32, 1),

		isDryRun: os.Getenv("DRY_RUN") != "",
	}, nil
}

func (c *Controller) initialize(haltCtx context.Context) {
	// Do we need to lock os thread on any of these?

	devices, err := keyboard.NewHidListener(haltCtx, c.keyCodeCh)
	if err != nil {
		log.Fatalln("controller: error initializing hid listener", err)
	}
	log.Printf("hid devices: %+v\n", devices)

	// This is a bit buggy, as Windows seems to time out our connection to WMI
	// TODO: Find a better way os listening from atk wmi events
	err = atkacpi.NewACPIListener(haltCtx, c.acpiCh)
	if err != nil {
		log.Fatalln("controller: error initializing atkacpi wmi listener", err)
	}

	err = power.NewEventListener(c.powerEvCh)
	if err != nil {
		log.Fatalln("controller: error initializing power event listener", err)
	}

	c.wmi, err = atkacpi.NewWMI()
	if err != nil {
		log.Fatalln("controller: error initializing atk wmi interface", err)
	}

	// initialize the ATKACPI interface
	log.Printf("controller: initializing ATKD for acpi events")
	initBuf := make([]byte, 4)
	if _, err := c.wmi.Evaluate(atkacpi.INIT, initBuf); err != nil {
		log.Fatalln("controller: cannot initialize ATKD")
	}

	debounceKeys := []uint32{
		// TODO: define these as constants
		58,  // ROG Key
		174, // Fn + F5
	}
	for _, key := range debounceKeys {
		// TODO: make debounce interval configurable for accessbility
		in, out := util.Debounce(haltCtx, time.Millisecond*500)
		c.workQueueCh[key] = workQueue{
			noisy: in,
			clean: out,
		}
	}

	workQueueImmediate := []uint32{
		fnCheckCharger,
		fnApplyConfigs,
		fnToggleTouchPad,
		fnKbCtrl,
		fnVolCtrl,
		fnHwCtrl,
		fnBeforeSuspend,
	}
	for _, work := range workQueueImmediate {
		in, out := util.PassThrough(haltCtx)
		c.workQueueCh[work] = workQueue{
			noisy: in,
			clean: out,
		}
	}

	workQueueDebounced := []uint32{
		fnPersistConfigs,
	}
	for _, work := range workQueueDebounced {
		in, out := util.Debounce(haltCtx, time.Millisecond*1000)
		c.workQueueCh[work] = workQueue{
			noisy: in,
			clean: out,
		}
	}

	// seed the channel so we get the the charger status
	c.workQueueCh[fnCheckCharger].noisy <- struct{}{}
	// load and apply configurations
	c.workQueueCh[fnApplyConfigs].noisy <- struct{}{}
}

func (c *Controller) handleACPINotification(haltCtx context.Context) {
	for {
		select {
		case acpi := <-c.acpiCh:
			switch acpi {
			case 87, 88, 207: // ignore these events
				continue
			/*case 87:
				log.Println("acpi: On battery")
			case 88:
				log.Println("acpi: On AC power")
				// this is when you plug in the 180W charger
				// However, plugging in the USB C PD will not show 88 (might need to detect it in user space)
				// there's also this mysterious 207 code, that only pops up when 180W charger is plugged in/unplugged*/
			case 123:
				log.Println("acpi: Power input changed")
				c.workQueueCh[fnCheckCharger].noisy <- struct{}{}
			case 233:
				log.Println("acpi: On Lid Open/Close")
			default:
				log.Printf("acpi: Unknown %d\n", acpi)
			}
		case <-haltCtx.Done():
			log.Println("controller: exiting handleACPINotification")
			return
		}
	}
}

func (c *Controller) handlePowerEvent(haltCtx context.Context) {
	for {
		select {
		case ev := <-c.powerEvCh:
			switch ev {
			case power.PBT_APMRESUMESUSPEND:
				// ignore this event
			case power.PBT_APMSUSPEND:
				log.Println("controller: housekeeping before suspend")
				c.workQueueCh[fnBeforeSuspend].noisy <- struct{}{}
			case power.PBT_APMRESUMEAUTOMATIC:
				log.Println("controller: re-applying configurations after suspend resume")
				c.workQueueCh[fnApplyConfigs].noisy <- struct{}{}
			}
		case <-haltCtx.Done():
			log.Println("controller: exiting handlePowerEvent")
			return
		}
	}
}

func (c *Controller) handleKeyPress(haltCtx context.Context) {
	for {
		select {
		case keyCode := <-c.keyCodeCh:
			switch keyCode {
			case 56:
				log.Println("hid: ROG Key Pressed (debounced)")
				c.workQueueCh[58].noisy <- struct{}{}

			case 174:
				log.Println("hid: Fn + F5 Pressed (debounced)")
				c.workQueueCh[174].noisy <- struct{}{}

			case 234:
				log.Println("hid: volume down Pressed")

			case 233:
				log.Println("hid: volume up Pressed")

			case 124:
				log.Println("hid: mute/unmute microphone Pressed")
				c.workQueueCh[fnVolCtrl].noisy <- struct{}{}

			case 107:
				log.Println("hid: toggle enable/disable touchpad Pressed")
				c.workQueueCh[fnToggleTouchPad].noisy <- struct{}{}

			case
				16,  // screen brightness down
				32,  // screen brightness up
				108, // sleep
				136: // RF kill toggle
				c.workQueueCh[fnHwCtrl].noisy <- keyCode

			case
				178, // Fn + Arrow Left
				179, // Fn + Arrow Right
				197, // keyboard brightness down (Fn + Arrow Down)
				196: // keyboard brightness up (Fn + Arrow Up)
				c.workQueueCh[fnKbCtrl].noisy <- keyCode

			default:
				log.Printf("hid: Unknown %d\n", keyCode)
			}
		case <-haltCtx.Done():
			log.Println("controller: exiting handleKeyPress")
			return
		}
	}
}

type notification struct {
	title   string
	message string
}

func (c *Controller) sendToastNotification(n notification) error {
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

// In the future this will be notifying OSD
func (c *Controller) handleNotify(haltCtx context.Context) {
	for {
		select {
		case msg := <-c.notifyQueueCh:
			if err := c.sendToastNotification(msg); err != nil {
				log.Printf("Error sending toast notification: %s\n", err)
			}
		case <-haltCtx.Done():
			log.Println("controller: exiting handleNotify")
			return
		}
	}
}

func (c *Controller) handleWorkQueue(haltCtx context.Context) {
	for {
		select {
		case ev := <-c.workQueueCh[58].clean:
			log.Printf("controller: ROG Key pressed %d times\n", ev.Counter)
			if int(ev.Counter) <= len(c.Config.ROGKey) {
				if err := run("cmd.exe", "/C", c.Config.ROGKey[ev.Counter-1]); err != nil {
					log.Println(err)
				}
			}

		case ev := <-c.workQueueCh[174].clean:
			log.Printf("controller: Fn + F5 pressed %d times\n", ev.Counter)
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
			c.workQueueCh[fnPersistConfigs].noisy <- struct{}{}

		case <-c.workQueueCh[fnCheckCharger].clean:
			function := make([]byte, 4)
			binary.LittleEndian.PutUint32(function, atkacpi.DstsCheckCharger)
			status, err := c.wmi.Evaluate(atkacpi.DSTS, function)
			if err != nil {
				log.Println("controller: cannot check charger status")
				continue
			}
			switch binary.LittleEndian.Uint32(status[0:4]) {
			case 0x0:
				log.Printf("controller: charger is not plugged in")
			case 0x10001:
				log.Printf("controller: 180W charger plugged in")
			case 0x10002:
				log.Printf("controller: USB-C PD charger plugged in")
			}

		case <-c.workQueueCh[fnPersistConfigs].clean:
			if c.isDryRun {
				continue
			}
			if err := c.Config.Registry.Save(); err != nil {
				log.Fatalln("controller: error saving to registry", err)
			}

		case <-c.workQueueCh[fnApplyConfigs].clean:
			// load configs from registry and try to reapply
			if err := c.Config.Registry.Load(); err != nil {
				log.Fatalln("controller: error loading configurations from registry", err)
			}
			if err := c.Config.Registry.Apply(); err != nil {
				log.Fatalln("controller: error applying configurations", err)
			}
			c.notifyQueueCh <- notification{
				title:   "Settings Loaded from Registry",
				message: fmt.Sprintf("Current Thermal Plan: %s", c.Config.Thermal.CurrentProfile().Name),
			}

		case ev := <-c.workQueueCh[fnKbCtrl].clean:
			switch ev.Data.(uint32) {
			case 178:
				log.Println("kbCtrl: Fn + Array Left Pressed")
				if c.Config.EnableExperimental {
					log.Println("kbCtrl: (experimental) remapping to PgUp")
					if err := c.Config.KeyboardControl.EmulateKeyPress(0x49); err != nil {
						log.Printf("kbCtrl: error remapping: %v\n", err)
					}
				}
			case 179:
				log.Println("kbCtrl: Fn + Array Right Pressed")
				if c.Config.EnableExperimental {
					log.Println("kbCtrl: (experimental) remapping to PgDown")
					if err := c.Config.KeyboardControl.EmulateKeyPress(0x51); err != nil {
						log.Printf("kbCtrl: error remapping: %v\n", err)
					}
				}
			case 197: // keyboard brightness down (Fn + Arrow Down)
				log.Println("kbCtrl: Decrease keyboard backlight")
				c.Config.KeyboardControl.BrightnessDown()
				c.workQueueCh[fnPersistConfigs].noisy <- struct{}{}

			case 196: // keyboard brightness up (Fn + Arrow Up)
				log.Println("kbCtrl: Increase keyboard backlight")
				c.Config.KeyboardControl.BrightnessUp()
				c.workQueueCh[fnPersistConfigs].noisy <- struct{}{}
			}

		case <-c.workQueueCh[fnToggleTouchPad].clean:
			c.Config.KeyboardControl.ToggleTouchPad()

		case <-c.workQueueCh[fnVolCtrl].clean:
			if err := c.Config.VolumeControl.ToggleMicrophoneMute(); err != nil {
				log.Printf("volCtrl: error toggling mute: %+v\n", err)
			}

		case ev := <-c.workQueueCh[fnHwCtrl].clean:
			keyCode := ev.Data.(uint32)
			args := make([]byte, 8)
			log.Printf("hwCtrl: notification from keypress on %d\n", keyCode)

			binary.LittleEndian.PutUint32(args[0:], atkacpi.DevsHardwareCtrl)
			binary.LittleEndian.PutUint32(args[4:], keyCode)

			_, err := c.wmi.Evaluate(atkacpi.DEVS, args)
			if err != nil {
				log.Fatalln("hwCtrl: error sending key code to ATKACPI", err)
			}

		case <-c.workQueueCh[fnBeforeSuspend].clean:
			log.Println("kbCtrl: turning off keyboard backlight")
			c.Config.KeyboardControl.SetBrightness(keyboard.OFF)

		case <-haltCtx.Done():
			log.Println("controller: exiting handleWorkQueue")
			return
		}
	}
}

func (c *Controller) Run(haltCtx context.Context) {

	c.initialize(haltCtx)

	go c.handleNotify(haltCtx)
	go c.handleWorkQueue(haltCtx)
	go c.handlePowerEvent(haltCtx)
	go c.handleACPINotification(haltCtx)
	go c.handleKeyPress(haltCtx)

	<-haltCtx.Done()
	time.Sleep(time.Millisecond * 50)
	c.Config.Registry.Close()
	c.Config.VolumeControl.Close()
}

func run(commands ...string) error {
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Start()
}
