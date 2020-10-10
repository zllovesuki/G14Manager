package controller

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	kb "github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/system/volume"
	"github.com/zllovesuki/G14Manager/util"

	"github.com/pkg/errors"
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
	fnUtilityKey            // for when ROG Key is pressed
	fnThermalProfile        // for Fn+F5 to switch between profiles
)

type Config struct {
	EnableExperimental bool
	WMI                atkacpi.WMI

	VolumeControl   *volume.Control
	KeyboardControl *kb.Control
	Thermal         *thermal.Control
	Registry        persist.ConfigRegistry

	ROGKey []string
}

type workQueue struct {
	noisy chan<- interface{}
	clean <-chan util.DebounceEvent
}

type Controller struct {
	Config

	notifyQueueCh chan util.Notification
	workQueueCh   map[uint32]workQueue
	errorCh       chan error

	keyCodeCh chan uint32
	acpiCh    chan uint32
	powerEvCh chan uint32
}

func newController(conf Config) (*Controller, error) {
	if conf.WMI == nil {
		return nil, errors.New("[controller] nil WMI is invalid")
	}
	if conf.VolumeControl == nil {
		return nil, errors.New("[controller] nil volume.Control is invalid")
	}
	if conf.KeyboardControl == nil {
		return nil, errors.New("[controller] nil keyboard.Control is invalid")
	}
	if conf.Thermal == nil {
		return nil, errors.New("[controller] nil Thermal is invalid")
	}
	if conf.Registry == nil {
		return nil, errors.New("[controller] nil Registry is invalid")
	}
	if len(conf.ROGKey) == 0 {
		return nil, errors.New("[controller] empty key remap is invalid")
	}
	return &Controller{
		Config: conf,

		notifyQueueCh: make(chan util.Notification, 10),
		workQueueCh:   make(map[uint32]workQueue),
		errorCh:       make(chan error),

		keyCodeCh: make(chan uint32, 1),
		acpiCh:    make(chan uint32, 1),
		powerEvCh: make(chan uint32, 1),
	}, nil
}

func (c *Controller) initialize(haltCtx context.Context) error {
	// Do we need to lock os thread on any of these?

	devices, err := kb.NewHidListener(haltCtx, c.keyCodeCh)
	if err != nil {
		return errors.Wrap(err, "[controller] error initializing hid listener")
	}
	log.Printf("hid devices: %+v\n", devices)

	// This is a bit buggy, as Windows seems to time out our connection to WMI
	// TODO: Find a better way os listening from atk wmi events
	err = atkacpi.NewACPIListener(haltCtx, c.acpiCh)
	if err != nil {
		return errors.Wrap(err, "[controller] error initializing atkacpi wmi listener")
	}

	err = power.NewEventListener(c.powerEvCh)
	if err != nil {
		return errors.Wrap(err, "[controller] error initializing power event listener")
	}

	initBuf := make([]byte, 4)
	if _, err := c.Config.WMI.Evaluate(atkacpi.INIT, initBuf); err != nil {
		return errors.Wrap(err, "[controller] cannot initialize ATKD")
	}

	debounceKeys := []uint32{
		fnUtilityKey,
		fnThermalProfile,
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

	// load and apply configurations
	c.workQueueCh[fnApplyConfigs].noisy <- struct{}{}
	// seed the channel so we get the the charger status
	c.workQueueCh[fnCheckCharger].noisy <- struct{}{}

	return nil
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
			log.Println("[controller] exiting handleACPINotification")
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
				log.Println("[controller] housekeeping before suspend")
				c.workQueueCh[fnBeforeSuspend].noisy <- struct{}{}
			case power.PBT_APMRESUMEAUTOMATIC:
				log.Println("[controller] re-applying configurations after suspend resume")
				c.workQueueCh[fnApplyConfigs].noisy <- struct{}{}
			}
		case <-haltCtx.Done():
			log.Println("[controller] exiting handlePowerEvent")
			return
		}
	}
}

func (c *Controller) handleKeyPress(haltCtx context.Context) {
	for {
		select {
		case keyCode := <-c.keyCodeCh:
			switch keyCode {
			case kb.KeyROG:
				log.Println("hid: ROG Key Pressed (debounced)")
				c.workQueueCh[fnUtilityKey].noisy <- struct{}{}

			case kb.KeyFnF5:
				log.Println("hid: Fn + F5 Pressed (debounced)")
				c.workQueueCh[fnThermalProfile].noisy <- struct{}{}

			case kb.KeyVolDown:
				log.Println("hid: volume down Pressed")

			case kb.KeyVolUp:
				log.Println("hid: volume up Pressed")

			case kb.KeyMuteMic:
				log.Println("hid: mute/unmute microphone Pressed")
				c.workQueueCh[fnVolCtrl].noisy <- struct{}{}

			case kb.KeyTpadToggle:
				log.Println("hid: toggle enable/disable touchpad Pressed")
				c.workQueueCh[fnToggleTouchPad].noisy <- struct{}{}

			case
				kb.KeyLCDUp,
				kb.KeyLCDDown,
				kb.KeySleep,
				kb.KeyRFKill:
				c.workQueueCh[fnHwCtrl].noisy <- keyCode

			case
				kb.KeyFnLeft,
				kb.KeyFnRight,
				kb.KeyFnUp,
				kb.KeyFnDown:
				c.workQueueCh[fnKbCtrl].noisy <- keyCode

			default:
				log.Printf("hid: Unknown %d\n", keyCode)
			}
		case <-haltCtx.Done():
			log.Println("[controller] exiting handleKeyPress")
			return
		}
	}
}

