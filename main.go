package main

import (
	"context"
	"flag"
	"log"
	"os"

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
	var enableExperimental = flag.Bool("experimental", false, "enable experimental features (such as Fn+Left remapping)")

	flag.Parse()

	log.Printf("G14Manager version: %s\n", Version)
	log.Printf("Experimental enabled: %v\n", *enableExperimental)

	supervisor := oversight.New(
		oversight.WithRestartStrategy(oversight.OneForOne()),
		oversight.Processes(func(ctx context.Context) error {
			return controller.Run(ctx, controller.RunConfig{
				RogRemap:           rogRemap,
				EnableExperimental: *enableExperimental,
				DryRun:             os.Getenv("DRY_RUN") != "",
			})
		}),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Println("Starting supervisor")
	if err := supervisor.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
