package config

import (
	"github.com/guilhem/csi-runtime/node"
	"github.com/guilhem/csi-runtime/storage"
	"k8s.io/mount-utils"
)

type Options struct {
	// DriverName should match the driver name as configured in the Kubernetes
	// CSIDriver object (e.g. 'csi.cert-manager.io')
	DriverName string
	// DriverVersion is the version of the driver to be returned during
	// IdentityServer calls
	DriverVersion string
	// NodeID is the name/ID of the node this driver is running on (typically
	// the Kubernetes node name)
	NodeID string
	// Store is a reference to a storage backend for writing files
	Store storage.Interface
	// Manager is used to fetch & renew certificate data
	NodeManager node.Interface
	// Mounter will be used to invoke operating system mount operations.
	// If not specified, the current operating system's default implementation
	// will be used (i.e. 'mount.New("")')
	Mounter mount.Interface

	MaxVolumesPerNode int64
}
