/*
Copyright 2021 The cert-manager Authors.
Copyright 2022 Guilhem Lettron <guilhem@barpilot.io>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	manager Interface
}

func New(m Interface) *Server {
	return &Server{manager: m}
}

func (cs *Server) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	if s, ok := cs.manager.(GET_VOLUME); ok {
		return s.ControllerGetVolume(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "ControllerGetVolume not implemented")
}

func (cs *Server) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if s, ok := cs.manager.(CREATE_DELETE_VOLUME); ok {
		return s.CreateVolume(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "CreateVolume not implemented")
}

func (cs *Server) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if s, ok := cs.manager.(CREATE_DELETE_VOLUME); ok {
		return s.DeleteVolume(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "DeleteVolume not implemented")
}

func (cs *Server) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	if s, ok := cs.manager.(PUBLISH_UNPUBLISH_VOLUME); ok {
		return s.ControllerPublishVolume(ctx, req)
	}

	return nil, status.Error(codes.Unimplemented, "ControllerPublishVolume not implemented")
}

func (cs *Server) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	if s, ok := cs.manager.(PUBLISH_UNPUBLISH_VOLUME); ok {
		return s.ControllerUnpublishVolume(ctx, req)
	}

	return nil, status.Error(codes.Unimplemented, "ControllerUnpublishVolume not implemented")
}

func (cs *Server) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ValidateVolumeCapabilities not implemented")
}

func (cs *Server) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	if s, ok := cs.manager.(LIST_VOLUME); ok {
		return s.ListVolumes(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "ListVolumes not implemented")
}

func (cs *Server) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	if s, ok := cs.manager.(GET_CAPACITY); ok {
		return s.GetCapacity(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "GetCapacity not implemented")
}

func (cs *Server) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	if s, ok := cs.manager.(CREATE_DELETE_SNAPSHOT); ok {
		return s.CreateSnapshot(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "CreateSnapshot not implemented")
}

func (cs *Server) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	if s, ok := cs.manager.(CREATE_DELETE_SNAPSHOT); ok {
		return s.DeleteSnapshot(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "DeleteSnapshot not implemented")
}

func (cs *Server) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	if s, ok := cs.manager.(LIST_SNAPSHOT); ok {
		return s.ListSnapshots(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "ListSnapshots not implemented")
}

func (cs *Server) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	if s, ok := cs.manager.(EXPAND_VOLUME); ok {
		return s.ControllerExpandVolume(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "ControllerExpandVolume not implemented")
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *Server) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	cap := make([]*csi.ControllerServiceCapability, 0)

	// Advertise additional capabilities
	if s, ok := cs.manager.(Capabilities); ok {
		addCaps, err := s.GetCapabilities(ctx, req)
		if err != nil {
			return nil, err
		}
		for _, c := range addCaps {
			cap = append(cap, &csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: c,
					},
				},
			})
		}
	}

	if _, ok := cs.manager.(CREATE_DELETE_VOLUME); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME}}})
	}

	if _, ok := cs.manager.(PUBLISH_UNPUBLISH_VOLUME); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME}}})
	}

	if _, ok := cs.manager.(LIST_VOLUME); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES}}})
	}

	if _, ok := cs.manager.(GET_CAPACITY); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_GET_CAPACITY}}})
	}

	if _, ok := cs.manager.(CREATE_DELETE_SNAPSHOT); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT}}})
	}

	if _, ok := cs.manager.(LIST_SNAPSHOT); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS}}})
	}

	if _, ok := cs.manager.(EXPAND_VOLUME); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_EXPAND_VOLUME}}})
	}

	if _, ok := cs.manager.(GET_VOLUME); ok {
		cap = append(cap, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_GET_VOLUME}}})
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cap,
	}, nil
}
