package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zllovesuki/G14Manager/box"
	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/rpc/server"
	"github.com/zllovesuki/G14Manager/util"

	"github.com/thejerf/suture"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Compile time injected variables
var (
	Version = "dev"
	IsDebug = "yes"
)

var defaultCommandWithArgs = "Taskmgr.exe"

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

	var rogRemap util.ArrayFlags
	flag.Var(&rogRemap, "rog", "customize ROG key behavior when pressed multiple times")

	var enableRemap = flag.Bool("remap", false, "enable remapping Fn+Left/Right to PgUp/PgDown")
	var enableAutoThermal = flag.Bool("autoThermal", false, "enable automatic thermal profile switching on power source change")

	flag.Parse()

	if len(rogRemap) == 0 {
		rogRemap = append(rogRemap, "Taskmgr.exe")
	}

	log.Printf("G14Manager version: %s\n", Version)
	log.Printf("Remapping enabled: %v\n", *enableRemap)
	log.Printf("Automatic Thermal Profile Switching enabled: %v\n", *enableAutoThermal)

	var logoPath string
	logoPng := box.Get("/Logo.png")
	if logoPng != nil {
		logoFile, err := ioutil.TempFile(os.TempDir(), "G14Manager-")
		if err != nil {
			log.Fatal("[supervisor] Cannot create temporary file for logo", err)
		}
		defer func() {
			time.Sleep(time.Second)
			os.Remove(logoFile.Name())
		}()

		if _, err = logoFile.Write(logoPng); err != nil {
			log.Fatal("[supervisor] Failed to write to temporary file for logo", err)
		}

		if err := logoFile.Close(); err != nil {
			log.Fatal(err)
		}

		logoPath = logoFile.Name()
		log.Printf("[supervisor] Logo extracted to %s\n", logoPath)
	}

	controllerConfig := controller.RunConfig{
		LogoPath: logoPath,
		DryRun:   os.Getenv("DRY_RUN") != "",
	}

	dep, err := controller.GetDependencies(controllerConfig)
	if err != nil {
		log.Fatalf("[supervisor] cannot get dependencies\n")
	}

	reload := make(chan *controller.Dependencies, 1)
	managerCtrl := make(chan server.ManagerSupervisorRequest, 1)

	grpcServer, grpcStartErrCh, err := controller.NewGRPCServer(controller.GRPCRunConfig{
		ReloadCh:     reload,
		ManagerReqCh: managerCtrl,
	})
	if err != nil {
		log.Fatalf("[supervisor] cannot start gRPCServer: %+v\n", err)
	}

	supervisor := suture.New("gRPCServer", suture.Spec{
		Log: func(msg string) {
			log.Printf("[supervisor] %s\n", msg)
		},
	})
	supervisor.Add(grpcServer)

	ctx, cancel := context.WithCancel(context.Background())

	/*
		How the supervisor tree is structured:

		root -> gRPCServer
					\-> Controller

		Since the gRPCServer can control the lifecycle of the Controller,
		we need a two-way communication between the Supervisor tree and
		the gRPC Manager Server (RequestCh).

		Since the Controller initializes the hardware control functions,
		gRPCServer managing the hardware functions neeed fresh instances
		of those Controls, we will forward Dependencies to gRPCServer
		for hot reloading (ReloadCh).
	*/

	supervisor.ServeBackground()

	select {
	case grpcStartErr := <-grpcStartErrCh:
		log.Fatalf("[supervisor] Cannot start gRPC Server: %+v\n", grpcStartErr)
	case <-time.After(time.Second * 2):
		// This requires some explaination

		reload <- dep // the first reload is to populate gRPC servers dependencies
		time.Sleep(time.Millisecond * 500)
		dep.ConfigRegistry.Load() // this is to load configurations from config registry
		time.Sleep(time.Millisecond * 500)
		reload <- dep // this will annouce configurations to annoucement.Updatable's
	}

	go controller.ManagerResponder(ctx, controller.ManagerResponderOption{
		Supervisor:       supervisor,
		ReloadCh:         reload,
		Dependencies:     dep,
		ManagerReqCh:     managerCtrl,
		ControllerConfig: controllerConfig,
	})

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
	supervisor.Stop()
	srv.Shutdown(context.Background())
	dep.ConfigRegistry.Close()
	time.Sleep(time.Second) // 1 second for grace period
}
