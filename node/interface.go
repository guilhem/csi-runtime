package node

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

type Interface interface {
}

type Validate interface {
	ValidateCapability(vCap *csi.VolumeCapability) error
}

type Mount interface {
	Mount(ctx context.Context, path string, fstype string, volumeContext map[string]string, controllerContext map[string]string, secrets map[string]string) (*MountResponse, error)
}

type Populate interface {
	Populate(ctx context.Context, path string, volumeContext map[string]string, controllerContext map[string]string, secrets map[string]string) error
}

type Block interface {
	AttachBlock(ctx context.Context, volumeContext map[string]string, controllerContext map[string]string, secrets map[string]string) (string, error)
	DetachBlock(ctx context.Context, path string) error
}

type Expand interface {
	Expand()
}
