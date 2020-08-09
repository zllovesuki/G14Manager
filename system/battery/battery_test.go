package battery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatteryPersist(t *testing.T) {

	expectedLimit := uint8(80)

	limit := &ChargeLimit{
		currentLimit: expectedLimit,
	}
	require.NotEmpty(t, limit.Name())

	b := limit.Value()
	require.NotEmpty(t, b)

	loaded := ChargeLimit{}

	require.NoError(t, loaded.Load(b))
	require.Equal(t, expectedLimit, loaded.currentLimit)
}
