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

type FanTable struct {
	byteTable []byte
}

func NewFanTable(curve string) (*FanTable, error) {
	if len(curve) == 0 {
		return nil, nil
	}
	match := curveRe.FindAllStringSubmatch(curve, -1)
	t := &FanTable{
		byteTable: make([]byte, 16),
	}
	if len(match) != 8 {
		return t, nil
	}
	for i, b := range match {
		degree, err := strconv.Atoi(b[1])
		if err != nil {
			return nil, errors.New("Parse error")
		}
		t.byteTable[i] = byte(degree)
		fanPct, err := strconv.Atoi(b[2])
		if err != nil {
			return nil, errors.New("Parse error")
		}
		t.byteTable[i+8] = byte(fanPct)
	}
	return t, nil
}

func (f *FanTable) Bytes() []byte {
	b := make([]byte, 16)
	copy(b, f.byteTable)
	return b
}
