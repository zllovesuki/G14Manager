package battery

import (
	"encoding/binary"
	"errors"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/ioctl"
	"github.com/zllovesuki/G14Manager/system/persist"
)

const (
	persistKey = "BatteryChargeLimit"
)

// ChargeLimit allows you to limit the full charge percentage on your laptop
type ChargeLimit struct {
	currentLimit uint8
}

// NewChargeLimit initializes the control interface and returns an instance of ChargeLimit
func NewChargeLimit() (*ChargeLimit, error) {
	return &ChargeLimit{
		currentLimit: 80,
	}, nil
}

// Set will write to ACPI and set the battery charge limit in percentage. Note that the minimum percentage is 40
func (c *ChargeLimit) Set(pct uint8) error {
	if pct <= 40 || pct >= 100 {
		return errors.New("charge limit percentage must be between 40 and 100, inclusive")
	}
	ctrl, err := atkacpi.NewAtkControl(ioctl.ATK_ACPI_WMIFUNCTION)
	if err != nil {
		return err
	}
	defer ctrl.Close()

	inputBuf := make([]byte, atkacpi.BatteryChargeLimitInputBufferLength)
	copy(inputBuf, atkacpi.BatteryChargeLimitControlBuffer)
	inputBuf[atkacpi.BatteryChargeLimitControlByteIndex] = byte(pct)

	_, err = ctrl.Write(inputBuf)
	if err != nil {
		return err
	}
	c.currentLimit = pct
	return nil
}

var _ persist.Registry = &ChargeLimit{}

// Name satisfies persist.Registry
func (c *ChargeLimit) Name() string {
	return persistKey
}

// Value satisfies persist.Registry
func (c *ChargeLimit) Value() []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(c.currentLimit))
	return b
}

// Load satisfies persist.Registry
func (c *ChargeLimit) Load(v []byte) error {
	if len(v) == 0 {
		return nil
	}
	c.currentLimit = uint8(binary.LittleEndian.Uint16(v))
	return nil
}

// Apply satisfies persist.Registry
func (c *ChargeLimit) Apply() error {
	return c.Set(c.currentLimit)
}

// Close satisfied persist.Registry
func (c *ChargeLimit) Close() error {
	return nil
}
