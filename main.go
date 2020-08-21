package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zllovesuki/ROGManager/system/keyboard"
	"github.com/zllovesuki/ROGManager/system/volume"

	"github.com/zllovesuki/ROGManager/controller"
	"github.com/zllovesuki/ROGManager/system/battery"
	"github.com/zllovesuki/ROGManager/system/persist"
	"github.com/zllovesuki/ROGManager/system/thermal"
	"github.com/zllovesuki/ROGManager/util"
)

var (
	Version = "dev"
)

var defaultCommandWithArgs = "Taskmgr.exe"

func main() {

	log.Printf("ROGManager version: %s\n", Version)

	var rogRemap util.ArrayFlags

	flag.Var(&rogRemap, "rog", "customize ROG key behavior when pressed multiple times")
	flag.Parse()

	if len(rogRemap) == 0 {
		rogRemap = []string{defaultCommandWithArgs}
	}

	config, _ := persist.NewRegistryHelper()

	powercfg, err := thermal.NewPowerCfg()
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: allow user to specify profiles
	thermalCfg := thermal.Config{
		PowerCfg: powercfg,
		Profiles: thermal.GetDefaultThermalProfiles(),
	}

	profile, err := thermal.NewThermal(thermalCfg)
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
		VolumeControl:   volCtrl,
		KeyboardControl: kbCtrl,
		Thermal:         profile,
		Registry:        config,
		ROGKey:          rogRemap,
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
