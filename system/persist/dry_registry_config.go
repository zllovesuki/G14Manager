package persist

import "log"

type dryRegistryHelper struct {
	ConfigRegistry
}

var _ ConfigRegistry = &dryRegistryHelper{}

// NewDryRegistryHelper returns a helper to persist config to the Registry but without actual IO to save
func NewDryRegistryHelper() (ConfigRegistry, error) {
	helper, _ := NewRegistryConfigHelper()
	log.Println("[dry run] persist: initializing Registry without save IOs")
	return &dryRegistryHelper{
		ConfigRegistry: helper,
	}, nil
}

// Save will do nothing
func (d *dryRegistryHelper) Save() error {
	return nil
}
