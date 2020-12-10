package controller

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/zllovesuki/G14Manager/cxx/plugin/keyboard"
	"github.com/zllovesuki/G14Manager/rpc/server"
	"github.com/zllovesuki/G14Manager/system/battery"

	"google.golang.org/grpc"
)

type GRPCConfig struct {
	KeyboardControl *keyboard.Control
	BatteryControl  *battery.ChargeLimit
}

type servers struct {
	Keyboard *server.KeyboardServer
	Battery  *server.BatteryServer
	Manager  *server.ManagerServer
}

type Server struct {
	reload  <-chan *Dependencies
	errorCh chan error
	server  *grpc.Server
	servers servers
}

type GRPCRunConfig struct {
	ReloadCh  <-chan *Dependencies
	RequestCh chan server.SupervisorRequest
}

func NewGRPCServer(conf GRPCRunConfig) (*Server, error) {
	if conf.ReloadCh == nil {
		return nil, fmt.Errorf("nil reload channel is invalid")
	}
	if conf.RequestCh == nil {
		return nil, fmt.Errorf("nil control channel is invalid")
	}
	s := grpc.NewServer()
	return &Server{
		reload:  conf.ReloadCh,
		errorCh: make(chan error),
		server:  s,
		servers: servers{
			Keyboard: server.RegisterKeyboardServer(s, nil),
			Battery:  server.RegisterBatteryChargeLimitServer(s, nil),
			Manager:  server.RegisterManagerServer(s, conf.RequestCh),
		},
	}, nil
}

func (s *Server) loop(haltCtx context.Context) {
	for {
		select {
		case dep := <-s.reload:
			log.Printf("[grpc] hot reloading control interfaces\n")
			s.hotReload(dep)
		case <-haltCtx.Done():
			log.Printf("[grpc] stopping grpc server\n")
			s.server.GracefulStop()
			return
		}
	}
}

func (s *Server) Run(haltCtx context.Context) error {
	// TODO: configurable port?
	lis, err := net.Listen("tcp", "127.0.0.1:9963")
	if err != nil {
		return err
	}

	go s.loop(haltCtx)
	go func() {
		log.Printf("[grpc] grpc server available at 127.0.0.1:9963\n")
		s.errorCh <- s.server.Serve(lis)
	}()

	return <-s.errorCh
}

func (s *Server) hotReload(dep *Dependencies) {
	s.servers.Battery.HotReload(dep.Battery)
	s.servers.Keyboard.HotReload(dep.Keyboard)
}
