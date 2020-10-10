package atkacpi

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/zllovesuki/G14Manager/system/ioctl"
)

type dryWmi struct{}

var _ WMI = &dryWmi{}

// NewDryWMI returns an WMI without actual IOs
func NewDryWMI() (WMI, error) {
	return &dryWmi{}, nil
}

func (d *dryWmi) Evaluate(id Method, args []byte) ([]byte, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("args should have at least one parameter")
	}

	acpiBuf := make([]byte, 8)
	binary.LittleEndian.PutUint32(acpiBuf[0:], uint32(id))
	binary.LittleEndian.PutUint32(acpiBuf[4:], uint32(len(args)))
	log.Printf("[dry run] device: (%d) execute input buffer [0:8]: %+v\n", ioctl.ATK_ACPI_WMIFUNCTION, acpiBuf)

	return make([]byte, 1024), nil
}

func (d *dryWmi) Close() error {
	return nil
}
