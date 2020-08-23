package persist

import (
	"testing"

	"golang.org/x/sys/windows/registry"

	"github.com/stretchr/testify/require"
)

type mockConfig struct {
	bytes []byte
}

func (m *mockConfig) Name() string        { return "MockConfig" }
func (m *mockConfig) Value() []byte       { return m.bytes }
func (m *mockConfig) Load(v []byte) error { m.bytes = v; return nil }
func (m *mockConfig) Apply() error        { return nil }
func (m *mockConfig) Close() error        { return nil }

var _ Registry = &mockConfig{}

const (
	testRegistryKey  = registry.CURRENT_USER
	testRegistryPath = `SOFTWARE/G14ManagerTest`
)

func TestPersistToRegistry(t *testing.T) {
	expectedBytes := []byte{1, 2, 3, 4, 5, 6}
	h := RegistryHelper{
		configs: make(map[string]Registry),
		key:     testRegistryKey,
		path:    testRegistryPath,
	}

	m := mockConfig{
		bytes: expectedBytes,
	}
	h.Register(&m)

	err := h.Save()
	require.NoError(t, err)

	hL := RegistryHelper{
		configs: make(map[string]Registry),
		key:     registry.CURRENT_USER,
		path:    `SOFTWARE/G14ManagerTest`,
	}

	m = mockConfig{}
	hL.Register(&m)

	err = hL.Load()
	require.NoError(t, err)

	require.EqualValues(t, expectedBytes, m.bytes)

	require.NoError(t, registry.DeleteKey(testRegistryKey, testRegistryPath))

}
