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

package driver

import (
	"net"

	"github.com/go-logr/logr"

	"github.com/guilhem/csi-runtime/controller"
	"github.com/guilhem/csi-runtime/identity"
	"github.com/guilhem/csi-runtime/node"
	"github.com/guilhem/csi-runtime/server"
	"github.com/guilhem/csi-runtime/utils"
)

// A Driver is a gRPC server that implements the CSI spec.
// It can be used to build a CSI driver that generates private key data and
// automatically creates cert-manager CertificateRequests to obtain signed
// certificate data.
type Driver struct {
	server *server.Server
}

func New(endpoint, name, version string, idm identity.Interface, cs *controller.Server, ns *node.Server, log logr.Logger) (*Driver, error) {

	lis, err := utils.ListenerFromUrl(endpoint)
	if err != nil {
		return nil, err
	}
	return NewWithListener(lis, name, version, idm, cs, ns, log)
}

// NewWithListener will construct a new CSI driver using the given net.Listener.
// This is useful when more control over the listening parameters is required.
func NewWithListener(lis net.Listener, name, version string, idm identity.Interface, cm controller.Interface, ns *node.Server, log logr.Logger) (*Driver, error) {

	var cs *controller.Server

	ids := identity.New(name, version, cm != nil, idm)

	if cm != nil {
		cs = controller.New(cm)
	}

	// storage.NewFilesystem(log, baseDir string)

	serv := server.New(lis, log, ids, cs, ns)

	return &Driver{server: serv}, nil
}

func (d *Driver) Run() error {
	return d.server.Run()
}

func (d *Driver) Stop() {
	d.server.Stop()
}
