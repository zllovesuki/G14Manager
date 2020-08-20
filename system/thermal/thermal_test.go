package thermal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThermalPersist(t *testing.T) {
	defaultProfiles := GetDefaultThermalProfiles()
	thermal := Thermal{
		currentProfileIndex: 1,
		Config: Config{
			Profiles: defaultProfiles,
		},
	}

	require.NotEmpty(t, thermal.Name())

	b := thermal.Value()
	require.NotEmpty(t, b)

	loaded := Thermal{
		Config: Config{
			Profiles: defaultProfiles,
		},
	}

	require.NoError(t, loaded.Load(b))
	require.Equal(t, thermal.currentProfileIndex, loaded.currentProfileIndex)
	require.Equal(t, defaultProfiles[1].Name, thermal.CurrentProfile().Name)

}
