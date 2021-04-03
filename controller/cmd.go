package controller

import (
	"fmt"

	"github.com/zllovesuki/G14Manager/cxx/plugin/gpu"
	"github.com/zllovesuki/G14Manager/cxx/plugin/keyboard"
	"github.com/zllovesuki/G14Manager/cxx/plugin/rr"
	"github.com/zllovesuki/G14Manager/cxx/plugin/volume"
	"github.com/zllovesuki/G14Manager/rpc/announcement"
	"github.com/zllovesuki/G14Manager/system/atkacpi"
	"github.com/zllovesuki/G14Manager/system/battery"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/plugin"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/util"

	"github.com/pkg/errors"
)

// RunConfig contains the start up configuration for the controller
type RunConfig struct {
	DryRun     bool
	NotifierCh chan util.Notification
}

type Dependencies struct {
	WMI            atkacpi.WMI
	Keyboard       *keyboard.Control
	Battery        *battery.ChargeLimit
	Volume         *volume.Control
	Thermal        *thermal.Control
	GPU            *gpu.Control
	RR             *rr.Control
	ConfigRegistry persist.ConfigRegistry
	Updatable      []announcement.Updatable
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

	thermalCfg := thermal.Config{
		WMI:      wmi,
		PowerCfg: powercfg,
		Profiles: thermal.GetDefaultThermalProfiles(),
	}

	thermal, err := thermal.NewControl(thermalCfg)
	if err != nil {
		return nil, err
	}

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

	gpuCtrl, err := gpu.NewGPUControl(conf.DryRun)
	if err != nil {
		return nil, err
	}

	rrCtrl, err := rr.NewRRControl(conf.DryRun)
	if err != nil {
		return nil, err
	}

	config.Register(battery)
	config.Register(kbCtrl)

	updatable := []announcement.Updatable{
		thermal,
		kbCtrl,
	}

	return &Dependencies{
		WMI:            wmi,
		Keyboard:       kbCtrl,
		Battery:        battery,
		Volume:         volCtrl,
		Thermal:        thermal,
		GPU:            gpuCtrl,
		RR:             rrCtrl,
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
		return nil, nil, errors.New("nil WMI is invalid")
	}
	if dep.ConfigRegistry == nil {
		return nil, nil, errors.New("nil Registry is invalid")
	}
	if conf.NotifierCh == nil {
		return nil, nil, errors.New("nil NotifierCh is invalid")
	}

	startErrorCh := make(chan error, 1)
	control := &Controller{
		Config: Config{
			WMI: dep.WMI,

			Plugins: []plugin.Plugin{
				dep.Keyboard,
				dep.Volume,
				dep.Thermal,
				dep.GPU,
				dep.RR,
			},
			Registry: dep.ConfigRegistry,

			Notifier: conf.NotifierCh,
		},

		workQueueCh:  make(map[uint32]workQueue, 1),
		errorCh:      make(chan error),
		startErrorCh: startErrorCh,

		keyCodeCh:  make(chan uint32, 1),
		acpiCh:     make(chan uint32, 1),
		powerEvCh:  make(chan uint32, 1),
		pluginCbCh: make(chan plugin.Callback, 1),
	}

	return control, startErrorCh, nil
}
