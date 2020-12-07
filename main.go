package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/util"

	"cirello.io/oversight"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Version = "dev"
)

var defaultCommandWithArgs = "Taskmgr.exe"

func main() {

	if Version != "dev" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   `C:\Logs\G14Manager.log`,
			MaxSize:    5,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		})
	}

	var rogRemap util.ArrayFlags
	flag.Var(&rogRemap, "rog", "customize ROG key behavior when pressed multiple times")

	var enableRemap = flag.Bool("remap", false, "enable remapping Fn+Left/Right to PgUp/PgDown")
	var enableAutoThermal = flag.Bool("autoThermal", false, "enable automatic thermal profile switching on power source change")

	flag.Parse()

	log.Printf("G14Manager version: %s\n", Version)
	log.Printf("Remapping enabled: %v\n", *enableRemap)
	log.Printf("Automatic Thermal Profile Switching enabled: %v\n", *enableAutoThermal)

	controllerConfig := controller.RunConfig{
		RogRemap: rogRemap,
		EnabledFeatures: controller.Features{
			FnRemap:            *enableRemap,
			AutoThermalProfile: *enableAutoThermal,
		},
		DryRun: os.Getenv("DRY_RUN") != "",
	}

	supervisor := oversight.New(
		oversight.WithRestartStrategy(oversight.OneForOne()),
		oversight.Process(oversight.ChildProcessSpecification{
			Name: "Controller",
			Start: func(ctx context.Context) error {
				control, err := controller.New(controllerConfig)
				if err != nil {
					return err
				}
				return control.Run(ctx)
			},
			Restart: func(err error) bool {
				if err == nil {
					return false
				}
				log.Println("[supervisor] controller returned an error:")
				log.Printf("%+v\n", err)
				util.SendToastNotification("G14Manager Supervisor", util.Notification{
					Title:   "G14Manager will be restarted",
					Message: fmt.Sprintf("An error has occurred: %s", err),
				})
				return true
			},
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())

	sigc := make(chan os.Signal, 1)
	signal.Notify(
		sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		log.Println("[supervisor] Monitoring controller")
		if err := supervisor.Start(ctx); err != nil {
			util.SendToastNotification("G14Manager Supervisor", util.Notification{
				Title:   "G14Manager cannot be started",
				Message: fmt.Sprintf("Error: %v", err),
			})
			log.Fatalf("[supervisor] controller start error: %v\n", err)
		}
	}()

	<-sigc
	cancel()
	time.Sleep(time.Second) // 1 second for grace period
}
