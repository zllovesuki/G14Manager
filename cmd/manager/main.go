package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/rpc/server"
	"github.com/zllovesuki/G14Manager/supervisor"
	"github.com/zllovesuki/G14Manager/supervisor/background"
	"github.com/zllovesuki/G14Manager/util"

	suture "github.com/thejerf/suture/v4"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Compile time injected variables
var (
	Version     = "v0.0.0-dev"
	IsDebug     = "yes"
	logLocation = `C:\Logs\G14Manager.log`
)

func main() {

	if IsDebug == "no" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   logLocation,
			MaxSize:    5,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		})
	}

	log.Printf("G14Manager version: %s\n", Version)

	notifier := background.NewNotifier()

	versionChecker, err := background.NewVersionCheck(Version, "zllovesuki/G14Manager", notifier.C)
	if err != nil {
		log.Fatalf("[supervisor] cannot get version checker")
	}

	controllerConfig := controller.RunConfig{
		DryRun:     os.Getenv("DRY_RUN") != "",
		NotifierCh: notifier.C,
	}

	dep, err := controller.GetDependencies(controllerConfig)
	if err != nil {
		log.Fatalf("[supervisor] cannot get dependencies\n")
	}

	managerCtrl := make(chan server.ManagerSupervisorRequest, 1)

	grpcServer, err := supervisor.NewGRPCServer(supervisor.GRPCRunConfig{
		ManagerReqCh: managerCtrl,
		Dependencies: dep,
	})
	if err != nil {
		log.Fatalf("[supervisor] cannot create gRPCServer: %+v\n", err)
	}

	managerResponder := &supervisor.ManagerResponder{
		Dependencies:     dep,
		ManagerReqCh:     managerCtrl,
		ControllerConfig: controllerConfig,
	}

	evtHook := &supervisor.EventHook{
		Notifier: notifier.C,
	}

	ctx, cancel := context.WithCancel(context.Background())

	/*
		How the supervisor tree is structured:
			gRPCSupervisor:		supervisor/grpc.go
			gRPCServer: 		rpc/server
			ManagerResponder:	supervisor/responder.go
			versionChecker:		supervisor/background/version.go
			osdNotifier:		supervisor/background/notifier.go
			controller:			controller

								rootSupervisor  +----+  externalWeb
									+    +
									|    |
									|    |
				gRPCSupervisor  +---+    +---+   backgroundSupervisor
				+ + +                            + +
				| | |                            | |
				| | +-> gRPCServer               | +-> versionChecker
				| |                              |
				| |                              |
				| +---> ManagerResponder         +---> osdNotifier
				|
				|
				+-----> controllerSupervisor
							+
							|
							+-> Controller

		Since the gRPCServer can control the lifecycle of the Controller,
		we need a two-way communication between the gRPCSupervisor and
		the gRPC ManagerServer via ManagerReqCh. The coordination is handled
		by ManagerResponder

	*/

	backgroundSupervisor := suture.New("backgroundSupervisor", suture.Spec{})
	backgroundSupervisor.Add(versionChecker)
	backgroundSupervisor.Add(notifier)

	grpcSupervisor := suture.New("gRPCSupervisor", suture.Spec{})
	managerResponder.SetSupervisor(grpcSupervisor)
	grpcSupervisor.Add(grpcServer)
	grpcSupervisor.Add(managerResponder)

	rootSupervisor := suture.New("Supervisor", suture.Spec{
		EventHook: evtHook.Event,
	})
	rootSupervisor.Add(grpcSupervisor)
	rootSupervisor.Add(backgroundSupervisor)
	rootSupervisor.Add(NewWeb(grpcServer.GetWebHandler()))

	sigc := make(chan os.Signal, 1)

	go func() {
		notifier.C <- util.Notification{
			Message:   "Starting up G14Manager Supervisor",
			Immediate: true,
			Delay:     time.Second * 2,
		}
		supervisorErr := rootSupervisor.Serve(ctx)
		if supervisorErr != nil {
			log.Printf("[supervisor] rootSupervisor returns error: %+v\n", supervisorErr)
			sigc <- syscall.SIGTERM
		}
	}()

	signal.Notify(
		sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	sig := <-sigc
	log.Printf("[supervisor] signal received: %+v\n", sig)

	cancel()
	dep.ConfigRegistry.Close()
	time.Sleep(time.Second) // 1 second for grace period
}
