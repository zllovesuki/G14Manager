package controller

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"runtime"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/plugin"
	kb "github.com/zllovesuki/G14Manager/system/plugin/keyboard"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/util"

	"github.com/pkg/errors"
)

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
				c.workQueueCh[fnCheckCharger].noisy <- false // indicating non initial (continuous) check
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
				log.Println("[controller] housekeeping after suspend")
				c.workQueueCh[fnAfterSuspend].noisy <- struct{}{}
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
	defer func() {
		if r := recover(); r != nil {
			err := r.(error)
			c.errorCh <- err
		}
	}()

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

		case ev := <-c.workQueueCh[fnCheckCharger].clean:
			function := make([]byte, 4)
			binary.LittleEndian.PutUint32(function, atkacpi.DstsCheckCharger)
			status, err := c.Config.WMI.Evaluate(atkacpi.DSTS, function)
			if err != nil {
				c.errorCh <- errors.New("[controller] cannot check charger status")
				return
			}
			isInitialCheck := ev.Data.(bool)
			switch binary.LittleEndian.Uint32(status[0:4]) {
			case 0x0:
				log.Println("[controller] charger is not plugged in")
				if !isInitialCheck {
					c.workQueueCh[fnAutoThermal].noisy <- chargerUnplugged
				}
			case 0x10001:
				log.Println("[controller] 180W charger plugged in")
				if !isInitialCheck {
					c.workQueueCh[fnAutoThermal].noisy <- chargerPluggedIn
				}
			case 0x10002:
				log.Println("[controller] USB-C PD charger plugged in")
				if !isInitialCheck {
					c.workQueueCh[fnAutoThermal].noisy <- chargerPluggedIn
				}
			}

		case ev := <-c.workQueueCh[fnAutoThermal].clean:
			if c.Config.EnabledFeatures.AutoThermalProfile {
				pluggedInStatus := ev.Data.(chargerStatus)
				log.Printf("[controller] automatically switching thermal profile. Charger is: %s\n", pluggedInStatus)

				var next string
				var err error
				var message string
				// TODO: make it configurable
				if pluggedInStatus == chargerPluggedIn {
					next, err = c.Config.Thermal.SwitchToProfile("Performance")
				} else {
					next, err = c.Config.Thermal.SwitchToProfile("Silent")
				}
				if err != nil {
					log.Println(err)
					message = err.Error()
				} else {
					message = fmt.Sprintf("Thermal plan changed to %s", next)
				}
				c.notifyQueueCh <- util.Notification{
					Title:   "Automatic Thermal Plan Switching: " + pluggedInStatus.String(),
					Message: message,
				}
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
			case kb.KeyFnLeft:
				log.Println("kbCtrl: Fn + Arrow Left Pressed")
				if c.Config.EnabledFeatures.ExperimentalFnRemap {
					log.Println("kbCtrl: (experimental) remapping to PgUp")
					c.notifyPlugins(plugin.EvtKbEmulateKeyPress, kb.KeyPgUp)
				}
			case kb.KeyFnRight:
				log.Println("kbCtrl: Fn + Arrow Right Pressed")
				if c.Config.EnabledFeatures.ExperimentalFnRemap {
					log.Println("kbCtrl: (experimental) remapping to PgDown")
					c.notifyPlugins(plugin.EvtKbEmulateKeyPress, kb.KeyPgDown)
				}
			case kb.KeyFnDown:
				log.Println("kbCtrl: Decrease keyboard backlight")
				c.notifyPlugins(plugin.EvtKbBrightnessDown, nil)
				c.workQueueCh[fnPersistConfigs].noisy <- struct{}{}

			case kb.KeyFnUp:
				log.Println("kbCtrl: Increase keyboard backlight")
				c.notifyPlugins(plugin.EvtKbBrightnessUp, nil)
				c.workQueueCh[fnPersistConfigs].noisy <- struct{}{}
			}

		case <-c.workQueueCh[fnToggleTouchPad].clean:
			c.notifyPlugins(plugin.EvtKbToggleTouchpad, nil)

		case <-c.workQueueCh[fnVolCtrl].clean:
			c.notifyPlugins(plugin.EvtVolToggleMute, nil)

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
			c.notifyPlugins(plugin.EvtKbBrightnessSet, kb.OFF)

		case <-c.workQueueCh[fnAfterSuspend].clean:
			log.Println("[controller] reinitialize kbCtrl and apply config")
			c.notifyPlugins(plugin.EvtKbReInit, nil)
			c.workQueueCh[fnApplyConfigs].noisy <- struct{}{}

		case <-haltCtx.Done():
			log.Println("[controller] exiting handleWorkQueue")
			return
		}
	}
}

func (c Controller) notifyPlugins(evt plugin.Event, val interface{}) {
	t := plugin.Task{
		Event: evt,
		Value: val,
	}
	for _, p := range c.Config.Plugins {
		go p.Notify(t)
	}
}
