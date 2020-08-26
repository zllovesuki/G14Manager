package power

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersistPowerCfg(t *testing.T) {
	expectedPlan := plan{
		Name:         "name",
		GUID:         "guid",
		OriginalName: "Name",
	}

	cfg := &Cfg{
		activePlan: expectedPlan,
	}
	require.NotEmpty(t, cfg.Name())

	b := cfg.Value()
	require.NotEmpty(t, b)

	loaded := Cfg{}

	require.NoError(t, loaded.Load(b))
	require.EqualValues(t, expectedPlan, loaded.activePlan)
}
