package node

import (
	"context"

	"k8s.io/mount-utils"
)

type Volume struct {
	StagingPath   string
	PublishedPath string
}

func NewVolume()

func (v *Volume) Mount(ctx context.Context, mounter mount.Interface) {

}
