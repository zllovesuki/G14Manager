package server

import (
	"context"

	"github.com/zllovesuki/G14Manager/rpc/protocol"

	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type BatteryServer struct {
	protocol.UnimplementedBatteryChargeLimitServer
}

var _ protocol.BatteryChargeLimitServer = &BatteryServer{}

func New(s *grpc.Server) {
	protocol.RegisterBatteryChargeLimitServer(s, &BatteryServer{})
}

func (b *BatteryServer) GetCurrent(ctx context.Context, _ *empty.Empty) (*protocol.BatteryChargeLimitResponse, error) {
	return nil, nil
}

func (b *BatteryServer) Set(ctx context.Context, req *protocol.SetBatteryLimitRequest) (*protocol.BatteryChargeLimitResponse, error) {
	return nil, nil
}
