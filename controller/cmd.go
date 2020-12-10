package controller

import (
	"fmt"

	"github.com/zllovesuki/G14Manager/cxx/plugin/keyboard"
	"github.com/zllovesuki/G14Manager/cxx/plugin/volume"
	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/battery"
	kb "github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/plugin"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/util"
)

const defaultCommandWithArgs = "Taskmgr.exe"

// RunConfig contains the start up configuration for the controller
type RunConfig struct {
	RogRemap        util.ArrayFlags
	DryRun          bool
	EnabledFeatures Features
	LogoPath        string
}

type Dependencies struct {
	WMI            atkacpi.WMI
	Keyboard       *keyboard.Control
	Battery        *battery.ChargeLimit
	Volume         *volume.Control
	Thermal        *thermal.Control
	ConfigRegistry persist.ConfigRegistry
}

func GetDependencies(conf RunConfig) (*Dependencies, error) {

	wmi, err := atkacpi.NewWMI(conf.DryRun)
	if err != nil {
		return nil, err
	}

	var config persist.ConfigRegistry

	if conf.DryRun {
		config, _ = persist.NewDryRegistryHelper()
	} else {
		config, _ = persist.NewRegistryConfigHelper()
	}

	// TODO: make powercfg dryrun-able as well
	powercfg, err := power.NewCfg()
	if err != nil {
		return nil, err
	}

	// TODO: allow user to specify profiles
	thermalCfg := thermal.Config{
		WMI:         wmi,
		PowerCfg:    powercfg,
		Profiles:    thermal.GetDefaultThermalProfiles(),
		AutoThermal: conf.EnabledFeatures.AutoThermalProfile,
		AutoThermalConfig: struct {
			PluggedIn string
			Unplugged string
		}{
			PluggedIn: "Performance",
			Unplugged: "Silent",
		},
	}

	thermal, err := thermal.NewControl(thermalCfg)
	if err != nil {
		return nil, err
	}

	// TODO: allow user to change the charge limit
	battery, err := battery.NewChargeLimit(wmi)
	if err != nil {
		return nil, err
	}

	remap := make(map[uint32]uint16)
	if conf.EnabledFeatures.FnRemap {
		remap[kb.KeyFnLeft] = kb.KeyPgUp
		remap[kb.KeyFnRight] = kb.KeyPgDown
	}
	kbCtrl, err := keyboard.NewControl(keyboard.Config{
		DryRun: conf.DryRun,
		Remap:  remap,
	})
	if err != nil {
		return nil, err
	}

	volCtrl, err := volume.NewVolumeControl(conf.DryRun)
	if err != nil {
		return nil, err
	}

	config.Register(battery)
	config.Register(thermal)
	config.Register(kbCtrl)

	return &Dependencies{
		WMI:            wmi,
		Keyboard:       kbCtrl,
		Battery:        battery,
		Volume:         volCtrl,
		Thermal:        thermal,
		ConfigRegistry: config,
	}, nil
}

// New returns a Controller to be ran
func New(conf RunConfig, dep *Dependencies) (*Controller, error) {

	if dep == nil {
		return nil, fmt.Errorf("nil Dependencies is invalid")
	}

	if len(conf.RogRemap) == 0 {
		conf.RogRemap = []string{defaultCommandWithArgs}
	}

	control, err := newController(Config{
		WMI: dep.WMI,

		Plugins: []plugin.Plugin{
			dep.Keyboard,
			dep.Volume,
			dep.Thermal,
		},
		Registry: dep.ConfigRegistry,

		LogoPath:        conf.LogoPath,
		EnabledFeatures: conf.EnabledFeatures,
		ROGKey:          conf.RogRemap,
	})

	if err != nil {
		return nil, err
	}

	return control, nil
}
