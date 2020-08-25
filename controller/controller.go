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
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/system/volume"
	"github.com/zllovesuki/G14Manager/util"

	"gopkg.in/toast.v1"
)

const (
	appName = "G14Manager"
)

type Controller interface {
	Run(haltCtx context.Context)
}

var _ Controller = &controller{}

type Config struct {
	EnableExperimental bool

	VolumeControl   *volume.Control
	KeyboardControl *keyboard.Control
	Thermal         *thermal.Control
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

	keyCodeCh chan uint32
	acpiCh    chan uint32
	hwCtrlCh  chan uint32
	kbCtrlCh  chan uint32

	wmi      atkacpi.WMI
	isDryRun bool
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
		Config: conf,

		notifyQueueCh: make(chan notification, 10),
		debounceCh:    make(map[uint32]keyedDebounce),

		keyCodeCh: make(chan uint32, 1),
		acpiCh:    make(chan uint32, 1),
		hwCtrlCh:  make(chan uint32, 1),
		kbCtrlCh:  make(chan uint32, 1),

		isDryRun: os.Getenv("DRY_RUN") != "",
	}, nil
}

func (c *controller) initialize(haltCtx context.Context) {

	devices, err := keyboard.NewHidListener(haltCtx, c.keyCodeCh)
	if err != nil {
		log.Fatalln("controller: error initializing hidListener", err)
	}
	log.Printf("hid devices: %+v\n", devices)

	err = atkacpi.NewACPIListener(haltCtx, c.acpiCh)
	if err != nil {
		log.Fatalln("controller: error initializing wmiListener", err)
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

	// "keys" with 0x are for internal functions
	keys := []uint32{
		58,  // ROG Key
		174, // Fn + F5

		0x0,   // for debouncing persisting to Registry
		0x123, // for debouncing power input change acpi event
	}
	for _, key := range keys {
		// TODO: make debounce interval configurable for accessbility
		in, out := util.Debounce(haltCtx, time.Millisecond*500)
		c.debounceCh[key] = keyedDebounce{
			noisy: in,
			clean: out,
		}
	}

	// seed the channel so we get the the charger status
	c.debounceCh[0x123].noisy <- struct{}{}
}

func (c *controller) handleACPINotification(haltCtx context.Context) {
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
				c.debounceCh[0x123].noisy <- struct{}{}
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

func (c *controller) handleACPIHardwareControl(haltCtx context.Context) {
	// pre-allocate memory so we don't have to reallocate everytime we need it
	args := make([]byte, 8)

	for {
		select {
		case keyCode := <-c.hwCtrlCh:
			log.Printf("hwCtrl: notification from keypress on %d\n", keyCode)

			binary.LittleEndian.PutUint32(args[0:], atkacpi.DevsHardwareCtrl)
			binary.LittleEndian.PutUint32(args[4:], keyCode)

			_, err := c.wmi.Evaluate(atkacpi.DEVS, args)
			if err != nil {
				log.Fatalln("hwCtrl: error sending key code to ATKACPI", err)
			}
		case <-haltCtx.Done():
			log.Println("controller: exiting handleACPIHardwareControl")
			return
		}
	}
}

func (c *controller) handleKeyboardControl(haltCtx context.Context) {
	for {
		select {
		case keyCode := <-c.kbCtrlCh:
			log.Printf("kbCtrl: notification from keypress on %d\n", keyCode)

			switch keyCode {
			case 197: // keyboard brightness down (Fn + Arrow Down)
				log.Println("kbCtrl: Decrease keyboard backlight")
				c.Config.KeyboardControl.BrightnessDown()
				c.debounceCh[0].noisy <- struct{}{}

			case 196: // keyboard brightness up (Fn + Arrow Up)
				log.Println("kbCtrl: Increase keyboard backlight")
				c.Config.KeyboardControl.BrightnessUp()
				c.debounceCh[0].noisy <- struct{}{}

			case 107: // toggle touchpad disable/enable
				log.Println("kbCtrl: Toggle disable/enable TouchPad")
				c.Config.KeyboardControl.ToggleTouchPad()
			}
		case <-haltCtx.Done():
			log.Println("controller: exiting handleKeyboardControl")
			return
		}
	}
}

func (c *controller) handleKeyPress(haltCtx context.Context) {
	for {
		select {
		case keyCode := <-c.keyCodeCh:
			switch keyCode {
			case 56:
				log.Println("hid: ROG Key Pressed (debounced)")
				c.debounceCh[58].noisy <- struct{}{}

			case 174:
				log.Println("hid: Fn + F5 Pressed (debounced)")
				c.debounceCh[174].noisy <- struct{}{}

			case 178:
				log.Println("hid: Fn + Array Left Pressed")
				if c.Config.EnableExperimental {
					log.Println("controller: (experimental) remapping to PgUp")
					if err := c.Config.KeyboardControl.EmulateKeyPress(0x49); err != nil {
						log.Printf("controller: %v\n", err)
					}
				}

			case 179:
				log.Println("hid: Fn + Array Right Pressed")
				if c.Config.EnableExperimental {
					log.Println("controller: (experimental) remapping to PgDown")
					if err := c.Config.KeyboardControl.EmulateKeyPress(0x51); err != nil {
						log.Printf("controller: %v\n", err)
					}
				}

			case 124: // TODO: make it run on a separate goroutine
				log.Println("hid: mute/unmute microphone Pressed")
				c.Config.VolumeControl.ToggleMicrophoneMute()

			case 234:
				log.Println("hid: volume down Pressed")

			case 233:
				log.Println("hid: volume up Pressed")

			case
				16,  // screen brightness down
				32,  // screen brightness up
				108, // sleep
				136: // RF kill toggle
				// notify the ATK interface on some special key combo for hardware functions
				c.hwCtrlCh <- keyCode

			// TODO: revisit this
			case
				197, // keyboard brightness down (Fn + Arrow Down)
				196, // keyboard brightness up (Fn + Arrow Up)
				107: // toggle touchpad disable/enable
				// let another goroutine handles it
				c.kbCtrlCh <- keyCode

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

func (c *controller) sendToastNotification(n notification) error {
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
func (c *controller) handleNotify(haltCtx context.Context) {
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

func (c *controller) handleDebounce(haltCtx context.Context) {
	for {
		select {
		case ev := <-c.debounceCh[58].clean:
			log.Printf("controller: ROG Key pressed %d times\n", ev.Counter)
			if int(ev.Counter) <= len(c.Config.ROGKey) {
				if err := run("cmd.exe", "/C", c.Config.ROGKey[ev.Counter-1]); err != nil {
					log.Println(err)
				}
			}

		case ev := <-c.debounceCh[174].clean:
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
			c.debounceCh[0].noisy <- struct{}{}

		case <-c.debounceCh[0].clean:
			if c.isDryRun {
				continue
			}
			if err := c.Config.Registry.Save(); err != nil {
				log.Fatalln("controller: error saving to registry", err)
			}

		case <-c.debounceCh[0x123].clean:
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

		case <-haltCtx.Done():
			log.Println("controller: exiting handleDebounce")
			return
		}
	}
}

func (c *controller) Run(haltCtx context.Context) {

	log.Println("controller: loading configuration from Registry")
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
	go c.handleACPINotification(haltCtx)
	go c.handleACPIHardwareControl(haltCtx)
	go c.handleKeyboardControl(haltCtx)
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
