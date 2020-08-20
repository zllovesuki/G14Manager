package atkacpi

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/karalabe/usb"
)

const (
	vendorID  = 0x0b05
	productID = 0x1866
)

const (
	reportBufSize = 6
	reportID      = 0x5a
)

var (
	hidDevices = []string{
		"mi_02&col01", // Special key combo
		"mi_02&col02", // Volume up/down?
	}
)

// NewHidListener will read HID report and return key code to the channel
func NewHidListener(haltCtx context.Context, eventCh chan uint32) (map[string]usb.DeviceInfo, error) {
	devicesFound := make(map[string]usb.DeviceInfo)
	devices, err := usb.EnumerateHid(vendorID, productID)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		// TODO: make it less inefficient
		for _, hid := range hidDevices {
			if !strings.Contains(device.Path, hid) {
				continue
			}
			devicesFound[hid] = device
		}
	}
	if len(devicesFound) == 0 {
		return nil, fmt.Errorf("No devices found")
	}

	for _, device := range devicesFound {
		d, err := device.Open()
		if err != nil {
			return nil, err
		}
		go readDevice(haltCtx, eventCh, d)
	}
	return devicesFound, nil
}

func readDevice(haltCtx context.Context, eventCh chan uint32, dev usb.Device) {
	for {
		select {
		case <-haltCtx.Done():
			return
		default:
		}
		buf := make([]byte, reportBufSize)
		buf[0] = reportID
		_, err := dev.Read(buf)
		if err != nil {
			log.Fatalln(err)
		}
		if buf[1] > 0 && buf[1] < 236 {
			eventCh <- uint32(buf[1])
		}
	}
}
