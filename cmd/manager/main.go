package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zllovesuki/G14Manager/background"
	"github.com/zllovesuki/G14Manager/box"
	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/controller/supervisor"
	"github.com/zllovesuki/G14Manager/rpc/server"

	suture "github.com/thejerf/suture/v4"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Compile time injected variables
var (
	Version = "v0.0.0-dev"
	IsDebug = "yes"
)

func main() {

	if IsDebug == "no" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   `C:\Logs\G14Manager.log`,
			MaxSize:    5,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		})
	}

	log.Printf("G14Manager version: %s\n", Version)

	asset := box.GetAssetExtractor()
	defer asset.Close()

	notifier := background.NewNotifier()

	versionChecker, err := background.NewVersionCheck(Version, "zllovesuki/G14Manager", notifier.C)
	if err != nil {
		log.Fatalf("[supervisor] cannot get version checker")
	}

	backgroundSupervisor := suture.New("backgroundSupervisor", suture.Spec{})
	backgroundSupervisor.Add(versionChecker)
	backgroundSupervisor.Add(notifier)

	controllerConfig := controller.RunConfig{
		LogoPath:   asset.Get("/Logo.png"),
		DryRun:     os.Getenv("DRY_RUN") != "",
		NotifierCh: notifier.C,
	}

	dep, err := controller.GetDependencies(controllerConfig)
	if err != nil {
		log.Fatalf("[supervisor] cannot get dependencies\n")
	}

	managerCtrl := make(chan server.ManagerSupervisorRequest, 1)

	grpcServer, grpcStartErrCh, err := supervisor.NewGRPCServer(supervisor.GRPCRunConfig{
		ManagerReqCh: managerCtrl,
		Dependencies: dep,
	})
	if err != nil {
		log.Fatalf("[supervisor] cannot create gRPCServer: %+v\n", err)
	}

	grpcSupervisor := suture.New("gRPCSupervisor", suture.Spec{})
	managerResponder := &supervisor.ManagerResponderOption{
		Supervisor:       grpcSupervisor,
		Dependencies:     dep,
		ManagerReqCh:     managerCtrl,
		ControllerConfig: controllerConfig,
	}
	grpcSupervisor.Add(grpcServer)
	grpcSupervisor.Add(managerResponder)

	ctx, cancel := context.WithCancel(context.Background())

	/*
		How the supervisor tree is structured:
		(gRPCSupervisor: controller/supervisor)

								rootSupervisor
									+    +
									|    |
									|    |
				gRPCSupervisor  +---+    +---+   backgroundSupervisor
				+ + +                            + +
				| | |                            | |
				| | +-> gRPCServer               | +-> versionChecker
				| |                              |
				| |                              |
				| +---> ManagerResponder         +---> toastNotifier
				|
				|
				+-----> controllerSupervisor
							+
							|
							+-> Controller

		Since the gRPCServer can control the lifecycle of the Controller,
		we need a two-way communication between the gRPCSupervisor and
		the gRPC Manager Server (ManagerReqCh).

	*/

	evtHook := &supervisor.EventHook{
		Notifier: notifier.C,
	}

	rootSupervisor := suture.New("Supervisor", suture.Spec{
		EventHook: evtHook.Event,
	})
	rootSupervisor.Add(grpcSupervisor)
	rootSupervisor.Add(backgroundSupervisor)

	rootSupervisor.ServeBackground(ctx)

	select {
	case grpcStartErr := <-grpcStartErrCh:
		log.Fatalf("[supervisor] Cannot start gRPC Server: %+v\n", grpcStartErr)
	case <-time.After(time.Second * 2):
		dep.ConfigRegistry.Load()
	}

	srv := &http.Server{Addr: "127.0.0.1:9969"}
	go func() {
		log.Printf("[supervisor] pprof at 127.0.0.1:9969/debug/pprof\n")
		log.Printf("[supervisor] pprof exit: %+v\n", srv.ListenAndServe())
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(
		sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	<-sigc

	cancel()
	srv.Shutdown(context.Background())
	dep.ConfigRegistry.Close()
	time.Sleep(time.Second) // 1 second for grace period
}
