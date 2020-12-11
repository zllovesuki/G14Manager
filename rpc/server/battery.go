package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/system/battery"

	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type BatteryServer struct {
	protocol.UnimplementedBatteryChargeLimitServer

	mu      sync.RWMutex
	control *battery.ChargeLimit
}

var _ protocol.BatteryChargeLimitServer = &BatteryServer{}

func RegisterBatteryChargeLimitServer(s *grpc.Server, ctrl *battery.ChargeLimit) *BatteryServer {
	server := &BatteryServer{
		control: ctrl,
	}
	protocol.RegisterBatteryChargeLimitServer(s, server)
	return server
}

func (b *BatteryServer) GetCurrentLimit(ctx context.Context, _ *empty.Empty) (*protocol.BatteryChargeLimitResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.control == nil {
		return nil, fmt.Errorf("battery server is not initialized")
	}

	resp := &protocol.BatteryChargeLimitResponse{
		Success:    true,
		Percentage: uint32(b.control.CurrentLimit()),
	}

	return resp, nil
}

func (b *BatteryServer) Set(ctx context.Context, req *protocol.SetBatteryLimitRequest) (*protocol.BatteryChargeLimitResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.control == nil {
		return nil, fmt.Errorf("battery server is not initialized")
	}

	resp := &protocol.BatteryChargeLimitResponse{}
	err := b.control.Set(uint8(req.GetPercentage()))
	if err != nil {
		resp.Success = false
		resp.Percentage = uint32(b.control.CurrentLimit())
		resp.Message = err.Error()
	} else {
		resp.Success = true
		resp.Percentage = uint32(b.control.CurrentLimit())
	}
	return resp, nil
}

func (b *BatteryServer) HotReload(ctrl *battery.ChargeLimit) {
	b.mu.Lock()
	defer b.mu.Unlock()

	log.Println("[gRPCServer] hot reloading battery server")

	b.control = ctrl
}
