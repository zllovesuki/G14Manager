package controller

import (
	"context"
	"time"

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
	RogRemap           util.ArrayFlags
	EnableExperimental bool
	DryRun             bool
}

// Run start the controller daemon
func Run(ctx context.Context, conf RunConfig) error {

	if len(conf.RogRemap) == 0 {
		conf.RogRemap = []string{defaultCommandWithArgs}
	}

	var wmi atkacpi.WMI
	var config persist.ConfigRegistry
	var err error

	if conf.DryRun {
		wmi, _ = atkacpi.NewDryWMI()
		config, err = persist.NewDryRegistryHelper()
		if err != nil {
			return err
		}
	} else {
		wmi, err = atkacpi.NewWMI()
		if err != nil {
			return err
		}
		config, _ = persist.NewRegistryHelper()
	}

	powercfg, err := power.NewCfg()
	if err != nil {
		return err
	}

	// TODO: allow user to specify profiles
	thermalCfg := thermal.Config{
		WMI:      wmi,
		PowerCfg: powercfg,
		Profiles: thermal.GetDefaultThermalProfiles(),
	}

	profile, err := thermal.NewControl(thermalCfg)
	if err != nil {
		return err
	}

	// TODO: allow user to change the charge limit
	battery, err := battery.NewChargeLimit(wmi)
	if err != nil {
		return err
	}

	kbCtrl, err := keyboard.NewControl()
	if err != nil {
		return err
	}

	volCtrl, err := volume.NewControl()
	if err != nil {
		return err
	}

	// order powercfg to last
	config.Register(battery)
	config.Register(profile)
	config.Register(powercfg)
	config.Register(kbCtrl)

	control, err := NewController(Config{
		WMI:                wmi,
		EnableExperimental: conf.EnableExperimental,
		VolumeControl:      volCtrl,
		KeyboardControl:    kbCtrl,
		Thermal:            profile,
		Registry:           config,
		ROGKey:             conf.RogRemap,
	})

	if err != nil {
		return err
	}

	control.Run(ctx)

	<-time.After(time.Second)
	return nil
}
