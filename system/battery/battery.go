package battery

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/persist"
)

const (
	persistKey = "BatteryChargeLimit"
)

// ChargeLimit allows you to limit the full charge percentage on your laptop
type ChargeLimit struct {
	wmi          atkacpi.WMI
	currentLimit uint8
	mu           sync.RWMutex
}

// NewChargeLimit initializes the control interface and returns an instance of ChargeLimit
func NewChargeLimit(wmi atkacpi.WMI) (*ChargeLimit, error) {
	return &ChargeLimit{
		wmi:          wmi,
		currentLimit: 80,
	}, nil
}

// Set will write to ACPI and set the battery charge limit in percentage. Note that the minimum percentage is 40
func (c *ChargeLimit) Set(pct uint8) error {
	if pct < 40 || pct > 100 {
		return errors.New("charge limit percentage must be between 40 and 100, inclusive")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	args := make([]byte, 8)
	binary.LittleEndian.PutUint32(args[0:], atkacpi.DevsBatteryChargeLimit)
	binary.LittleEndian.PutUint32(args[4:], uint32(pct))

	_, err := c.wmi.Evaluate(atkacpi.DEVS, args)
	if err != nil {
		return err
	}
	c.currentLimit = pct
	return nil
}

func (c *ChargeLimit) CurrentLimit() uint8 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.currentLimit
}

var _ persist.Registry = &ChargeLimit{}

// Name satisfies persist.Registry
func (c *ChargeLimit) Name() string {
	return persistKey
}

// Value satisfies persist.Registry
func (c *ChargeLimit) Value() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(c.currentLimit))
	return b
}

// Load satisfies persist.Registry
func (c *ChargeLimit) Load(v []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.wmi.Close()
}
