package server

import (
	"context"

	"github.com/zllovesuki/G14Manager/rpc/protocol"

	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type KeyboardServer struct {
	protocol.UnimplementedKeyboardBrightnessServer
}

var _ protocol.KeyboardBrightnessServer = &KeyboardServer{}

func NewKeyboardServer(s *grpc.Server) {
	protocol.RegisterKeyboardBrightnessServer(s, &KeyboardServer{})
}

func (k *KeyboardServer) GetCurrent(ctx context.Context, _ *empty.Empty) (*protocol.KeyboardBrightnessResponse, error) {
	return nil, nil
}

func (k *KeyboardServer) Set(ctx context.Context, req *protocol.SetKeyboardBrightnessRequest) (*protocol.KeyboardBrightnessResponse, error) {
	return nil, nil
}

func (k *KeyboardServer) Change(ctx context.Context, req *protocol.ChangeKeyboardBrightnessRequest) (*protocol.KeyboardBrightnessResponse, error) {
	return nil, nil
}
