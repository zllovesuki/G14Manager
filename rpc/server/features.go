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

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	featuresPersistName = "Features"
)

var featuresOnce sync.Once

type FeaturesServer struct {
	protocol.UnimplementedFeaturesServer

	mu        sync.RWMutex
	updatable []announcement.Updatable
	features  shared.Features
}

var _ protocol.FeaturesServer = &FeaturesServer{}

func RegisterFeaturesServer(s *grpc.Server, u []announcement.Updatable) *FeaturesServer {
	server := &FeaturesServer{
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
	}
	protocol.RegisterFeaturesServer(s, server)
	return server
}

func (f *FeaturesServer) GetCurrentFeatures(ctx context.Context, req *emptypb.Empty) (*protocol.FeaturesResponse, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	fnRemap := make(map[uint32]uint32)
	for k, v := range f.features.FnRemap {
		fnRemap[k] = uint32(v)
	}
	return &protocol.FeaturesResponse{
		Success: true,
		Feature: &protocol.Feature{
			AutoThermal: &protocol.AutoThermal{
				Enabled:          f.features.AutoThermal.Enabled,
				PluggedInProfile: f.features.AutoThermal.PluggedIn,
				UnpluggedProfile: f.features.AutoThermal.Unplugged,
			},
			FnRemap:  fnRemap,
			RogRemap: f.features.RogRemap,
		},
	}, nil
}

func (f *FeaturesServer) UpdateFeatures(ctx context.Context, req *protocol.UpdateFeaturesRequest) (*protocol.FeaturesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request is invalid")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	feats := req.GetFeature()

	if feats == nil {
		return nil, fmt.Errorf("features must be specified")
	}

	newFeatures := f.features

	if feats.GetAutoThermal() != nil {
		// TODO: Validate input
		newFeatures.AutoThermal = shared.AutoThermal{
			Enabled:   feats.GetAutoThermal().Enabled,
			PluggedIn: feats.GetAutoThermal().PluggedInProfile,
			Unplugged: feats.GetAutoThermal().UnpluggedProfile,
		}
	}
	if len(feats.GetRogRemap()) > 0 {
		newFeatures.RogRemap = feats.GetRogRemap()
	}
	if feats.GetFnRemap() != nil {
		fnRemap := make(map[uint32]uint16)
		for k, v := range feats.GetFnRemap() {
			fnRemap[k] = uint16(v)
		}
	}

	f.features = newFeatures
	f.announceFeatures()

	return &protocol.FeaturesResponse{
		Success: true,
		Feature: req.GetFeature(),
	}, nil
}

func (f *FeaturesServer) announceFeatures() {
	featsUpdate := announcement.Update{
		Type:   announcement.FeaturesUpdate,
		Config: f.features,
	}
	for _, updatable := range f.updatable {
		log.Printf("[gRPCServer] notifying \"%s\" about features update", updatable.Name())
		go updatable.ConfigUpdate(featsUpdate)
	}
}

var _ persist.Registry = &FeaturesServer{}

func (f *FeaturesServer) Name() string {
	return featuresPersistName
}

type featurePersistMap struct {
	Features shared.Features
}

func (f *FeaturesServer) Value() []byte {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(featurePersistMap{
		Features: f.features,
	}); err != nil {
		return nil
	}

	return buf.Bytes()
}

func (f *FeaturesServer) Load(v []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	featuresOnce.Do(func() {
		if len(v) == 0 {
			return
		}

		var p featurePersistMap
		buf := bytes.NewBuffer(v)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&p); err != nil {
			return
		}

		f.features = p.Features

		f.announceFeatures()
	})

	return nil
}

func (f *FeaturesServer) Apply() error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.announceFeatures()

	return nil
}

func (f *FeaturesServer) Close() error {
	return nil
}
