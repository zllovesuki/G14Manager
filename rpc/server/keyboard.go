package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/zllovesuki/G14Manager/cxx/plugin/keyboard"
	"github.com/zllovesuki/G14Manager/rpc/protocol"

	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type KeyboardServer struct {
	protocol.UnimplementedKeyboardBrightnessServer

	mu      sync.RWMutex
	control *keyboard.Control
}

var _ protocol.KeyboardBrightnessServer = &KeyboardServer{}

func RegisterKeyboardServer(s *grpc.Server, ctrl *keyboard.Control) *KeyboardServer {
	server := &KeyboardServer{
		control: ctrl,
	}
	protocol.RegisterKeyboardBrightnessServer(s, server)
	return server
}

func (k *KeyboardServer) GetCurrentBrightness(ctx context.Context, _ *empty.Empty) (*protocol.KeyboardBrightnessResponse, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.control == nil {
		return nil, fmt.Errorf("keyboard server is not initialized")
	}

	level := toProtoLevel(k.control.CurrentBrightness())
	resp := &protocol.KeyboardBrightnessResponse{
		Success:    true,
		Brightness: level,
	}
	return resp, nil
}

func (k *KeyboardServer) Set(ctx context.Context, req *protocol.SetKeyboardBrightnessRequest) (*protocol.KeyboardBrightnessResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.control == nil {
		return nil, fmt.Errorf("keyboard server is not initialized")
	}

	level := fromProtoLevel(req.GetBrightness())
	resp := &protocol.KeyboardBrightnessResponse{}
	setError := k.control.SetBrightness(level)
	if setError != nil {
		resp.Success = false
		resp.Brightness = toProtoLevel(k.control.CurrentBrightness())
		resp.Message = setError.Error()
	} else {
		resp.Success = true
		resp.Brightness = toProtoLevel(k.control.CurrentBrightness())
	}
	return resp, nil
}

func (k *KeyboardServer) Change(ctx context.Context, req *protocol.ChangeKeyboardBrightnessRequest) (*protocol.KeyboardBrightnessResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.control == nil {
		return nil, fmt.Errorf("keyboard server is not initialized")
	}

	var changeErr error
	switch req.GetDirection() {
	case protocol.ChangeKeyboardBrightnessRequest_DECREMENT:
		changeErr = k.control.BrightnessDown()
	case protocol.ChangeKeyboardBrightnessRequest_INCREMENT:
		changeErr = k.control.BrightnessUp()
	}
	resp := &protocol.KeyboardBrightnessResponse{}
	if changeErr != nil {
		resp.Success = false
		resp.Brightness = toProtoLevel(k.control.CurrentBrightness())
		resp.Message = changeErr.Error()
	} else {
		resp.Success = true
		resp.Brightness = toProtoLevel(k.control.CurrentBrightness())
	}
	return resp, nil
}

func (k *KeyboardServer) HotReload(ctrl *keyboard.Control) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.control = ctrl
}

func toProtoLevel(k keyboard.Level) protocol.Level {
	var level protocol.Level
	switch k {
	case keyboard.OFF:
		level = protocol.Level_OFF
	case keyboard.LOW:
		level = protocol.Level_LOW
	case keyboard.MEDIUM:
		level = protocol.Level_MEDIUM
	case keyboard.HIGH:
		level = protocol.Level_HIGH
	}
	return level
}

func fromProtoLevel(p protocol.Level) keyboard.Level {
	var level keyboard.Level
	switch p {
	case protocol.Level_OFF:
		level = keyboard.OFF
	case protocol.Level_LOW:
		level = keyboard.LOW
	case protocol.Level_MEDIUM:
		level = keyboard.MEDIUM
	case protocol.Level_HIGH:
		level = keyboard.HIGH
	}
	return level
}
