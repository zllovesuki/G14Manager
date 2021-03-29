package thermal

// This is inspired by the atrofac utility (https://github.com/cronosun/atrofac)

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var (
	curveRe = regexp.MustCompile(`\s*(\d{1,3})c:(\d{1,3})%\s*`)
)

type FanTable struct {
	ByteTable []byte
}

func NewFanTable(curve string) (*FanTable, error) {
	if len(curve) == 0 {
		return nil, nil
	}
	match := curveRe.FindAllStringSubmatch(curve, -1)
	t := &FanTable{
		ByteTable: make([]byte, 16),
	}
	if len(match) != 8 {
		return t, nil
	}
	for i, b := range match {
		degree, err := strconv.Atoi(b[1])
		if err != nil {
			return nil, errors.New("Temperature parse error")
		}
		if degree < 20 {
			return nil, errors.New("Temperature must be greater than or equal to 20C")
		}
		t.ByteTable[i] = byte(degree)

		fanPct, err := strconv.Atoi(b[2])
		if err != nil {
			return nil, errors.New("Fan percentage parse error")
		}
		if fanPct < 0 || fanPct > 100 {
			return nil, errors.New("Fan percentage out of range")
		}
		t.ByteTable[i+8] = byte(fanPct)
	}
	return t, nil
}

// Bytes returns the binary representation of the table
func (f *FanTable) Bytes() []byte {
	if f == nil {
		return nil
	}
	b := make([]byte, 16)
	copy(b, f.ByteTable)
	return b
}

// String() returns the original fan curve in string reprensentation
func (f *FanTable) String() string {
	if f == nil {
		return ""
	}
	b := f.ByteTable
	return fmt.Sprintf("%dc:%d%%,%dc:%d%%,%dc:%d%%,%dc:%d%%,%dc:%d%%,%dc:%d%%,%dc:%d%%,%dc:%d%%",
		b[0],
		b[8],
		b[1],
		b[9],
		b[2],
		b[10],
		b[3],
		b[11],
		b[4],
		b[12],
		b[5],
		b[13],
		b[6],
		b[14],
		b[7],
		b[15],
	)
}
