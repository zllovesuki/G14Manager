package main

import (
	"errors"

	"github.com/zllovesuki/ROGManager/system/atkacpi"
)

type ChargeLimit struct {
	controlInterface *atkacpi.ATKControl
	currentLimit     int
}

func NewChargeLimit() (*ChargeLimit, error) {
	ctrl, err := atkacpi.NewAtkControl(atkacpi.WriteControlCode)
	if err != nil {
		return nil, err
	}
	return &ChargeLimit{
		controlInterface: ctrl,
		currentLimit:     60,
	}, nil
}

func (c *ChargeLimit) Set(pct int) error {
	if pct <= 40 || pct >= 100 {
		return errors.New("charge limit percentage must be between 40 and 100, inclusive")
	}
	inputBuf := make([]byte, atkacpi.BatteryChargeLimitInputBufferLength)
	copy(inputBuf, atkacpi.BatteryChargeLimitControlBuffer)
	inputBuf[atkacpi.BatteryChargeLimitControlByteIndex] = byte(pct)

	_, err := c.controlInterface.Write(inputBuf)
	if err != nil {
		return err
	}
	c.currentLimit = pct
	return nil
}

func main() {
	limit, err := NewChargeLimit()
	if err != nil {
		panic(err)
	}
	if err := limit.Set(60); err != nil {
		panic(err)
	}
}
