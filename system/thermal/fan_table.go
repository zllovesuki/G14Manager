package thermal

// This is inspired by the atrofac utility (https://github.com/cronosun/atrofac)

import (
	"errors"
	"regexp"
	"strconv"
)

var (
	curveRe = regexp.MustCompile(`\s*(\d{1,3})c:(\d{1,3})%\s*`)
)

type fanTable struct {
	ByteTable []byte
}

func newFanTable(curve string) (*fanTable, error) {
	if len(curve) == 0 {
		return nil, nil
	}
	match := curveRe.FindAllStringSubmatch(curve, -1)
	t := &fanTable{
		ByteTable: make([]byte, 16),
	}
	if len(match) != 8 {
		return t, nil
	}
	for i, b := range match {
		degree, err := strconv.Atoi(b[1])
		if err != nil {
			return nil, errors.New("Parse error")
		}
		// TODO: validate degree value
		t.ByteTable[i] = byte(degree)

		fanPct, err := strconv.Atoi(b[2])
		if err != nil {
			return nil, errors.New("Parse error")
		}
		if fanPct < 0 || fanPct > 100 {
			return nil, errors.New("Percentage out of range")
		}
		t.ByteTable[i+8] = byte(fanPct)
	}
	return t, nil
}

// Bytes returns the binary presentation of the table
func (f *fanTable) Bytes() []byte {
	b := make([]byte, 16)
	copy(b, f.ByteTable)
	return b
}
