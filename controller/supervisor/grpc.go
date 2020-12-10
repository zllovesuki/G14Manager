package supervisor

import (
	"context"
	"fmt"
	"log"
	"net"

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
	reload        <-chan *controller.Dependencies
	errorCh       chan error
	server        *grpc.Server
	servers       servers
	ctx           context.Context
	cancelFn      context.CancelFunc
	startErrorCh  chan error
	unrecoverable bool
}

type GRPCRunConfig struct {
	ReloadCh     <-chan *controller.Dependencies
	ManagerReqCh chan server.ManagerSupervisorRequest
}

func NewGRPCServer(conf GRPCRunConfig) (*Server, chan error, error) {
	if conf.ReloadCh == nil {
		return nil, nil, fmt.Errorf("nil reload channel is invalid")
	}
	if conf.ManagerReqCh == nil {
		return nil, nil, fmt.Errorf("nil manager request channel is invalid")
	}

	s := grpc.NewServer()

	startErrorCh := make(chan error)
	return &Server{
		reload:  conf.ReloadCh,
		errorCh: make(chan error),
		server:  s,
		servers: servers{
			Keyboard: server.RegisterKeyboardServer(s, nil),
			Battery:  server.RegisterBatteryChargeLimitServer(s, nil),
			Configs:  server.RegisterConfigListServer(s, nil),
			Manager:  server.RegisterManagerServer(s, conf.ManagerReqCh),
		},
		startErrorCh: startErrorCh,
	}, startErrorCh, nil
}

func (s *Server) loop() {
	for {
		select {
		case dep := <-s.reload:
			log.Printf("[grpc] hot reloading control interfaces\n")
			s.hotReload(dep)
		case err := <-s.errorCh:
			log.Printf("[grpc] grpc error: %+v\n", err)
			s.cancelFn()
			return
		case <-s.ctx.Done():
			log.Printf("[grpc] stopping grpc server\n")
			s.server.GracefulStop()
			return
		}
	}
}

func (s *Server) Serve() {
	s.ctx, s.cancelFn = context.WithCancel(context.Background())
	defer s.cancelFn()

	lis, err := net.Listen("tcp", "127.0.0.1:9963")
	if err != nil {
		log.Printf("[grpc] Failed to listen for connections: %+v\n", err)
		s.unrecoverable = true
		s.startErrorCh <- err
		return
	}

	go s.loop()
	go func() {
		log.Printf("[grpc] grpc server available at 127.0.0.1:9963\n")
		s.errorCh <- s.server.Serve(lis)
	}()

	<-s.ctx.Done()
}

func (s *Server) IsCompletable() bool {
	return !s.unrecoverable
}

func (s *Server) Stop() {
	log.Println("[grpc] stopper grpc server")
	s.cancelFn()
}

func (s *Server) hotReload(dep *controller.Dependencies) {
	s.servers.Battery.HotReload(dep.Battery)
	s.servers.Keyboard.HotReload(dep.Keyboard)
	s.servers.Configs.HotReload(dep.Updatable)
	dep.ConfigRegistry.Register(s.servers.Configs)
}
