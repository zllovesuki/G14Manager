package controller

import (
	"context"
	"log"
	"time"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/plugin"
	"github.com/zllovesuki/G14Manager/system/power"
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

// Config contains the configurations for the controller
type Config struct {
	WMI atkacpi.WMI

	Plugins  []plugin.Plugin
	Registry persist.ConfigRegistry

	LogoPath string

	Context  context.Context
	cancelFn context.CancelFunc
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
	startErrorCh  chan error
	unrecoverable bool

	keyCodeCh  chan uint32
	acpiCh     chan uint32
	powerEvCh  chan uint32
	pluginCbCh chan plugin.Callback

	ctx      context.Context
	cancelFn context.CancelFunc
}

func (c *Controller) initialize(haltCtx context.Context) error {
	for _, p := range c.Config.Plugins {
		if err := p.Initialize(); err != nil {
			return errors.Wrap(err, "[controller] plugin initializtion error")
		}
	}

	devices, err := keyboard.NewHidListener(haltCtx, c.keyCodeCh)
	if err != nil {
		return errors.Wrap(err, "[controller] error initializing hid listener")
	}
	log.Printf("hid devices: %+v\n", devices)

	err = atkacpi.NewACPIListener(haltCtx, c.acpiCh)
	if err != nil {
		return errors.Wrap(err, "[controller] error initializing atkacpi wmi listener")
	}

	// TODO: Unregister when we are done
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
	c.workQueueCh[fnCheckCharger].noisy <- true // indicating initial (startup) check

	// c.notifyQueueCh <- util.Notification{
	// 	Title:   "Settings Loaded from Registry",
	// 	Message: "Enjoy your bloat-free G14",
	// }

	return nil
}

func (c *Controller) startPlugins(haltCtx context.Context) {
	for _, p := range c.Config.Plugins {
		errChan := p.Run(haltCtx, c.pluginCbCh)
		go func(ch <-chan error) {
			for {
				select {
				case <-haltCtx.Done():
					return
				case err := <-ch:
					if err != nil {
						log.Printf("Plugin returned error: %v\n", err)
						c.errorCh <- err
					}
				}
			}
		}(errChan)
	}
}

func (c *Controller) Serve() {

	c.ctx, c.cancelFn = context.WithCancel(context.Background())
	defer c.cancelFn()

	log.Println("[controller] Starting controller loop")

	if err := c.initialize(c.ctx); err != nil {
		log.Printf("[controller] error initializing: %+v\n", err)
		c.unrecoverable = true
		c.startErrorCh <- err
		return
	}

	c.startPlugins(c.ctx)

	// defined in controller_loop.go
	go c.handlePluginCallback(c.ctx)
	go c.handleNotify(c.ctx)
	go c.handleWorkQueue(c.ctx)
	go c.handlePowerEvent(c.ctx)
	go c.handleACPINotification(c.ctx)
	go c.handleKeyPress(c.ctx)

	for {
		select {
		case <-c.ctx.Done():
			if err := c.Registry.Save(); err != nil {
				log.Printf("[controller] unable to save to config registry: %+v\n", err)
			}
			log.Println("[controller] exiting Run loop")
			return
		case err := <-c.errorCh:
			log.Printf("[controller] Recoverable error in controller loop: %v\n", err)
			return
		}
	}
}

func (c *Controller) IsCompletable() bool {
	return !c.unrecoverable
}

func (c *Controller) Stop() {
	c.cancelFn()
}
