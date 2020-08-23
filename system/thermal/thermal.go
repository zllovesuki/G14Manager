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
	"encoding/gob"
	"errors"
	"log"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/persist"
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

// Thermal defines contains the Windows Power Option and list of thermal profiles
type Thermal struct {
	wmi                 atkacpi.WMI
	currentProfileIndex int
	Config
}

// Config defines the entry point for Windows Power Option and a list of thermal profiles
type Config struct {
	PowerCfg *PowerCfg
	Profiles []Profile
}

// NewThermal allows you to cycle to the next thermal profile
func NewThermal(conf Config) (*Thermal, error) {
	if conf.PowerCfg == nil {
		return nil, errors.New("nil PowerCfg is invalid")
	}
	if len(conf.Profiles) == 0 {
		return nil, errors.New("empty Profiles is invalid")
	}
	wmi, err := atkacpi.NewWMI()
	if err != nil {
		return nil, err
	}

	return &Thermal{
		wmi:                 wmi,
		currentProfileIndex: 0,
		Config:              conf,
	}, nil
}

// CurrentProfile will return the currently active Profile
func (t *Thermal) CurrentProfile() Profile {
	return t.Config.Profiles[t.currentProfileIndex]
}

// NextProfile will cycle to the next profile
func (t *Thermal) NextProfile(howMany int) (string, error) {
	nextIndex := (t.currentProfileIndex + howMany) % len(t.Config.Profiles)
	nextProfile := t.Config.Profiles[nextIndex]

	// note: always set thermal throttle plan first, then override with user fan curve
	if err := t.setThrottlePlan(nextProfile); err != nil {
		return "", err
	}

	if err := t.setFanCurve(nextProfile); err != nil {
		return "", err
	}

	if _, err := t.Config.PowerCfg.Set(nextProfile.WindowsPowerPlan); err != nil {
		return "", err
	}

	t.currentProfileIndex = nextIndex

	return nextProfile.Name, nil
}

func (t *Thermal) setThrottlePlan(profile Profile) error {
	args := make([]byte, 0, 8)
	args = append(args, util.Uint32ToLEBuffer(atkacpi.DevsThrottleCtrl)...)
	args = append(args, util.Uint32ToLEBuffer(profile.ThrottlePlan)...)

	_, err := t.wmi.Evaluate(atkacpi.DEVS, args)
	if err != nil {
		return err
	}

	log.Printf("thermal: throttle plan set: 0x%x\n", profile.ThrottlePlan)

	return nil
}

func (t *Thermal) setFanCurve(profile Profile) error {

	if profile.CPUFanCurve != nil {
		cpuFanCurve := profile.CPUFanCurve.Bytes()

		if len(cpuFanCurve) != 16 {
			log.Printf("thermal: invalid cpu fan curve\n")
			return nil
		}

		cpuArgs := make([]byte, 0, 20)
		cpuArgs = append(cpuArgs, util.Uint32ToLEBuffer(atkacpi.DevsCPUFanCurve)...)
		copy(cpuArgs[4:], cpuFanCurve)

		if _, err := t.wmi.Evaluate(atkacpi.DEVS, cpuArgs); err != nil {
			return err
		}

		log.Printf("thermal: cpu fan curve set to %+v\n", cpuFanCurve)
	}

	if profile.GPUFanCurve != nil {
		gpuFanCurve := profile.GPUFanCurve.Bytes()

		if len(gpuFanCurve) != 16 {
			log.Printf("thermal: invalid gpu fan curve\n")
			return nil
		}

		gpuArgs := make([]byte, 0, 20)
		gpuArgs = append(gpuArgs, util.Uint32ToLEBuffer(atkacpi.DevsGPUFanCurve)...)
		copy(gpuArgs[4:], gpuFanCurve)

		if _, err := t.wmi.Evaluate(atkacpi.DEVS, gpuArgs); err != nil {
			return err
		}

		log.Printf("thermal: gpu fan curve set to %+v\n", gpuFanCurve)
	}

	return nil
}

var _ persist.Registry = &Thermal{}

// Name satisfies persist.Registry
func (t *Thermal) Name() string {
	return thermalPersistKey
}

// Value satisfies persist.Registry
func (t *Thermal) Value() []byte {
	var buf bytes.Buffer
	name := t.CurrentProfile().Name
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(name); err != nil {
		return nil
	}
	return buf.Bytes()
}

// Load staisfies persist.Registry
func (t *Thermal) Load(v []byte) error {
	if len(v) == 0 {
		return nil
	}
	var name string
	buf := bytes.NewBuffer(v)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&name); err != nil {
		return err
	}
	for i, profile := range t.Profiles {
		if profile.Name == name {
			t.currentProfileIndex = i
			return nil
		}
	}
	return nil
}

// Apply satisfies persist.Registry
func (t *Thermal) Apply() error {
	t.currentProfileIndex-- // drcrement the index so we reapply the current one
	_, err := t.NextProfile(1)
	return err
}

// Close satisfied persist.Registry
func (t *Thermal) Close() error {
	return nil
}
