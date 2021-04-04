package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/system/persist"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	managerPersistName = "Manager"
)

var managerOnce sync.Once

type ManagerRequestType int

const (
	RequestCheckState ManagerRequestType = iota
	RequestStartController
	RequestStopController
	RequestSaveConfig
)

type ManagerSupervisorRequest struct {
	Request  ManagerRequestType
	Response chan ManagerSupervisorResponse
}

type ManagerSupervisorResponse struct {
	State protocol.ManagerControlResponse_CurrentState
	Error error
}

type ManagerServer struct {
	protocol.UnimplementedManagerServer

	control chan ManagerSupervisorRequest

	mu        sync.RWMutex
	autoStart bool
}

var _ protocol.ManagerServer = &ManagerServer{}

func RegisterManagerServer(s *grpc.Server, ctrl chan ManagerSupervisorRequest) *ManagerServer {
	server := &ManagerServer{
		control: ctrl,
	}
	protocol.RegisterManagerServer(s, server)
	return server
}

func (m *ManagerServer) GetCurrentAutoStart(ctx context.Context, req *emptypb.Empty) (*protocol.ManagerAutoStartResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &protocol.ManagerAutoStartResponse{
		Success:   true,
		AutoStart: m.autoStart,
	}, nil
}

func (m *ManagerServer) waitForResponder(ctx context.Context, req ManagerRequestType) ManagerSupervisorResponse {
	respChan := make(chan ManagerSupervisorResponse)
	m.control <- ManagerSupervisorRequest{
		Request:  req,
		Response: respChan,
	}
	select {
	case <-ctx.Done():
		return ManagerSupervisorResponse{
			Error: ctx.Err(),
		}
	case <-time.After(time.Second * 5):
		return ManagerSupervisorResponse{
			Error: context.DeadlineExceeded,
		}
	case resp := <-respChan:
		return resp
	}
}

func (m *ManagerServer) SetAutoStart(ctx context.Context, req *protocol.ManagerAutoStartRequest) (*protocol.ManagerAutoStartResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.autoStart = req.GetAutoStart()

	go func() {
		resp := m.waitForResponder(context.Background(), RequestSaveConfig)
		if resp.Error != nil {
			log.Printf("[gRPCServer] unable to save config: %+v\n", resp.Error)
		}
	}()

	return &protocol.ManagerAutoStartResponse{
		Success:   true,
		AutoStart: m.autoStart,
	}, nil
}

func (m *ManagerServer) GetCurrentState(ctx context.Context, req *emptypb.Empty) (*protocol.ManagerControlResponse, error) {
	resp := m.waitForResponder(ctx, RequestCheckState)
	return &protocol.ManagerControlResponse{
		Success: true,
		State:   resp.State,
	}, nil
}

func (m *ManagerServer) Control(ctx context.Context, req *protocol.ManagerControlRequest) (*protocol.ManagerControlResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}
	var r ManagerRequestType
	if req.GetState() == protocol.ManagerControlRequest_START {
		r = RequestStartController
	} else {
		r = RequestStopController
	}
	resp := m.waitForResponder(ctx, r)
	if resp.Error != nil {
		return &protocol.ManagerControlResponse{
			Success: false,
			State:   resp.State,
			Message: resp.Error.Error(),
		}, nil
	}
	return &protocol.ManagerControlResponse{
		Success: true,
		State:   resp.State,
	}, nil
}

var _ persist.Registry = &ManagerServer{}

func (m *ManagerServer) Name() string {
	return managerPersistName
}

func (m *ManagerServer) Value() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, m.autoStart); err != nil {
		return nil
	}

	return buf.Bytes()
}

func (m *ManagerServer) Load(v []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(v) == 0 {
		// mainly for new installs
		m.autoStart = true
		log.Println("[gRPCServer] set autoStart to true on new install")
	} else {
		var autoStart bool
		buf := bytes.NewReader(v)
		if err := binary.Read(buf, binary.BigEndian, &autoStart); err != nil {
			return err
		}
		m.autoStart = autoStart
	}

	go managerOnce.Do(func() {
		if !m.autoStart {
			log.Println("[gRPCServer] not auto starting controller")
			return
		}
		log.Println("[gRPCServer] auto starting controller")
		resp := m.waitForResponder(context.Background(), RequestStartController)
		if resp.Error != nil {
			log.Printf("[gRPCServer] cannot auto start controller: %+v\n", resp.Error)
		} else {
			log.Println("[gRPCServer] controller started")
		}
	})

	return nil
}

func (m *ManagerServer) Apply() error {
	return nil
}

func (m *ManagerServer) Close() error {
	return nil
}
