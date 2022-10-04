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

package identity

import (
	"log"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Server struct {
	name       string
	version    string
	controller bool

	manager Interface
}

func New(name, version string, controller bool, manager Interface) *Server {
	return &Server{
		name:       name,
		version:    version,
		manager:    manager,
		controller: controller,
	}
}

func (ids *Server) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	if ids.name == "" {
		return nil, status.Error(codes.Unavailable, "driver name not configured")
	}

	if ids.version == "" {
		return nil, status.Error(codes.Unavailable, "driver is missing version")
	}

	return &csi.GetPluginInfoResponse{
		Name:          ids.name,
		VendorVersion: ids.version,
	}, nil
}

func (ids *Server) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	if p, ok := ids.manager.(Probe); ok {
		ready, err := p.Probe(ctx)
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				return nil, st.Err()
			}

			return nil, status.Error(codes.Internal, err.Error())
		}

		return &csi.ProbeResponse{Ready: wrapperspb.Bool(ready)}, nil
	}

	return &csi.ProbeResponse{Ready: wrapperspb.Bool(true)}, nil
}

func (ids *Server) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	cap := make([]*csi.PluginCapability, 0)

	if ids.controller {

		log.Print("PluginCapability_Service_CONTROLLER_SERVICE")

		cap = append(cap, &csi.PluginCapability{
			Type: &csi.PluginCapability_Service_{
				Service: &csi.PluginCapability_Service{
					Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
				},
			},
		})
	}

	return &csi.GetPluginCapabilitiesResponse{Capabilities: cap}, nil
}