// In the future this will be notifying OSD
func (c *Controller) handleNotify(haltCtx context.Context) {
	for {
		select {
		case msg := <-c.notifyQueueCh:
			if err := util.SendToastNotification(appName, msg); err != nil {
				log.Printf("Error sending toast notification: %s\n", err)
			}
		case <-haltCtx.Done():
			log.Println("[controller] exiting handleNotify")
			return
		}
	}
}

func (c *Controller) handleWorkQueue(haltCtx context.Context) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for {
		select {
		case ev := <-c.workQueueCh[fnUtilityKey].clean:
			log.Printf("[controller] ROG Key pressed %d times\n", ev.Counter)
			if int(ev.Counter) <= len(c.Config.ROGKey) {
				if err := run("cmd.exe", "/C", c.Config.ROGKey[ev.Counter-1]); err != nil {
					log.Println(err)
				}
			}

		case ev := <-c.workQueueCh[fnThermalProfile].clean:
			log.Printf("[controller] Fn + F5 pressed %d times\n", ev.Counter)
			next, err := c.Config.Thermal.NextProfile(int(ev.Counter))
			message := fmt.Sprintf("Thermal plan changed to %s", next)
			if err != nil {
				log.Println(err)
				message = err.Error()
			}
			c.notifyQueueCh <- util.Notification{
				Title:   "Toggle Thermal Plan",
				Message: message,
			}
			c.workQueueCh[fnPersistConfigs].noisy <- struct{}{}

		case <-c.workQueueCh[fnCheckCharger].clean:
			function := make([]byte, 4)
			binary.LittleEndian.PutUint32(function, atkacpi.DstsCheckCharger)
			status, err := c.Config.WMI.Evaluate(atkacpi.DSTS, function)
			if err != nil {
				c.errorCh <- errors.New("[controller] cannot check charger status")
				return
			}
			switch binary.LittleEndian.Uint32(status[0:4]) {
			case 0x0:
				log.Printf("[controller] charger is not plugged in")
			case 0x10001:
				log.Printf("[controller] 180W charger plugged in")
			case 0x10002:
				log.Printf("[controller] USB-C PD charger plugged in")
			}

		case <-c.workQueueCh[fnPersistConfigs].clean:
			if err := c.Config.Registry.Save(); err != nil {
				c.errorCh <- errors.Wrap(err, "[controller] error saving to registry")
				return
			}

		case <-c.workQueueCh[fnApplyConfigs].clean:
			// load configs from registry and try to reapply
			if err := c.Config.Registry.Load(); err != nil {
				c.errorCh <- errors.Wrap(err, "[controller] error loading configurations from registry")
				return
			}
			if err := c.Config.Registry.Apply(); err != nil {
				c.errorCh <- errors.Wrap(err, "[controller] error applying configurations")
				return
			}
			c.notifyQueueCh <- util.Notification{
				Title:   "Settings Loaded from Registry",
				Message: fmt.Sprintf("Current Thermal Plan: %s", c.Config.Thermal.CurrentProfile().Name),
			}

		case ev := <-c.workQueueCh[fnKbCtrl].clean:
			switch ev.Data.(uint32) {
			case 178:
				log.Println("kbCtrl: Fn + Array Left Pressed")
				if c.Config.EnableExperimental {
					log.Println("kbCtrl: (experimental) remapping to PgUp")
					if err := c.Config.KeyboardControl.EmulateKeyPress(kb.KeyPgUp); err != nil {
						log.Printf("kbCtrl: error remapping: %v\n", err)
					}
				}
			case 179:
				log.Println("kbCtrl: Fn + Array Right Pressed")
				if c.Config.EnableExperimental {
					log.Println("kbCtrl: (experimental) remapping to PgDown")
					if err := c.Config.KeyboardControl.EmulateKeyPress(kb.KeyPgDown); err != nil {
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
				c.errorCh <- errors.Wrap(err, "[volCtrl] error toggling mute")
				return
			}

		case ev := <-c.workQueueCh[fnHwCtrl].clean:
			keyCode := ev.Data.(uint32)
			args := make([]byte, 8)
			log.Printf("hwCtrl: notification from keypress on %d\n", keyCode)

			binary.LittleEndian.PutUint32(args[0:], atkacpi.DevsHardwareCtrl)
			binary.LittleEndian.PutUint32(args[4:], keyCode)

			_, err := c.Config.WMI.Evaluate(atkacpi.DEVS, args)
			if err != nil {
				c.errorCh <- errors.Wrap(err, "hwCtrl: error sending key code to ATKACPI")
				return
			}

		case <-c.workQueueCh[fnBeforeSuspend].clean:
			log.Println("kbCtrl: turning off keyboard backlight")
			c.Config.KeyboardControl.SetBrightness(kb.OFF)

		case <-haltCtx.Done():
			log.Println("[controller] exiting handleWorkQueue")
			return
		}
	}
}

func (c *Controller) Run(haltCtx context.Context) error {

	if err := c.initialize(haltCtx); err != nil {
		return errors.Wrap(err, "[controller] error initializing")
	}

	go c.handleNotify(haltCtx)
	go c.handleWorkQueue(haltCtx)
	go c.handlePowerEvent(haltCtx)
	go c.handleACPINotification(haltCtx)
	go c.handleKeyPress(haltCtx)

	for {
		select {
		case <-haltCtx.Done():
			time.Sleep(time.Millisecond * 50)
			c.Config.Registry.Close()
			return nil
		case err := <-c.errorCh:
			log.Printf("[controller] Unrecoverable error in controller loop: %v\n", err)
			c.Config.Registry.Close()
			return err
		}
	}
}

func run(commands ...string) error {
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Start()
}
