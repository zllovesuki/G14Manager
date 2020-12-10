package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/system/persist"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	managerPersistName = "Manager"
)

type ManagerRequestType int

const (
	RequestCheckState ManagerRequestType = iota
	RequestStartController
	RequestStopController
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
	protocol.UnimplementedManagerControlServer

	control chan ManagerSupervisorRequest

	mu        sync.RWMutex
	autoStart bool
}

var _ protocol.ManagerControlServer = &ManagerServer{}

func RegisterManagerServer(s *grpc.Server, ctrl chan ManagerSupervisorRequest) *ManagerServer {
	server := &ManagerServer{
		control: ctrl,
	}
	protocol.RegisterManagerControlServer(s, server)
	return server
}

func (m *ManagerServer) GetCurrentState(ctx context.Context, req *emptypb.Empty) (*protocol.ManagerControlResponse, error) {
	respChan := make(chan ManagerSupervisorResponse)
	supervisorReq := ManagerSupervisorRequest{
		Request:  RequestCheckState,
		Response: respChan,
	}
	m.control <- supervisorReq
	resp := <-respChan
	return &protocol.ManagerControlResponse{
		Success: true,
		State:   resp.State,
	}, nil
}

func (m *ManagerServer) Control(ctx context.Context, req *protocol.ManagerControlRequest) (*protocol.ManagerControlResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}
	respChan := make(chan ManagerSupervisorResponse)
	supervisorReq := ManagerSupervisorRequest{
		Response: respChan,
	}
	if req.GetState() == protocol.ManagerControlRequest_START {
		supervisorReq.Request = RequestStartController
	} else {
		supervisorReq.Request = RequestStopController
	}
	m.control <- supervisorReq
	resp := <-respChan
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
		return nil
	}

	var autoStart bool
	buf := bytes.NewReader(v)
	if err := binary.Read(buf, binary.BigEndian, &autoStart); err != nil {
		return err
	}

	m.autoStart = autoStart

	return nil
}

func (m *ManagerServer) Apply() error {
	return nil
}

func (m *ManagerServer) Close() error {
	return nil
}
