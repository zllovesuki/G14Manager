package controller

import (
	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/battery"
	"github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/system/volume"
	"github.com/zllovesuki/G14Manager/util"
)

const defaultCommandWithArgs = "Taskmgr.exe"

// RunConfig contains the start up configuration for the controller
type RunConfig struct {
	RogRemap        util.ArrayFlags
	DryRun          bool
	EnabledFeatures Features
}

// New returns a Controller to be ran
func New(conf RunConfig) (*Controller, error) {

	if len(conf.RogRemap) == 0 {
		conf.RogRemap = []string{defaultCommandWithArgs}
	}

	wmi, err := atkacpi.NewWMI(conf.DryRun)
	if err != nil {
		return nil, err
	}

	var config persist.ConfigRegistry

	if conf.DryRun {
		config, _ = persist.NewDryRegistryHelper()
	} else {
		config, _ = persist.NewRegistryHelper()
	}

	// TODO: make powercfg dryrun-able as well
	powercfg, err := power.NewCfg()
	if err != nil {
		return nil, err
	}

	// TODO: allow user to specify profiles
	thermalCfg := thermal.Config{
		WMI:      wmi,
		PowerCfg: powercfg,
		Profiles: thermal.GetDefaultThermalProfiles(),
	}

	profile, err := thermal.NewControl(thermalCfg)
	if err != nil {
		return nil, err
	}

	// TODO: allow user to change the charge limit
	battery, err := battery.NewChargeLimit(wmi)
	if err != nil {
		return nil, err
	}

	kbCtrl, err := keyboard.NewControl(conf.DryRun)
	if err != nil {
		return nil, err
	}

	volCtrl, err := volume.NewVolumeControl(conf.DryRun)
	if err != nil {
		return nil, err
	}

	// order powercfg to last
	config.Register(battery)
	config.Register(profile)
	config.Register(powercfg)
	config.Register(kbCtrl)

	control, err := newController(Config{
		WMI: wmi,

		VolumeControl:   volCtrl,
		KeyboardControl: kbCtrl,
		Thermal:         profile,
		Registry:        config,

		EnabledFeatures: conf.EnabledFeatures,
		ROGKey:          conf.RogRemap,
	})

	if err != nil {
		return nil, err
	}

	return control, nil
}