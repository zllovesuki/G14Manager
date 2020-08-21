package persist

import (
	"log"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	registryKey  = registry.LOCAL_MACHINE
	registryPath = `SOFTWARE\G14Manager`
)

// RegistryHelper contains a list of configurations to be loaded, saved, and applied
type RegistryHelper struct {
	configs map[string]Registry
	key     registry.Key
	path    string
}

// NewRegistryHelper returns a helper to persist config to the Registry
func NewRegistryHelper() (*RegistryHelper, error) {
	return &RegistryHelper{
		configs: make(map[string]Registry),
		key:     registryKey,
		path:    registryPath,
	}, nil
}

// Register will add the config to the list
func (h *RegistryHelper) Register(config Registry) {
	h.configs[config.Name()] = config
}

// Load will retrive and populate configs from Registry
func (h *RegistryHelper) Load() error {
	key, exists, err := registry.CreateKey(
		h.key,
		h.path,
		registry.ALL_ACCESS,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	if !exists {
		// nothing to load
		return nil
	}

	for _, config := range h.configs {
		log.Printf("persist: loading \"%s\" from the Registry\n", config.Name())
		v, _, err := key.GetBinaryValue(config.Name())
		if err != nil && err != registry.ErrNotExist {
			log.Printf("persist: error loading \"%s\" from the Registry: %s\n", config.Name(), err)
			return err
		}
		config.Load(v)
	}

	return nil
}

// Save will persist all the configs to Registry as binary values
func (h *RegistryHelper) Save() error {
	key, _, err := registry.CreateKey(
		h.key,
		h.path,
		registry.ALL_ACCESS,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	for _, config := range h.configs {
		log.Printf("persist: saving \"%s\" to the Registry\n", config.Name())
		err := key.SetBinaryValue(config.Name(), config.Value())
		if err != nil {
			log.Printf("persist: error saving \"%s\" to the Registry: %s\n", config.Name(), err)
			return err
		}
	}

	return nil
}

// Apply will apply each config accordingly. This is usually called after Load()
func (h *RegistryHelper) Apply() error {
	for _, config := range h.configs {
		log.Printf("persist: applying \"%s\" config\n", config.Name())
		err := config.Apply()
		if err != nil {
			log.Printf("persist: error applying \"%s\": %s\n", config.Name(), err)
			return err
		}
		time.Sleep(time.Millisecond * 25) // allow time for hardware configuration to propagate
	}

	return nil
}

// Close will release resources of each config
func (h *RegistryHelper) Close() {
	for _, config := range h.configs {
		log.Printf("persist: closing \"%s\"\n", config.Name())
		err := config.Close()
		if err != nil {
			log.Printf("persist: error closing \"%s\": %s\n", config.Name(), err)
		}
	}
}
