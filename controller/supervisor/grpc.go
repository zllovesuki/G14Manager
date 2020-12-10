package supervisor

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/thejerf/suture/v4"
	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/rpc/server"

	"google.golang.org/grpc"
)

type servers struct {
	Keyboard *server.KeyboardServer
	Battery  *server.BatteryServer
	Manager  *server.ManagerServer
	Configs  *server.ConfigListServer
}

type Server struct {
	errorCh      chan error
	server       *grpc.Server
	servers      servers
	startErrorCh chan error
}

type GRPCRunConfig struct {
	ManagerReqCh chan server.ManagerSupervisorRequest
	Dependencies *controller.Dependencies
}

func NewGRPCServer(conf GRPCRunConfig) (*Server, chan error, error) {
	if conf.ManagerReqCh == nil {
		return nil, nil, fmt.Errorf("nil manager request channel is invalid")
	}
	if conf.Dependencies == nil {
		return nil, nil, fmt.Errorf("nil dependencies is invalid")
	}

	s := grpc.NewServer()

	startErrorCh := make(chan error)
	server := &Server{
		errorCh: make(chan error),
		server:  s,
		servers: servers{
			Keyboard: server.RegisterKeyboardServer(s, conf.Dependencies.Keyboard),
			Battery:  server.RegisterBatteryChargeLimitServer(s, conf.Dependencies.Battery),
			Configs:  server.RegisterConfigListServer(s, conf.Dependencies.Updatable),
			Manager:  server.RegisterManagerServer(s, conf.ManagerReqCh),
		},
		startErrorCh: startErrorCh,
	}

	conf.Dependencies.ConfigRegistry.Register(server.servers.Configs)
	conf.Dependencies.ConfigRegistry.Register(server.servers.Manager)

	return server, startErrorCh, nil
}

func (s *Server) loop(haltCtx context.Context) {
	for {
		select {
		case err := <-s.errorCh:
			log.Printf("[grpc] grpc error: %+v\n", err)
			return
		case <-haltCtx.Done():
			log.Printf("[grpc] stopping grpc server\n")
			s.server.GracefulStop()
			return
		}
	}
}

func (s *Server) Serve(haltCtx context.Context) error {
	lis, err := net.Listen("tcp", "127.0.0.1:9963")
	if err != nil {
		log.Printf("[grpc] Failed to listen for connections: %+v\n", err)
		s.startErrorCh <- err
		return suture.ErrDoNotRestart
	}

	go s.loop(haltCtx)
	log.Printf("[grpc] grpc server available at 127.0.0.1:9963\n")

	return s.server.Serve(lis)
}

func (s *Server) String() string {
	return "gRPCServer"
}

func (s *Server) hotReload(dep *controller.Dependencies) {
	s.servers.Battery.HotReload(dep.Battery)
	s.servers.Keyboard.HotReload(dep.Keyboard)
	s.servers.Configs.HotReload(dep.Updatable)
	dep.ConfigRegistry.Register(s.servers.Configs)
}
