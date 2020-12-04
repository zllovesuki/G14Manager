package controller

import (
	"context"
	"log"
	"os/exec"
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
	// AutoThermalDelay defines how long the Controller should wait before changing thermal profile when power source is changed
	AutoThermalDelay = time.Second * 5
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
	fnAfterSuspend          // for doing work after suspend
	fnUtilityKey            // for when ROG Key is pressed
	fnThermalProfile        // for Fn+F5 to switch between profiles
	fnAutoThermal           // for switching thermal on power source change
)

type chargerStatus int

const (
	chargerPluggedIn chargerStatus = iota
	chargerUnplugged
)

// https://yourbasic.org/golang/iota/
func (c chargerStatus) String() string {
	return [...]string{"Plugged In", "Unplugged"}[c]
}

// Features contains feature flags
type Features struct {
	ExperimentalFnRemap bool
	AutoThermalProfile  bool
}

// Config contains the configurations for the controller
type Config struct {
	WMI atkacpi.WMI

	VolumeControl   *volume.Control
	KeyboardControl *kb.Control
	Thermal         *thermal.Control
	Registry        persist.ConfigRegistry

	EnabledFeatures Features
	ROGKey          []string
}

type workQueue struct {
	noisy chan<- interface{}
	clean <-chan util.DebounceEvent
}

// Controller contains configuration for the controller loop
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
		workQueueCh:   make(map[uint32]workQueue, 1),
		errorCh:       make(chan error),

		keyCodeCh: make(chan uint32, 1),
		acpiCh:    make(chan uint32, 1),
		powerEvCh: make(chan uint32, 1),
	}, nil
}

func (c *Controller) initialize(haltCtx context.Context) error {
	// Do we need to lock os thread on any of these?

	go c.VolumeControl.Run(haltCtx)

	if err := c.Config.VolumeControl.CheckMicrophoneMute(); err != nil {
		return errors.Wrap(err, "[controller] error checking for microphone mute status")
	}

	if err := c.Config.KeyboardControl.InitializeInterface(); err != nil {
		return errors.Wrap(err, "[controller] error initializing kbCtrl")
	}

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
		fnAfterSuspend,
	}
	for _, work := range workQueueImmediate {
		in, out := util.PassThrough(haltCtx)
		c.workQueueCh[work] = workQueue{
			noisy: in,
			clean: out,
		}
	}

	workQueueDebounced := []struct {
		code  uint32
		delay time.Duration
	}{
		{
			code:  fnPersistConfigs,
			delay: time.Second,
		},
		{
			code:  fnAutoThermal,
			delay: AutoThermalDelay,
		},
	}
	for _, work := range workQueueDebounced {
		in, out := util.Debounce(haltCtx, work.delay)
		c.workQueueCh[work.code] = workQueue{
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

// Run will start the controller loop and blocked until context cancel, or an error has occurred
func (c *Controller) Run(haltCtx context.Context) error {

	ctx, cancel := context.WithCancel(haltCtx)
	defer func() {
		c.Config.Registry.Close()
		cancel()
	}()

	log.Println("[controller] Starting controller loop")

	if err := c.initialize(ctx); err != nil {
		return errors.Wrap(err, "[controller] error initializing")
	}

	// defined in controller_loop.go
	go c.handleNotify(ctx)
	go c.handleWorkQueue(ctx)
	go c.handlePowerEvent(ctx)
	go c.handleACPINotification(ctx)
	go c.handleKeyPress(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-c.errorCh:
			log.Printf("[controller] Unrecoverable error in controller loop: %v\n", err)
			return err
		}
	}
}

func run(commands ...string) error {
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Start()
}
