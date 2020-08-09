package thermal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThermalPersist(t *testing.T) {
	defaultProfiles := GetDefaultThermalProfiles()
	thermal := Thermal{
		currentProfileIndex: 10,
		Config: Config{
			Profiles: defaultProfiles,
		},
	}

	require.NotEmpty(t, thermal.Name())

	b := thermal.Value()
	require.NotEmpty(t, b)

	loaded := Thermal{}

	require.NoError(t, loaded.Load(b))
	require.EqualValues(t, thermal.Config.Profiles, loaded.Config.Profiles)
	require.EqualValues(t, thermal.currentProfileIndex, loaded.currentProfileIndex)

}
