package atkacpi

import (
	"log"

	"github.com/zllovesuki/ROGManager/system/device"
)

const devicePath = `\\.\ATKACPI`
const controlCode = uint32(2237452)

const (
	BatteryChargeLimitControlByteIndex = 12
	ThrottlePlanControlByteIndex       = 12
	FanCurveControlByteIndex           = 8
)

var (
	BatteryChargeLimitControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, 0x08, 0x00, 0x00, 0x00,
		0x57, 0x00, 0x12, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	ThrottlePlanControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, 0x08, 0x00, 0x00, 0x00,
		0x75, 0x00, 0x12, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	FanCurveControlBuffer = []byte{
		0x44, 0x45, 0x56, 0x53, 0x14, 0x00, 0x00,
		0x00, 0xFF, 0x00, 0x11, 0x00, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
)

type ATKControl struct {
	device *device.Control
}

func NewAtkControl() (*ATKControl, error) {
	device, err := device.NewControl(devicePath, controlCode)
	if err != nil {
		return nil, err
	}
	return &ATKControl{
		device: device,
	}, nil
}

func (a *ATKControl) Write(buf []byte) (n int, err error) { // implements io.Writer
	log.Printf("device %s input buffer: %+v\n", devicePath, buf)
	result, err := a.device.Write(buf)
	if err != nil {
		return 0, err
	}
	log.Printf("device %s control result: %+v\n", devicePath, result)
	return int(result.Written), nil
}
