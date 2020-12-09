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

type Server struct {
	reload  <-chan GRPCConfig
	errorCh chan error
	server  *grpc.Server
	servers struct {
		Keyboard *server.KeyboardServer
		Battery  *server.BatteryServer
	}
}

func NewGRPCServer(reload <-chan GRPCConfig) (*Server, error) {
	if reload == nil {
		return nil, fmt.Errorf("nil reload channel is invalid")
	}
	s := grpc.NewServer()
	return &Server{
		reload:  reload,
		errorCh: make(chan error),
		server:  s,
		servers: struct {
			Keyboard *server.KeyboardServer
			Battery  *server.BatteryServer
		}{
			Keyboard: server.RegisterKeyboardServer(s, nil),
			Battery:  server.RegisterBatteryChargeLimitServer(s, nil),
		},
	}, nil
}

func (s *Server) loop(haltCtx context.Context) {
	for {
		select {
		case conf := <-s.reload:
			log.Printf("[grpc] hot reloading control interfaces\n")
			s.hotReload(conf)
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

func (s *Server) hotReload(conf GRPCConfig) {
	s.servers.Battery.HotReload(conf.BatteryControl)
	s.servers.Keyboard.HotReload(conf.KeyboardControl)
}
