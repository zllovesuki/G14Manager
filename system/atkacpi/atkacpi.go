package atkacpi

import (
	"log"

	"github.com/zllovesuki/ROGManager/system/device"
)

const devicePath = `\\.\ATKACPI`

type ATKControl struct {
	device *device.Control
}

func NewAtkControl(controlCode uint32) (*ATKControl, error) {
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
