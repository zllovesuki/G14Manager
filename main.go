package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/system/battery"
	"github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/power"
	"github.com/zllovesuki/G14Manager/system/thermal"
	"github.com/zllovesuki/G14Manager/system/volume"
	"github.com/zllovesuki/G14Manager/util"
)

var (
	Version = "dev"
)

var defaultCommandWithArgs = "Taskmgr.exe"

func main() {

	var enableExperimental = flag.Bool("experimental", false, "enable experimental features (such as Fn+Left remapping)")

	var rogRemap util.ArrayFlags
	flag.Var(&rogRemap, "rog", "customize ROG key behavior when pressed multiple times")

	flag.Parse()

	log.Printf("G14Manager version: %s\n", Version)
	log.Printf("Experimental enabled: %v\n", *enableExperimental)
	if os.Getenv("DRY_RUN") != "" {
		log.Printf("[dry run] no hardware i/o will be performed")
	}

	if len(rogRemap) == 0 {
		rogRemap = []string{defaultCommandWithArgs}
	}

	config, _ := persist.NewRegistryHelper()

	powercfg, err := power.NewCfg()
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: allow user to specify profiles
	thermalCfg := thermal.Config{
		PowerCfg: powercfg,
		Profiles: thermal.GetDefaultThermalProfiles(),
	}

	profile, err := thermal.NewControl(thermalCfg)
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: allow user to change the charge limit
	battery, err := battery.NewChargeLimit()
	if err != nil {
		log.Fatalln(err)
	}

	kbCtrl, err := keyboard.NewControl()
	if err != nil {
		log.Fatalln(err)
	}

	volCtrl, err := volume.NewControl()
	if err != nil {
		log.Fatalln(err)
	}

	// order powercfg to last
	config.Register(battery)
	config.Register(profile)
	config.Register(powercfg)
	config.Register(kbCtrl)

	control, err := controller.NewController(controller.Config{
		EnableExperimental: *enableExperimental,
		VolumeControl:      volCtrl,
		KeyboardControl:    kbCtrl,
		Thermal:            profile,
		Registry:           config,
		ROGKey:             rogRemap,
	})

	if err != nil {
		log.Fatalln(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		cancel()
		time.Sleep(time.Second * 5) // 5 second for grace period
		os.Exit(0)
	}()

	control.Run(ctx)
}
