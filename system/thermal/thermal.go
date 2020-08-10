package thermal

// This is inspired by the atrofac utility (https://github.com/cronosun/atrofac)

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"time"

	"github.com/zllovesuki/ROGManager/system/atkacpi"
	"github.com/zllovesuki/ROGManager/system/persist"
)

const (
	thermalPersistKey = "ThermalProfile"
)

const (
	throttlePlanPerformance = byte(0x00)
	throttlePlanTurbo       = byte(0x01)
	throttlePlanSilent      = byte(0x02)
)

const (
	cpuFanCurveDevice = byte(0x24)
	gpuFanCurveDevice = byte(0x25)
)

// Profile contain each thermal profile definition
// TODO: Revisit this
type Profile struct {
	Name             string
	WindowsPowerPlan string
	ThrottlePlan     byte
	CPUFanCurve      *fanTable
	GPUFanCurve      *fanTable
}

// Thermal defines contains the Windows Power Option and list of thermal profiles
type Thermal struct {
	controlInterface    *atkacpi.ATKControl
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

	ctrl, err := atkacpi.NewAtkControl(atkacpi.WriteControlCode)
	if err != nil {
		return nil, err
	}
	return &Thermal{
		controlInterface:    ctrl,
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

	// note: always set thermal throttle plan first
	if err := t.setPowerPlan(nextProfile); err != nil {
		return "", err
	}
	time.Sleep(time.Millisecond * 25)

	if err := t.setFanCurve(nextProfile); err != nil {
		return "", err
	}

	if _, err := t.Config.PowerCfg.Set(nextProfile.WindowsPowerPlan); err != nil {
		return "", err
	}

	t.currentProfileIndex = nextIndex

	return nextProfile.Name, nil
}

func (t *Thermal) setPowerPlan(profile Profile) error {
	inputBuf := make([]byte, atkacpi.ThrottlePlanInputBufferLength)
	copy(inputBuf, atkacpi.ThrottlePlanControlBuffer)

	inputBuf[atkacpi.ThrottlePlanControlByteIndex] = profile.ThrottlePlan

	_, err := t.controlInterface.Write(inputBuf)
	if err != nil {
		return err
	}

	log.Printf("thermal throttle plan set: 0x%x\n", profile.ThrottlePlan)

	return nil
}

func (t *Thermal) setFanCurve(profile Profile) error {
	if profile.CPUFanCurve != nil {
		if err := t.setFan(cpuFanCurveDevice, profile.CPUFanCurve.Bytes()); err != nil {
			return err
		}
	}
	if profile.GPUFanCurve != nil {
		if err := t.setFan(gpuFanCurveDevice, profile.GPUFanCurve.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (t *Thermal) setFan(device byte, curve []byte) error {
	if len(curve) != 16 {
		log.Println("invalid found, skipping")
		return nil
	}

	inputBuf := make([]byte, atkacpi.FanCurveInputBufferLength)
	copy(inputBuf, atkacpi.FanCurveControlBuffer)

	inputBuf[atkacpi.FanCurveDeviceControlByteIndex] = device
	copy(inputBuf[atkacpi.FanCurveControlByteStartIndex:], curve)

	_, err := t.controlInterface.Write(inputBuf)
	if err != nil {
		return err
	}

	log.Printf("device 0x%x curve set to %+v\n", device, curve)

	return nil
}

func (t *Thermal) setCPUFan(curve []byte) error {
	return t.setFan(cpuFanCurveDevice, curve)
}

func (t *Thermal) setGPUFan(curve []byte) error {
	return t.setFan(gpuFanCurveDevice, curve)
}

var _ persist.Registry = &Thermal{}

type persistThermal struct {
	Index    int
	Profiles []Profile
}

// Name satisfies persist.Registry
func (t *Thermal) Name() string {
	return thermalPersistKey
}

// Value satisfies persist.Registry
func (t *Thermal) Value() []byte {
	s := persistThermal{
		Index:    t.currentProfileIndex,
		Profiles: t.Config.Profiles,
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil
	}
	return buf.Bytes()
}

// Load staisfies persist.Registry
func (t *Thermal) Load(v []byte) error {
	if len(v) == 0 {
		return nil
	}
	s := persistThermal{}
	buf := bytes.NewBuffer(v)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&s); err != nil {
		return err
	}
	t.currentProfileIndex = s.Index
	// TODO: remove this once we allow user to specify their curve
	t.Config.Profiles = s.Profiles
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
	return t.controlInterface.Close()
}
