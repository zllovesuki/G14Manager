package main

import (
	"context"
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
	"github.com/zllovesuki/G14Manager/controller/supervisor"
	"github.com/zllovesuki/G14Manager/rpc/server"

	suture "github.com/thejerf/suture/v4"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Compile time injected variables
var (
	Version = "dev"
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

	managerCtrl := make(chan server.ManagerSupervisorRequest, 1)

	grpcServer, grpcStartErrCh, err := supervisor.NewGRPCServer(supervisor.GRPCRunConfig{
		ManagerReqCh: managerCtrl,
		Dependencies: dep,
	})
	if err != nil {
		log.Fatalf("[supervisor] cannot start gRPCServer: %+v\n", err)
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
		                               /---    ---\
		                        /------           -------\
		                    ----                          ----
		            gRPCSupervisor                     backgroundSupervisor
		             /--  |    --\                             |
		         /---     |       ---\                         |
		      ---         |           --                       |
		 gRPCServer       |     ManagerResponder         versionChecker
		                  |
		                  |
		         controllerSupervisor
		                  |
		                  |
		                  |
		             Controller (runs plugins, etc)

		Since the gRPCServer can control the lifecycle of the Controller,
		we need a two-way communication between the gRPCSupervisor and
		the gRPC Manager Server (RequestCh).

	*/

	rootSupervisor := suture.New("Supervisor", suture.Spec{})
	rootSupervisor.Add(grpcSupervisor)

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
