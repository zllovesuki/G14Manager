package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/system/thermal"

	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type ThermalServer struct {
	protocol.UnimplementedThermalServer

	mu      sync.RWMutex
	control *thermal.Control
}

var _ protocol.ThermalServer = &ThermalServer{}

func RegisterThermalServer(s *grpc.Server, ctrl *thermal.Control) *ThermalServer {
	server := &ThermalServer{
		control: ctrl,
	}
	protocol.RegisterThermalServer(s, server)
	return server
}

func (t *ThermalServer) GetCurrentProfile(ctx context.Context, _ *empty.Empty) (*protocol.SetProfileResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.control.CurrentProfile()

	return &protocol.SetProfileResponse{
		Success: true,
		Profile: &protocol.Profile{
			Name:             current.Name,
			WindowsPowerPlan: current.WindowsPowerPlan,
			ThrottlePlan:     toProtoThrottlw(current.ThrottlePlan),
			CPUFanCurve:      current.CPUFanCurve.String(),
			GPUFanCurve:      current.GPUFanCurve.String(),
		},
	}, nil
}

func (t *ThermalServer) Set(ctx context.Context, req *protocol.SetProfileRequest) (*protocol.SetProfileResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	_, err := t.control.SwitchToProfile(req.GetProfileName())
	if err != nil {
		return &protocol.SetProfileResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	current := t.control.CurrentProfile()

	return &protocol.SetProfileResponse{
		Success: true,
		Profile: &protocol.Profile{
			Name:             current.Name,
			WindowsPowerPlan: current.WindowsPowerPlan,
			ThrottlePlan:     toProtoThrottlw(current.ThrottlePlan),
			CPUFanCurve:      current.CPUFanCurve.String(),
			GPUFanCurve:      current.GPUFanCurve.String(),
		},
	}, nil

}

func (t *ThermalServer) HotReload(ctrl *thermal.Control) {
	t.mu.Lock()
	defer t.mu.Unlock()

	log.Println("[gRPCServer] hot reloading thermal server")

	t.control = ctrl
}

func toProtoThrottlw(v uint32) protocol.Profile_ThrottleValue {
	var val protocol.Profile_ThrottleValue
	switch v {
	case thermal.ThrottlePlanPerformance:
		val = protocol.Profile_PERFORMANCE
	case thermal.ThrottlePlanSilent:
		val = protocol.Profile_SILENT
	case thermal.ThrottlePlanTurbo:
		val = protocol.Profile_TURBO
	}
	return val
}
