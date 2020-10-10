package persist

import "log"

type dryRegistryHelper struct {
	helper ConfigRegistry
}

var _ ConfigRegistry = &dryRegistryHelper{}

// NewDryRegistryHelper returns a helper to persist config to the Registry but without actual IO to save
func NewDryRegistryHelper() (ConfigRegistry, error) {
	helper, _ := NewRegistryHelper()
	log.Println("[dry run] persist: initializing Registry without save IOs")
	return &dryRegistryHelper{
		helper: helper,
	}, nil
}

// Register will add the config to the list
func (d *dryRegistryHelper) Register(config Registry) {
	d.helper.Register(config)
}

// Load will retrive and populate configs from Registry
func (d *dryRegistryHelper) Load() error {
	return d.helper.Load()
}

// Save will do nothing
func (d *dryRegistryHelper) Save() error {
	return nil
}

// Apply will apply each config accordingly. This is usually called after Load()
func (d *dryRegistryHelper) Apply() error {
	return d.helper.Apply()
}

// Close will release resources of each config
func (d *dryRegistryHelper) Close() {
	d.helper.Close()
}
