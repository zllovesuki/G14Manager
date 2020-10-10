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
	var enableExperimental = flag.Bool("experimental", false, "enable experimental features (such as Fn+Left remapping)")

	flag.Parse()

	log.Printf("G14Manager version: %s\n", Version)
	log.Printf("Experimental enabled: %v\n", *enableExperimental)

	control, err := controller.New(controller.RunConfig{
		RogRemap:           rogRemap,
		EnableExperimental: *enableExperimental,
		DryRun:             os.Getenv("DRY_RUN") != "",
	})
	if err != nil {
		util.SendToastNotification("G14Manager Supervisor", util.Notification{
			Title:   "G14Manager cannot be started",
			Message: fmt.Sprintf("Error: %v", err),
		})
		log.Fatalf("[supervisor] controller configuration error: %v\n", err)
	}

	supervisor := oversight.New(
		oversight.WithRestartStrategy(oversight.OneForOne()),
		oversight.Process(oversight.ChildProcessSpecification{
			Name:  "Controller",
			Start: control.Run,
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
			log.Fatalln(err)
		}
	}()

	<-sigc
	cancel()
	time.Sleep(time.Second * 5) // 5 second for grace period
	os.Exit(0)
}
