package thermal

// This is inspired by the atrofac utility (https://github.com/cronosun/atrofac)

import (
	"log"
	"strings"
	"time"

	"github.com/zllovesuki/ROGManager/system/atkacpi"
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

type thermalProfile struct {
	name             string
	windowsPowerPlan string
	throttlePlan     byte
	cpuFanCurve      *FanTable
	gpuFanCurve      *FanTable
}

type Thermal struct {
	controlInterface *atkacpi.ATKControl
	powercfg         *Powercfg
	profiles         []thermalProfile
	currentProfile   int
}

func NewThermal() (*Thermal, error) {
	ctrl, err := atkacpi.NewAtkControl()
	if err != nil {
		return nil, err
	}
	power, err := NewPowerCfg()
	if err != nil {
		return nil, err
	}
	return &Thermal{
		controlInterface: ctrl,
		powercfg:         power,
		profiles:         make([]thermalProfile, 0),
		currentProfile:   -1,
	}, nil
}

func (t *Thermal) Default() {
	defaults := []struct {
		name             string
		windowsPowerPlan string
		throttlePlan     byte
		cpuFanCurve      string
		gpuFanCurve      string
	}{
		{
			name:             "Fanless",
			windowsPowerPlan: "Power saver",
			throttlePlan:     throttlePlanSilent,
			cpuFanCurve:      "39c:0%,49c:0%,59c:0%,69c:0%,79c:31%,89c:49%,99c:56%,109c:56%",
			gpuFanCurve:      "39c:0%,49c:0%,59c:0%,69c:0%,79c:34%,89c:51%,99c:61%,109c:61%",
		},
		{
			name:             "Quiet",
			windowsPowerPlan: "Power saver",
			throttlePlan:     throttlePlanSilent,
			cpuFanCurve:      "39c:10%,49c:10%,59c:10%,69c:10%,79c:31%,89c:49%,99c:56%,109c:56%",
			gpuFanCurve:      "39c:0%,49c:0%,59c:0%,69c:0%,79c:34%,89c:51%,99c:61%,109c:61%",
		},
		{
			name:             "Silent",
			windowsPowerPlan: "Power saver",
			throttlePlan:     throttlePlanSilent,
		},
		{
			name:             "Performance",
			windowsPowerPlan: "High performance",
			throttlePlan:     throttlePlanPerformance,
		},
		/*{
			name:             "Turbo",
			windowsPowerPlan: "High performance",
			throttlePlan:     throttlePlanTurbo,
		},*/
	}
	for _, d := range defaults {
		var cpuTable, gpuTable *FanTable
		var err error
		profile := thermalProfile{
			name:             d.name,
			throttlePlan:     d.throttlePlan,
			windowsPowerPlan: strings.ToLower(d.windowsPowerPlan),
		}
		if d.cpuFanCurve != "" {
			cpuTable, err = NewFanTable(d.cpuFanCurve)
			if err != nil {
				panic(err)
			}
			profile.cpuFanCurve = cpuTable
		}
		if d.gpuFanCurve != "" {
			gpuTable, err = NewFanTable(d.gpuFanCurve)
			if err != nil {
				panic(err)
			}
			profile.gpuFanCurve = gpuTable
		}
		t.profiles = append(t.profiles, profile)
	}
}

func (t *Thermal) NextProfile() (string, error) {
	next := (t.currentProfile + 1) % len(t.profiles)
	profile := t.profiles[next]
	// note: always set thermal throttle plan first
	if err := t.setPowerPlan(profile); err != nil {
		return "", err
	}
	log.Println("thermal throttle plan set")
	time.Sleep(time.Millisecond * 25)
	if err := t.setFanCurve(profile); err != nil {
		return "", err
	}
	log.Println("fan profile set")
	if _, err := t.powercfg.Set(profile.windowsPowerPlan); err != nil {
		return "", err
	}
	log.Println("windows power plan set")
	t.currentProfile = next
	return profile.name, nil
}

func (t *Thermal) setPowerPlan(profile thermalProfile) error {
	inputBuf := make([]byte, 16)
	copy(inputBuf, atkacpi.ThrottlePlanControlBuffer)

	inputBuf[atkacpi.ThrottlePlanControlByteIndex] = profile.throttlePlan

	_, err := t.controlInterface.Write(inputBuf)
	if err != nil {
		return err
	}

	return nil
}

func (t *Thermal) setFanCurve(profile thermalProfile) error {
	if profile.cpuFanCurve != nil {
		if err := t.setFan(cpuFanCurveDevice, profile.cpuFanCurve.Bytes()); err != nil {
			return err
		}
	}
	if profile.gpuFanCurve != nil {
		if err := t.setFan(gpuFanCurveDevice, profile.gpuFanCurve.Bytes()); err != nil {
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

	inputBuf := make([]byte, 28)
	copy(inputBuf, atkacpi.FanCurveControlBuffer)

	inputBuf[atkacpi.FanCurveControlByteIndex] = device
	copy(inputBuf[12:], curve)

	_, err := t.controlInterface.Write(inputBuf)
	if err != nil {
		return err
	}

	return nil
}

func (t *Thermal) setCPUFan(curve []byte) error {
	return t.setFan(cpuFanCurveDevice, curve)
}

func (t *Thermal) setGPUFan(curve []byte) error {
	return t.setFan(gpuFanCurveDevice, curve)
}
