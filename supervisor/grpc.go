package supervisor

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/pkg/errors"
	"github.com/thejerf/suture/v4"
	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/rpc/server"
	"github.com/zllovesuki/G14Manager/system/shared"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

type servers struct {
	Keyboard *server.KeyboardServer
	Battery  *server.BatteryServer
	Thermal  *server.ThermalServer
	Manager  *server.ManagerServer
	Configs  *server.ConfigListServer
}

type Server struct {
	server  *grpc.Server
	servers servers
	dep     *controller.Dependencies
	grpcWeb *grpcweb.WrappedGrpcServer
}

func (s *Server) GetWebHandler() *grpcweb.WrappedGrpcServer {
	return s.grpcWeb
}

type GRPCRunConfig struct {
	ManagerReqCh chan server.ManagerSupervisorRequest
	Dependencies *controller.Dependencies
}

func NewGRPCServer(conf GRPCRunConfig) (*Server, error) {
	if conf.ManagerReqCh == nil {
		return nil, fmt.Errorf("nil manager request channel is invalid")
	}
	if conf.Dependencies == nil {
		return nil, fmt.Errorf("nil dependencies is invalid")
	}

	s := grpc.NewServer()

	server := &Server{
		server: s,
		servers: servers{
			Keyboard: server.RegisterKeyboardServer(s, conf.Dependencies.Keyboard),
			Battery:  server.RegisterBatteryChargeLimitServer(s, conf.Dependencies.Battery),
			Thermal:  server.RegisterThermalServer(s, conf.Dependencies.Thermal),
			Configs:  server.RegisterConfigListServer(s, conf.Dependencies.Updatable),
			Manager:  server.RegisterManagerServer(s, conf.ManagerReqCh),
		},
		dep: conf.Dependencies,
	}

	conf.Dependencies.ConfigRegistry.Register(server.servers.Configs)
	conf.Dependencies.ConfigRegistry.Register(server.servers.Manager)

	server.grpcWeb = grpcweb.WrapServer(s)

	return server, nil
}

func (s *Server) Serve(haltCtx context.Context) error {
	lis, err := net.Listen("tcp", shared.GRPCAddress)
	if err != nil {
		log.Printf("[gRPCServer] Failed to listen for connections: %+v\n", err)
		return errors.Wrap(suture.ErrTerminateSupervisorTree, "[gRPCServer] failed to listen for connectios") // If we cannot start gRPC Server, kill the entire tree
	}

	go func() {
		<-haltCtx.Done()
		log.Printf("[gRPCServer] stopping grpc server\n")
		s.server.GracefulStop()
		log.Printf("[gRPCServer] server stopped\n")
	}()
	log.Printf("[gRPCServer] grpc server available at %s\n", shared.GRPCAddress)

	s.dep.ConfigRegistry.Load()

	return s.server.Serve(lis)
}

func (s *Server) String() string {
	return "gRPCServer"
}

func (s *Server) hotReload(dep *controller.Dependencies) {
	s.servers.Battery.HotReload(dep.Battery)
	s.servers.Keyboard.HotReload(dep.Keyboard)
	s.servers.Thermal.HotReload(dep.Thermal)
	s.servers.Configs.HotReload(dep.Updatable)
	dep.ConfigRegistry.Register(s.servers.Configs)
	dep.ConfigRegistry.Register(s.servers.Manager)
}
