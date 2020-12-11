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
	"github.com/zllovesuki/G14Manager/system/keyboard"
	"github.com/zllovesuki/G14Manager/system/persist"
	"github.com/zllovesuki/G14Manager/system/shared"
	"github.com/zllovesuki/G14Manager/system/thermal"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	featuresPersistName = "Configs"
)

var configOnce sync.Once

type ConfigListServer struct {
	protocol.UnimplementedConfigListServer

	mu        sync.RWMutex
	updatable []announcement.Updatable
	features  shared.Features
	profiles  []thermal.Profile
}

var _ protocol.ConfigListServer = &ConfigListServer{}

func RegisterConfigListServer(s *grpc.Server, u []announcement.Updatable) *ConfigListServer {
	server := &ConfigListServer{
		updatable: u,
		// sensible defaults
		features: shared.Features{
			FnRemap: map[uint32]uint16{
				keyboard.KeyFnLeft:  keyboard.KeyPgUp,
				keyboard.KeyFnRight: keyboard.KeyPgDown,
			},
			AutoThermal: shared.AutoThermal{
				Enabled: false,
			},
			RogRemap: []string{"Taskmgr.exe"},
		},
		profiles: thermal.GetDefaultThermalProfiles(),
	}
	protocol.RegisterConfigListServer(s, server)
	return server
}

func (f *ConfigListServer) GetCurrentConfigs(ctx context.Context, req *emptypb.Empty) (*protocol.SetConfigsResponse, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	fnRemap := make(map[uint32]uint32)
	for k, v := range f.features.FnRemap {
		fnRemap[k] = uint32(v)
	}
	profiles := make([]*protocol.Profile, 0, 3)
	for _, p := range f.profiles {
		var val protocol.Profile_ThrottleValue
		switch p.ThrottlePlan {
		case thermal.ThrottlePlanPerformance:
			val = protocol.Profile_PERFORMANCE
		case thermal.ThrottlePlanSilent:
			val = protocol.Profile_SILENT
		case thermal.ThrottlePlanTurbo:
			val = protocol.Profile_TURBO
		}
		profiles = append(profiles, &protocol.Profile{
			Name:             p.Name,
			WindowsPowerPlan: p.WindowsPowerPlan,
			ThrottlePlan:     val,
			CPUFanCurve:      p.CPUFanCurve.String(),
			GPUFanCurve:      p.GPUFanCurve.String(),
		})
	}
	return &protocol.SetConfigsResponse{
		Success: true,
		Configs: &protocol.Configs{
			Features: &protocol.Features{
				AutoThermal: &protocol.AutoThermal{
					Enabled:          f.features.AutoThermal.Enabled,
					PluggedInProfile: f.features.AutoThermal.PluggedIn,
					UnpluggedProfile: f.features.AutoThermal.Unplugged,
				},
				FnRemap:  fnRemap,
				RogRemap: f.features.RogRemap,
			},
			Profiles: profiles,
		},
	}, nil
}

