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
		"col01", // Special key combo
		"col02", // Volume up/down?
	}
)

// NewHidListener will read HID report and return key code to the channel
func NewHidListener(haltCtx context.Context, eventCh chan uint32) error {
	devicesFound := make([]usb.DeviceInfo, 0, 2)
	devices, err := usb.EnumerateHid(vendorID, productID)
	if err != nil {
		return err
	}

	for _, device := range devices {
		// TODO: make it less inefficient
		log.Println(device.Path)
		for _, hid := range hidDevices {
			if !strings.Contains(device.Path, hid) {
				continue
			}
			devicesFound = append(devicesFound, device)
		}
	}
	if len(devicesFound) == 0 {
		return fmt.Errorf("No devices found")
	}

	for _, device := range devicesFound {
		d, err := device.Open()
		if err != nil {
			return err
		}
		go readDevice(haltCtx, eventCh, d)
	}
	return nil
}

func readDevice(haltCtx context.Context, eventCh chan uint32, dev usb.Device) {
	for {
		select {
		case <-haltCtx.Done():
			return
		default:
			buf := make([]byte, reportBufSize)
			buf[0] = reportID
			_, err := dev.Read(buf)
			if err != nil {
				log.Fatalln(err)
			}
			if buf[1] > 0 {
				eventCh <- uint32(buf[1])
			}
		}
	}
}
