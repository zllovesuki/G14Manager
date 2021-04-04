package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"sync"

	"github.com/zllovesuki/G14Manager/rpc/announcement"
	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/thermal"

	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

const (
	thermalPersistKey = "Thermal"
)

var thermalOnce sync.Once

type ThermalServer struct {
	protocol.UnimplementedThermalServer

	mu        sync.RWMutex
	control   *thermal.Control
	updatable []announcement.Updatable
	profiles  []thermal.Profile
}

var _ protocol.ThermalServer = &ThermalServer{}

func RegisterThermalServer(s *grpc.Server, ctrl *thermal.Control, u []announcement.Updatable) *ThermalServer {
	server := &ThermalServer{
		control:   ctrl,
		updatable: u,
		profiles:  thermal.GetDefaultThermalProfiles(),
	}
	protocol.RegisterThermalServer(s, server)
	return server
}

func (t *ThermalServer) GetCurrentProfile(ctx context.Context, _ *empty.Empty) (*protocol.ThermalResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.control.CurrentProfile()

	return &protocol.ThermalResponse{
		Success: true,
		Profiles: []*protocol.Profile{
			{
				Name:             current.Name,
				WindowsPowerPlan: current.WindowsPowerPlan,
				ThrottlePlan:     toProtoThrottle(current.ThrottlePlan),
				CPUFanCurve:      current.CPUFanCurve.String(),
				GPUFanCurve:      current.GPUFanCurve.String(),
			},
		},
	}, nil
}

func (t *ThermalServer) ListProfiles(ctx context.Context, _ *empty.Empty) (*protocol.ThermalResponse, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	resp := &protocol.ThermalResponse{
		Success:  true,
		Profiles: make([]*protocol.Profile, 0, len(t.profiles)),
	}

	for _, current := range t.profiles {
		resp.Profiles = append(resp.Profiles, &protocol.Profile{
			Name:             current.Name,
			WindowsPowerPlan: current.WindowsPowerPlan,
			ThrottlePlan:     toProtoThrottle(current.ThrottlePlan),
			CPUFanCurve:      current.CPUFanCurve.String(),
			GPUFanCurve:      current.GPUFanCurve.String(),
		})
	}

	return resp, nil
}

func (t *ThermalServer) UpdateProfiles(ctx context.Context, req *protocol.UpdateProfileRequest) (*protocol.ThermalResponse, error) {
	t.annouceProfiles()
	return nil, nil
}

func (t *ThermalServer) SetProfile(ctx context.Context, req *protocol.SetProfileRequest) (*protocol.ThermalResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	_, err := t.control.SwitchToProfile(req.GetProfileName())
	if err != nil {
		return &protocol.ThermalResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	current := t.control.CurrentProfile()

	return &protocol.ThermalResponse{
		Success: true,
		Profiles: []*protocol.Profile{
			{
				Name:             current.Name,
				WindowsPowerPlan: current.WindowsPowerPlan,
				ThrottlePlan:     toProtoThrottle(current.ThrottlePlan),
				CPUFanCurve:      current.CPUFanCurve.String(),
				GPUFanCurve:      current.GPUFanCurve.String(),
			},
		},
	}, nil

}

func (t *ThermalServer) annouceProfiles() {
	profilesUpdate := announcement.Update{
		Type:   announcement.ProfilesUpdate,
		Config: nil,
	}
	for _, updatable := range t.updatable {
		log.Printf("[gRPCServer] notifying \"%s\" about profiles update", updatable.Name())
		go updatable.ConfigUpdate(profilesUpdate)
	}
}

var _ persist.Registry = &ThermalServer{}

func (t *ThermalServer) Name() string {
	return thermalPersistKey
}

type thermalPersistMap struct {
	Profiles []thermal.Profile
	Current  string
}

func (t *ThermalServer) Value() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(thermalPersistMap{
		Profiles: t.profiles,
		Current:  t.control.CurrentProfile().Name,
	}); err != nil {
		return nil
	}

	return buf.Bytes()
}

func (t *ThermalServer) Load(v []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	thermalOnce.Do(func() {
		if len(v) == 0 {
			return
		}

		var p thermalPersistMap
		buf := bytes.NewBuffer(v)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&p); err != nil {
			return
		}

		t.profiles = p.Profiles
		t.annouceProfiles()
		t.control.SwitchToProfile(p.Current)
		log.Printf("[gRPCServer] thermal profile set to %s\n", p.Current)
	})

	return nil
}

func (t *ThermalServer) Apply() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	t.annouceProfiles()

	return nil
}

func (t *ThermalServer) Close() error {
	return nil
}

func toProtoThrottle(v uint32) protocol.Profile_ThrottleValue {
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
