package shared

type Features struct {
	AutoThermal AutoThermal
	FnRemap     map[uint32]uint16
	RogRemap    []string
}

type AutoThermal struct {
	Enabled   bool
	PluggedIn string
	Unplugged string
}
