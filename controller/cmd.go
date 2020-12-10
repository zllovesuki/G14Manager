package controller

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/zllovesuki/G14Manager/cxx/plugin/keyboard"
	"github.com/zllovesuki/G14Manager/cxx/plugin/volume"
	"github.com/zllovesuki/G14Manager/rpc/annoucement"
	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/battery"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/plugin"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/util"
)

const defaultCommandWithArgs = "Taskmgr.exe"

// RunConfig contains the start up configuration for the controller
type RunConfig struct {
	DryRun   bool
	LogoPath string
}

type Dependencies struct {
	WMI            atkacpi.WMI
	Keyboard       *keyboard.Control
	Battery        *battery.ChargeLimit
	Volume         *volume.Control
	Thermal        *thermal.Control
	ConfigRegistry persist.ConfigRegistry
	Updatable      []annoucement.Updatable
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
		WMI:      wmi,
		PowerCfg: powercfg,
		Profiles: thermal.GetDefaultThermalProfiles(),
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

	kbCtrl, err := keyboard.NewControl(keyboard.Config{
		DryRun: conf.DryRun,
		RogKey: []string{"Taskmgr.exe"},
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

	updatable := []annoucement.Updatable{
		thermal,
		kbCtrl,
	}

	return &Dependencies{
		WMI:            wmi,
		Keyboard:       kbCtrl,
		Battery:        battery,
		Volume:         volCtrl,
		Thermal:        thermal,
		ConfigRegistry: config,
		Updatable:      updatable,
	}, nil
}

// New returns a Controller to be ran
func New(conf RunConfig, dep *Dependencies) (*Controller, chan error, error) {

	if dep == nil {
		return nil, nil, fmt.Errorf("nil Dependencies is invalid")
	}
	if dep.WMI == nil {
		return nil, nil, errors.New("[controller] nil WMI is invalid")
	}
	if dep.ConfigRegistry == nil {
		return nil, nil, errors.New("[controller] nil Registry is invalid")
	}

	startErrorCh := make(chan error, 1)
	control := &Controller{
		Config: Config{
			WMI: dep.WMI,

			Plugins: []plugin.Plugin{
				dep.Keyboard,
				dep.Volume,
				dep.Thermal,
			},
			Registry: dep.ConfigRegistry,

			LogoPath: conf.LogoPath,
		},

		notifyQueueCh: make(chan util.Notification, 10),
		workQueueCh:   make(map[uint32]workQueue, 1),
		errorCh:       make(chan error),
		startErrorCh:  startErrorCh,

		keyCodeCh:  make(chan uint32, 1),
		acpiCh:     make(chan uint32, 1),
		powerEvCh:  make(chan uint32, 1),
		pluginCbCh: make(chan plugin.Callback, 1),
	}

	return control, startErrorCh, nil
}
