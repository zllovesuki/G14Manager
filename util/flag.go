package util

import (
	"fmt"
	"strings"
)

type ArrayFlags []string

func (r *ArrayFlags) String() string {
	str := make([]string, 0, len(*r))
	for i := 0; i < len(*r); i++ {
		str = append(str, fmt.Sprintf("%d: %s", i+1, str[i]))
	}
	return strings.Join(str, ";")
}

func (r *ArrayFlags) Set(value string) error {
	*r = append(*r, value)
	return nil
}
