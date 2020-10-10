package persist

// Registry provides an interface for different configurations to save to Registry and reload
type Registry interface {
	// Name should return the name of the configuration
	Name() string
	// Value should return the values to be persisted in binary format
	Value() []byte
	// Load should load the binary returned by Value() and populate the configuration
	Load([]byte) error
	// Apply should re-apply configurations
	Apply() error
	// Close should handle graceful shutdown (e.g. closing sockets, etc)
	Close() error
}

// ConfigRegistry defines an interface to save/load/apply configs in each Registry
type ConfigRegistry interface {
	// Register should register the given Registry to be load/save/apply
	Register(registry Registry)
	// Load should restore the configurations from the backing storage
	Load() error
	// Save should save the configurations to the backing storage
	Save() error
	// Apply should instruct each Registry to apply the configurations
	Apply() error
	// Close should instruct each Registry to close/clean up
	Close()
}
