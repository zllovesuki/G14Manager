package thermal

// This is inspired by the atrofac utility (https://github.com/cronosun/atrofac)

/*
Factory fan curves:
device 0x24 in profile 0x0 has fan curve [20 48 51 54 57 61 65 98 14 19 22 26 31 43 49 56]
device 0x24 in profile 0x1 has fan curve [20 44 47 50 53 56 60 98 11 14 17 19 22 26 31 38]
device 0x24 in profile 0x2 has fan curve [20 50 55 60 65 70 75 98 21 26 31 38 43 48 56 65]

device 0x25 in profile 0x0 has fan curve [20 48 51 54 57 61 65 98 14 21 25 28 34 44 51 61]
device 0x25 in profile 0x1 has fan curve [20 44 47 50 53 56 60 98 11 14 18 21 25 28 34 40]
device 0x25 in profile 0x2 has fan curve [20 50 55 60 65 70 75 98 25 28 34 40 44 49 61 70]
*/

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/plugin"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/util"
)

const (
	thermalPersistKey = "ThermalProfile"
)

// TODO: validate these constants are actually what they say they are
const (
	throttlePlanPerformance uint32 = 0x00
	throttlePlanTurbo       uint32 = 0x01
	throttlePlanSilent      uint32 = 0x02
)

// Profile contain each thermal profile definition
// TODO: Revisit this
type Profile struct {
	Name             string
	WindowsPowerPlan string
	ThrottlePlan     uint32
	CPUFanCurve      *fanTable
	GPUFanCurve      *fanTable
}

// Control defines contains the Windows Power Option and list of thermal profiles
type Control struct {
	Config

	mu                  sync.RWMutex
	wmi                 atkacpi.WMI
	currentProfileIndex int

	errorCh chan error
	queue   chan plugin.Notification
}

// Config defines the entry point for Windows Power Option and a list of thermal profiles
type Config struct {
	WMI               atkacpi.WMI
	PowerCfg          *power.Cfg
	Profiles          []Profile
	AutoThermal       bool
	AutoThermalConfig struct {
		PluggedIn string
		Unplugged string
	}
}

var _ plugin.Plugin = &Control{}

// NewControl allows you to cycle to the next thermal profile
func NewControl(conf Config) (*Control, error) {
	if conf.WMI == nil {
		return nil, errors.New("nil WMI is invalid")
	}
	if conf.PowerCfg == nil {
		return nil, errors.New("nil PowerCfg is invalid")
	}
	if len(conf.Profiles) == 0 {
		return nil, errors.New("empty Profiles is invalid")
	}
	if conf.AutoThermal {
		if len(conf.AutoThermalConfig.PluggedIn) == 0 || len(conf.AutoThermalConfig.Unplugged) == 0 {
			return nil, errors.New("must specify auto thermal profiles if enabled")
		}
	}

	return &Control{
		Config:              conf,
		wmi:                 conf.WMI,
		currentProfileIndex: 0,
		errorCh:             make(chan error),
		queue:               make(chan plugin.Notification),
	}, nil
}

// CurrentProfile will return the currently active Profile
func (c *Control) CurrentProfile() Profile {
	return c.Config.Profiles[c.currentProfileIndex]
}

