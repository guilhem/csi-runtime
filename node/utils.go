package node

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	fsTypeXFS = "xfs"
)

// // MergeMountOptions returns only unique mount options in slice.
// func MergeMountOptions(mountOptions []string, volCap *csi.VolumeCapability) []string {
// 	if m := volCap.GetMount(); m != nil {
// 		for _, f := range m.MountFlags {
// 			mountOptions = slice.AppendIfAbsent(mountOptions, f)
// 		}
// 	}

// 	return mountOptions
// }

func Readonly(vc *csi.VolumeCapability) bool {
	if vc.GetAccessMode() == nil {
		return false
	}
	mode := vc.GetAccessMode().GetMode()
	return (mode == csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY ||
		mode == csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY)
}