func (f *ConfigListServer) Set(ctx context.Context, req *protocol.SetConfigsRequest) (*protocol.SetConfigsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	configs := req.GetConfigs()
	if configs == nil {
		return nil, fmt.Errorf("nil configs is invalid")
	}

	feats := configs.GetFeatures()
	profiles := configs.GetProfiles()

	if feats == nil && profiles == nil {
		return nil, fmt.Errorf("either features or profiles must be specified")
	}

	var newFeatures *shared.Features
	var newProfiles []thermal.Profile

	if feats != nil {
		fnRemap := make(map[uint32]uint16)
		for k, v := range feats.FnRemap {
			fnRemap[k] = uint16(v)
		}
		newFeatures = &shared.Features{
			AutoThermal: shared.AutoThermal{
				Enabled:   feats.AutoThermal.Enabled,
				PluggedIn: feats.AutoThermal.PluggedInProfile,
				Unplugged: feats.AutoThermal.UnpluggedProfile,
			},
			FnRemap:  fnRemap,
			RogRemap: feats.GetRogRemap(),
		}
	}

	if profiles != nil {
		newProfiles = make([]thermal.Profile, 0)
		for _, p := range profiles {
			// TODO: verify windows power plan existence
			if p.GetName() == "" {
				return nil, fmt.Errorf("Profile name must not be empty")
			}
			var err error
			var val uint32
			switch p.GetThrottlePlan() {
			case protocol.Profile_PERFORMANCE:
				val = thermal.ThrottlePlanPerformance
			case protocol.Profile_SILENT:
				val = thermal.ThrottlePlanSilent
			case protocol.Profile_TURBO:
				val = thermal.ThrottlePlanTurbo
			default:
				return nil, fmt.Errorf("Unrecognized throttle plan in profile")
			}
			profile := thermal.Profile{
				Name:             p.GetName(),
				ThrottlePlan:     val,
				WindowsPowerPlan: p.GetWindowsPowerPlan(),
			}
			if p.GetCPUFanCurve() != "" {
				profile.CPUFanCurve, err = thermal.NewFanTable(p.GetCPUFanCurve())
				if err != nil {
					return nil, fmt.Errorf("CPU fan curve parse error: %s", err.Error())
				}
			}
			if p.GetGPUFanCurve() != "" {
				profile.GPUFanCurve, err = thermal.NewFanTable(p.GetGPUFanCurve())
				if err != nil {
					return nil, fmt.Errorf("GPU fan curve parse error: %s", err.Error())
				}
			}
			newProfiles = append(newProfiles, profile)
		}
	}

	if newFeatures != nil && newFeatures.AutoThermal.Enabled && len(newProfiles) > 0 {
		var validPluggedInProfile bool
		var validUnpluggedProfile bool
		for _, p := range newProfiles {
			if p.Name == newFeatures.AutoThermal.PluggedIn {
				validPluggedInProfile = true
			}
			if p.Name == newFeatures.AutoThermal.Unplugged {
				validUnpluggedProfile = true
			}
		}
		if !validPluggedInProfile || !validUnpluggedProfile {
			return nil, fmt.Errorf("AutoThermal must specify a valid profile if enabled")
		}
	}

	if newFeatures != nil {
		fmt.Println("[gRPCServer] updating features config")
		f.features = *newFeatures
	}
	if len(newProfiles) > 0 {
		fmt.Println("[gRPCServer] updating profiles config")
		f.profiles = newProfiles
	}

	f.announceConfigs()

	return &protocol.SetConfigsResponse{
		Success: true,
		Configs: req.GetConfigs(),
	}, nil
}

func (f *ConfigListServer) announceConfigs() {
	featsUpdate := announcement.Update{
		Type:   announcement.FeaturesUpdate,
		Config: f.features,
	}
	profilesUpdate := announcement.Update{
		Type:   announcement.ProfilesUpdate,
		Config: f.profiles,
	}
	for _, updatable := range f.updatable {
		log.Printf("[gRPCServer] notifying \"%s\" about features update", updatable.Name())
		go updatable.ConfigUpdate(featsUpdate)
		log.Printf("[gRPCServer] notifying \"%s\" about profiles update", updatable.Name())
		go updatable.ConfigUpdate(profilesUpdate)
	}
}

func (f *ConfigListServer) HotReload(u []announcement.Updatable) {
	f.mu.Lock()
	defer f.mu.Unlock()

	log.Println("[gRPCServer] hot reloading configs server")

	f.updatable = u
	f.announceConfigs()
}

var _ persist.Registry = &ConfigListServer{}

func (f *ConfigListServer) Name() string {
	return featuresPersistName
}

type persistMap struct {
	Features shared.Features
	Profiles []thermal.Profile
}

func (f *ConfigListServer) Value() []byte {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(persistMap{
		Features: f.features,
		Profiles: f.profiles,
	}); err != nil {
		return nil
	}

	return buf.Bytes()
}

func (f *ConfigListServer) Load(v []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	configOnce.Do(func() {
		if len(v) == 0 {
			return
		}

		var p persistMap
		buf := bytes.NewBuffer(v)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&p); err != nil {
			return
		}

		f.features = p.Features
		f.profiles = p.Profiles

		f.announceConfigs()
	})

	return nil
}

func (f *ConfigListServer) Apply() error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.announceConfigs()

	return nil
}

func (f *ConfigListServer) Close() error {
	return nil
}