func (c *Control) findProfileIndexWithName(name string) int {
	for i, p := range c.Profiles {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func (c *Control) setProfile(index int) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	nextProfile := c.Config.Profiles[index]

	// note: always set thermal throttle plan first, then override with user fan curve
	if err := c.setThrottlePlan(nextProfile); err != nil {
		return "", err
	}

	if err := c.setFanCurve(nextProfile); err != nil {
		return "", err
	}

	if _, err := c.Config.PowerCfg.Set(nextProfile.WindowsPowerPlan); err != nil {
		return "", err
	}

	c.currentProfileIndex = index

	return nextProfile.Name, nil
}

// SwitchToProfile will switch the profile with the given name
func (c *Control) SwitchToProfile(name string) (string, error) {
	nextIndex := c.findProfileIndexWithName(name)
	if nextIndex < 0 {
		return "", errors.New("Cannot find profile with name: " + name)
	}

	return c.setProfile(nextIndex)
}

// NextProfile will cycle to the next profile
func (c *Control) NextProfile(howMany int) (string, error) {
	nextIndex := (c.currentProfileIndex + howMany) % len(c.Config.Profiles)

	return c.setProfile(nextIndex)
}

func (c *Control) setThrottlePlan(profile Profile) error {
	args := make([]byte, 8)
	binary.LittleEndian.PutUint32(args[0:], atkacpi.DevsThrottleCtrl)
	binary.LittleEndian.PutUint32(args[4:], profile.ThrottlePlan)

	_, err := c.wmi.Evaluate(atkacpi.DEVS, args)
	if err != nil {
		return err
	}

	log.Printf("thermal: throttle plan set: 0x%x\n", profile.ThrottlePlan)

	return nil
}

func (c *Control) setFanCurve(profile Profile) error {

	if profile.CPUFanCurve != nil {
		cpuFanCurve := profile.CPUFanCurve.Bytes()

		if len(cpuFanCurve) != 16 {
			log.Printf("thermal: invalid cpu fan curve\n")
			return nil
		}

		cpuArgs := make([]byte, 20)
		binary.LittleEndian.PutUint32(cpuArgs[0:], atkacpi.DevsCPUFanCurve)
		copy(cpuArgs[4:], cpuFanCurve)

		if _, err := c.wmi.Evaluate(atkacpi.DEVS, cpuArgs); err != nil {
			return err
		}

		log.Printf("thermal: cpu fan curve set to %+v\n", cpuFanCurve)
	}

	time.Sleep(time.Millisecond * 250)

	if profile.GPUFanCurve != nil {
		gpuFanCurve := profile.GPUFanCurve.Bytes()

		if len(gpuFanCurve) != 16 {
			log.Printf("thermal: invalid gpu fan curve\n")
			return nil
		}

		gpuArgs := make([]byte, 20)
		binary.LittleEndian.PutUint32(gpuArgs[0:], atkacpi.DevsGPUFanCurve)
		copy(gpuArgs[4:], gpuFanCurve)

		if _, err := c.wmi.Evaluate(atkacpi.DEVS, gpuArgs); err != nil {
			return err
		}

		log.Printf("thermal: gpu fan curve set to %+v\n", gpuFanCurve)
	}

	return nil
}

// Initialize satisfies system/plugin.Plugin
func (c *Control) Initialize() error {
	return nil
}

func (c *Control) loop(haltCtx context.Context, cb chan<- plugin.Callback) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("thermal: loop panic %+v\n", err)
			c.errorCh <- err.(error)
		}
	}()

	for {
		select {
		case t := <-c.queue:
			switch t.Event {
			case plugin.EvtSentinelCycleThermalProfile:
				counter := t.Value.(int64)
				name, err := c.NextProfile(int(counter))
				message := fmt.Sprintf("Thermal plan changed to %s", name)
				if err != nil {
					log.Println(err)
					message = err.Error()
				}
				cb <- plugin.Callback{
					Event: plugin.CbNotifyToast,
					Value: util.Notification{
						Title:   "Toggle Thermal Plan",
						Message: message,
					},
				}
				cb <- plugin.Callback{
					Event: plugin.CbPersistConfig,
				}
			case plugin.EvtChargerPluggedIn, plugin.EvtChargerUnplugged:
				if !c.Config.AutoThermal {
					continue
				}
				var next string
				var err error
				var message string
				switch t.Event {
				case plugin.EvtChargerPluggedIn:
					next, err = c.SwitchToProfile(c.AutoThermalConfig.PluggedIn)
				case plugin.EvtChargerUnplugged:
					next, err = c.SwitchToProfile(c.AutoThermalConfig.Unplugged)
				default:
					continue
				}
				if err != nil {
					log.Println(err)
					message = err.Error()
				} else {
					message = fmt.Sprintf("Thermal plan changed to %s", next)
				}
				cb <- plugin.Callback{
					Event: plugin.CbNotifyToast,
					Value: util.Notification{
						Title:   "Automatic Thermal Plan Switching",
						Message: message,
					},
				}
			}
		case <-haltCtx.Done():
			log.Println("thermal: exiting Plugin run loop")
			return
		}
	}
}

// Run satisfies system/plugin.Plugin
func (c *Control) Run(haltCtx context.Context, cb chan<- plugin.Callback) <-chan error {
	log.Println("thermal: Starting queue loop")

	go c.loop(haltCtx, cb)

	return c.errorCh
}

// Notify satisfies system/plugin.Plugin
func (c *Control) Notify(t plugin.Notification) {
	c.queue <- t
}

var _ persist.Registry = &Control{}

// Name satisfies persist.Registry
func (c *Control) Name() string {
	return thermalPersistKey
}

// Value satisfies persist.Registry
func (c *Control) Value() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var buf bytes.Buffer
	name := c.CurrentProfile().Name
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(name); err != nil {
		return nil
	}
	return buf.Bytes()
}

// Load staisfies persist.Registry
func (c *Control) Load(v []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(v) == 0 {
		return nil
	}
	var name string
	buf := bytes.NewBuffer(v)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&name); err != nil {
		return err
	}
	for i, profile := range c.Profiles {
		if profile.Name == name {
			c.currentProfileIndex = i
			return nil
		}
	}
	return nil
}

// Apply satisfies persist.Registry
func (c *Control) Apply() error {
	c.currentProfileIndex-- // drcrement the index so we reapply the current one
	_, err := c.NextProfile(1)
	return err
}

// Close satisfied persist.Registry
func (c *Control) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.wmi.Close()
}
