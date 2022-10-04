/*
Copyright 2018 The Kubernetes Authors.
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

package server

import (
	"net"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-logr/logr"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Defines Non blocking GRPC server interfaces
type NonBlockingGRPCServer interface {
	// Start services at the endpoint
	Start() error
	// Stops the service gracefully
	Stop()
	// Stops the service forcefully
	ForceStop()
}

type server struct {
	server *grpc.Server
	wg     sync.WaitGroup
	lis    net.Listener
	log    logr.Logger
}

func New(lis net.Listener, log logr.Logger, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) NonBlockingGRPCServer {

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(loggingInterceptor(log)),
	}
	s := grpc.NewServer(opts...)

	// if ids != nil {
	csi.RegisterIdentityServer(s, ids)
	// }

	// if cs != nil {
	csi.RegisterControllerServer(s, cs)
	// }

	// if ns != nil {
	csi.RegisterNodeServer(s, ns)
	// }

	return &server{
		server: s,
		lis:    lis,
		log:    log,
	}
}

func (s *server) Start() error {
	return s.server.Serve(s.lis)
}

func (s *server) Stop() {
	s.server.GracefulStop()
}

func (s *server) ForceStop() {
	s.server.Stop()
}

func loggingInterceptor(log logr.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log := log.WithValues("rpc_method", info.FullMethod, "request", protosanitizer.StripSecrets(req))
		log.V(3).Info("handling request")
		resp, err := handler(ctx, req)
		if err != nil {
			log.Error(err, "failed processing request")
		} else {
			log.V(5).Info("request completed", "response", protosanitizer.StripSecrets(resp))
		}
		return resp, err
	}
}
