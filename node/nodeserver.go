/*
Copyright 2021 The cert-manager Authors.
Copyright 2018 The Kubernetes Authors.
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

package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"

	"github.com/guilhem/csi-runtime/metadata"
	mountmanager "github.com/guilhem/csi-runtime/mount-manager"
	"github.com/guilhem/csi-runtime/utils"
)

type Server struct {
	nodeID  string
	manager Interface
	// store   storage.Interface
	mounter *mount.SafeFormatAndMount

	DeviceUtils   mountmanager.DeviceUtils
	VolumeStatter mountmanager.Statter

	maxVolumesPerNode int64
	segments          map[string]string

	time time.Duration

	// log logr.Logger

	// A map storing all volumes with ongoing operations so that additional operations
	// for that same volume (as defined by VolumeID) return an Aborted error
	volumeLocks *utils.VolumeLocks
}

var (
	ErrVolumeNotReady   = errors.New("volume is not yet ready to be setup")
	ErrNoImplementation = errors.New("no implementation")
)

func New(nodeID string, maxVolumesPerNode int64, manager Interface) (*Server, error) {
	mounter, err := mountmanager.NewSafeMounter()
	if err != nil {
		return nil, fmt.Errorf("can't create safe mounter: %w", err)
	}

	// sanity check
	switch manager.(type) {
	case Mount:
	case Populate:
	case Block:
	default:
		return nil, ErrNoImplementation
	}

	ns := &Server{
		nodeID:            nodeID,
		maxVolumesPerNode: maxVolumesPerNode,
		manager:           manager,
		// store:             store,
		mounter: mounter,
		// log:               log,
		time:        time.Second * 60,
		segments:    nil,
		volumeLocks: utils.NewVolumeLocks(),
	}

	return ns, nil
}

// func (ns *Server) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
// 	meta := metadata.FromNodePublishVolumeRequest(req)
// 	log := loggerForMetadata(ns.log, meta)

// 	if err := ns.manager.ValidatePublish(ctx); err != nil {
// 		st, ok := status.FromError(err)
// 		if ok {
// 			return nil, st.Err()
// 		}
// 		return nil, fmt.Errorf("can't validate: %w", err)
// 	}

// 	if registered, err := ns.store.RegisterMetadata(meta); err != nil {
// 		return nil, fmt.Errorf("can't register: %w", err)
// 	} else {
// 		if registered {
// 			log.Info("Registered new volume with storage backend")
// 		} else {
// 			log.Info("Volume already registered with storage backend")
// 		}
// 	}

// 	if ns.time != 0 {
// 		var cancel context.CancelFunc
// 		ctx, cancel = context.WithTimeout(ctx, time.Second*60)
// 		defer cancel()
// 	}

// 	rep, err := ns.publishVolume(ctx, req, log)
// 	if err != nil {
// 		// clean up after ourselves if provisioning fails.
// 		// this is required because if publishing never succeeds, unpublish is not
// 		// called which leaves files around (and we may continue to renew if so).

// 		log.Info("cleanup")
// 		ns.manager.UnmanageVolume(req.GetVolumeId())
// 		if err := ns.mounter.Unmount(req.GetTargetPath()); err != nil {
// 			log.Error(err, "fail to unmount")
// 		}
// 		if err := ns.store.RemoveVolume(req.GetVolumeId()); err != nil {
// 			log.Error(err, "fail to remove volume")
// 		}

// 		st, ok := status.FromError(err)
// 		if !ok {
// 			st = status.FromContextError(err)
// 		}

// 		return nil, st.Err()
// 	}

// 	return rep, nil
// }

func (ns *Server) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {

	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	stagingTargetPath := req.GetStagingTargetPath()
	// volumeContext := req.GetVolumeContext()
	volumeCapability := req.GetVolumeCapability()

	readOnly := req.GetReadonly()

	// publishContext := req.GetPublishContext()
	// secrets := req.GetSecrets()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Volume ID must be provided")
	}

	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Staging Target Path must be provided")
	}

	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Target Path must be provided")
	}

	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Volume Capability must be provided")
	}

	if err := validateVolumeCapability(volumeCapability); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability is invalid: %v", err)
	}

	if mv, ok := ns.manager.(Validate); ok {
		if err := mv.ValidateCapability(volumeCapability); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability is invalid: %v", err)
		}
	}

	targetFile, err := os.Stat(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Can't stat target path '%s': %v", targetPath, err)
	}

	if !targetFile.IsDir() {
		return nil, status.Errorf(codes.FailedPrecondition, "staging target path '%s' isn't a directory", stagingTargetPath)
	}

	// Lock volumeID

	if acquired := ns.volumeLocks.TryAcquire(volumeID); !acquired {
		return nil, status.Errorf(codes.Aborted, utils.VolumeOperationAlreadyExistsFmt, volumeID)
	}
	defer ns.volumeLocks.Release(volumeID)

	// req.VolumeCapability.GetBlock()

	// TODO validate volume

	switch req.VolumeCapability.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:

		blockPath, err := utils.GetDeviceFromPath(stagingTargetPath)
		if err != nil {
			return nil, err
		}

		// create symlink
		if err := os.Symlink(blockPath, targetPath); err != nil {
			return nil, status.Errorf(codes.Internal, "can't create symlink: %v", err)
		}

	case *csi.VolumeCapability_Mount:

		if ns.isVolumePathMounted(targetPath) {
			// TODO test values

			// klog.V(4).Infof("NodePublishVolume succeeded on volume %v to %s, mount already exists.", volumeID, targetPath)
			return &csi.NodePublishVolumeResponse{}, nil
		}

		file, err := os.Stat(stagingTargetPath)
		if err != nil || !file.IsDir() {
			return nil, status.Error(codes.FailedPrecondition, "can't stat staging file")
		}
		// // if mm, ok := ns.manager.(Mount); ok {
		// 	retErr = mm.Populate(ctx, req.StagingTargetPath, req.VolumeContext, req.PublishContext, req.Secrets)
		// } else {
		// 	return nil, status.Error(codes.Unimplemented, "Mount not implemented")
		// }

		// Perform a bind mount to the full path to allow duplicate mounts of the same PD.

		if err := utils.BindMount(stagingTargetPath, targetPath, readOnly, ns.mounter); err != nil {
			// // klog.Errorf("Mount of disk %s failed: %v", targetPath, err)
			// notMnt, mntErr := ns.mounter.IsLikelyNotMountPoint(targetPath)
			// if mntErr != nil {
			// 	// klog.Errorf("IsLikelyNotMountPoint check failed: %v", mntErr)
			// 	return nil, status.Error(codes.Internal, fmt.Sprintf("NodePublishVolume failed to check whether target path is a mount point: %v", err))
			// }
			// if !notMnt {
			// 	// TODO: check the logic here again. If mntErr == nil & notMnt == false, it means volume is actually mounted.
			// 	// Why need to unmount?
			// 	// klog.Warningf("Although volume mount failed, but IsLikelyNotMountPoint returns volume %s is mounted already at %s", volumeID, targetPath)
			// 	if mntErr = ns.mounter.Unmount(targetPath); mntErr != nil {
			// 		// klog.Errorf("Failed to unmount: %v", mntErr)
			// 		return nil, status.Error(codes.Internal, fmt.Sprintf("NodePublishVolume failed to unmount target path: %v", err))
			// 	}
			// 	notMnt, mntErr := ns.mounter.IsLikelyNotMountPoint(targetPath)
			// 	if mntErr != nil {
			// 		// klog.Errorf("IsLikelyNotMountPoint check failed: %v", mntErr)
			// 		return nil, status.Error(codes.Internal, fmt.Sprintf("NodePublishVolume failed to check whether target path is a mount point: %v", err))
			// 	}
			// 	if !notMnt {
			// 		// This is very odd, we don't expect it.  We'll try again next sync loop.
			// 		// klog.Errorf("%s is still mounted, despite call to unmount().  Will try again next sync loop.", targetPath)
			// 		return nil, status.Error(codes.Internal, fmt.Sprintf("NodePublishVolume something is wrong with mounting: %v", err))
			// 	}
			// }
			// if err := os.Remove(targetPath); err != nil {
			// 	// klog.Errorf("failed to remove targetPath %s: %v", targetPath, err)
			// }
			return nil, status.Error(codes.Internal, fmt.Sprintf("NodePublishVolume mount of disk failed: %v", err))
		}
	}

	// target := req.TargetPath

	// targetMounted, err := ns.mounter.IsMountPoint(target)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "fail to test target mount point: %s", target)
	// }

	// if targetMounted {
	// 	return nil, status.Errorf(codes.AlreadyExists, "target path is already a mount point: %s", target)
	// }

	// Bind options

	// ns.mounter.MountSensitive(source string, target string, fstype string, options []string, sensitiveOptions []string)
	return &csi.NodePublishVolumeResponse{}, nil
}

type VolumeResponse struct {
	Device string
	Mounts []struct {
		RelativePath     string
		Fstype           string
		Options          []string
		SensitiveOptions []string
	}
}

type MountResponse struct {
	Source           string
	Fstype           string
	Options          []string
	SensitiveOptions []string
	Format           bool
}

// This operation MUST be idempotent
// https://github.com/container-storage-interface/spec/blob/master/spec.md#nodestagevolume
func (ns *Server) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {

	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()
	volumeContext := req.GetVolumeContext()
	volumeCapability := req.GetVolumeCapability()

	publishContext := req.GetPublishContext()
	secrets := req.GetSecrets()

	// var volResp *VolumeResponse

	// Check inputs

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Target Path must be provided")
	}
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

	if err := validateVolumeCapability(volumeCapability); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability is invalid: %v", err)
	}

	if mv, ok := ns.manager.(Validate); ok {
		if err := mv.ValidateCapability(volumeCapability); err != nil {
			if err != nil {
				st, ok := status.FromError(err)
				if ok {
					return nil, st.Err()
				}

				return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability is invalid: %v", err)
			}
		}
	}

	// Check Node

	targetFile, err := os.Stat(stagingTargetPath)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Can't stat staging target path '%s': %v", stagingTargetPath, err)
	}

	if !targetFile.IsDir() {
		return nil, status.Errorf(codes.FailedPrecondition, "staging target path '%s' isn't a directory", stagingTargetPath)
	}

	// Lock volumeID

	if acquired := ns.volumeLocks.TryAcquire(volumeID); !acquired {
		return nil, status.Errorf(codes.Aborted, utils.VolumeOperationAlreadyExistsFmt, volumeID)
	}
	defer ns.volumeLocks.Release(volumeID)

	// Manage Block or Mount Volume

	var blockPath string
	// var mountResponse *MountResponse

	if mb, ok := ns.manager.(Block); ok {
		blockPath, err = mb.AttachBlock(ctx, volumeContext, publishContext, secrets)
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				return nil, st.Err()
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	switch req.VolumeCapability.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		if blockPath == "" {
			return nil, status.Error(codes.Unimplemented, "Block not implemented")
		}

		// create symlink
		if err := os.Symlink(blockPath, stagingTargetPath); err != nil {
			return nil, status.Errorf(codes.Internal, "can't create symlink: %v", err)
		}
	case *csi.VolumeCapability_Mount:

		var provisionned bool

		if mm, ok := ns.manager.(Mount); ok {
			fsType := defaultFsType

			if mnt := volumeCapability.GetMount(); mnt != nil {
				if fsT := mnt.GetFsType(); fsT != "" {
					fsType = fsT
				}
			}

			volumeCapability.GetAccessMode()

			mountResponse, err := mm.Mount(ctx, stagingTargetPath, fsType, volumeContext, publishContext, secrets)
			if err != nil {
				st, ok := status.FromError(err)
				if ok {
					return nil, st.Err()
				}
				return nil, status.Error(codes.Internal, err.Error())
			}

			if m := mountResponse; m != nil {
				if m.Fstype != "" {
					fsType = m.Fstype
				}

				options := []string{}

				if mnt := volumeCapability.GetMount(); mnt != nil {
					if mnt.FsType != "" {
						fsType = mnt.FsType
					}

					options = slice.Union(options, mnt.GetMountFlags())
				}

				if fsType == fsTypeXFS {
					// By default, xfs does not allow mounting of two volumes with the same filesystem uuid.
					// Force ignore this uuid to be able to mount volume + its clone / restored snapshot on the same node.
					options = slice.AppendIfAbsent(options, "nouuid")
				}

				if Readonly(volumeCapability) {
					options = slice.AppendIfAbsent(options, "ro")
					// klog.V(4).Infof("CSI volume is read-only, mounting with extra option ro")
				}

				// Copy options
				// localOptions := append([]string(nil), options...)

				// Merge with options from manager
				localOptions := slice.Union(options, m.Options)

				localSensitiveOptions := slice.Difference(m.SensitiveOptions, options)

				// // prevent bad relative paths
				// cleanedRelative := path.Clean(m.RelativePath)

				// // create absolute path
				// mTarget := path.Join(target, cleanedRelative)

				// ns.DeviceUtils.VerifyDevicePath(m.Source, deviceName string)

				// mount
				if m.Format {
					if err := ns.mounter.FormatAndMountSensitive(m.Source, stagingTargetPath, m.Fstype, localOptions, localSensitiveOptions); err != nil {
						return nil, status.Errorf(codes.Internal, "fail to format and mount: %v", err)
					}
				} else {
					if err := ns.mounter.MountSensitive(m.Source, stagingTargetPath, fsType, localOptions, localSensitiveOptions); err != nil {
						return nil, status.Errorf(codes.Internal, "fail to mount '%s': %v", stagingTargetPath, err)
					}
				}
				provisionned = true
			}
		}

		if mm, ok := ns.manager.(Populate); ok {
			if err := mm.Populate(ctx, stagingTargetPath, volumeContext, publishContext, secrets); err != nil {
				st, ok := status.FromError(err)
				if ok {
					return nil, st.Err()
				}
				return nil, status.Error(codes.Internal, err.Error())
			}

			provisionned = true
		}

		if !provisionned {
			return nil, status.Error(codes.Unimplemented, "Mount not implemented")
		}
	}

	// if retErr != nil {
	// 	st, ok := status.FromError(retErr)
	// 	if !ok {
	// 		st = status.FromContextError(retErr)
	// 	}
	// 	return nil, st.Err()
	// }

	// mountFlags := MergeMountOptions([]string{}, req.VolumeCapability)

	// switch request.VolumeCapability.GetAccessMode().Mode {
	// case:

	// case:
	// }
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *Server) isVolumePathMounted(path string) bool {
	notMnt, err := ns.mounter.IsLikelyNotMountPoint(path)
	// klog.V(4).Infof("NodePublishVolume check volume path %s is mounted %t: error %v", path, !notMnt, err)
	if err == nil && !notMnt {
		// TODO(#95): check if mount is compatible. Return OK if it is, or appropriate error.
		/*
			1) Target Path MUST be the vol referenced by vol ID
			2) TODO(#253): Check volume capability matches for ALREADY_EXISTS
			3) Readonly MUST match
		*/
		return true
	}
	return false
}

func validateVolumeCapability(vc *csi.VolumeCapability) error {
	if err := validateAccessMode(vc.GetAccessMode()); err != nil {
		return err
	}
	blk := vc.GetBlock()
	mnt := vc.GetMount()
	if mnt == nil && blk == nil {
		return errors.New("must specify an access type")
	}
	if mnt != nil && blk != nil {
		return errors.New("specified both mount and block access types")
	}
	// if mnt != nil && mod == csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER {
	// 	return errors.New("specified multi writer with mount access type")
	// }
	return nil
}

func validateAccessMode(am *csi.VolumeCapability_AccessMode) error {
	if am == nil {
		return errors.New("access mode is nil")
	}

	// switch am.GetMode() {
	// case csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER:
	// case csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY:
	// case csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY:
	// case csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER:
	// default:
	// 	return fmt.Errorf("%v access mode is not supported for for PD", am.GetMode())
	// }
	return nil
}

// https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeunstagevolume
func (ns *Server) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {

	// Validate arguments
	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Volume ID must be provided")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Staging Target Path must be provided")
	}

	if acquired := ns.volumeLocks.TryAcquire(volumeID); !acquired {
		return nil, status.Errorf(codes.Aborted, utils.VolumeOperationAlreadyExistsFmt, volumeID)
	}
	defer ns.volumeLocks.Release(volumeID)

	// if block

	blockPath, err := utils.GetDeviceFromPath(stagingTargetPath)
	if err != nil {
		return nil, err
	}

	if blockPath != "" {
		// if block
		if err := os.Remove(stagingTargetPath); err != nil {
			return nil, err
		}
		return &csi.NodeUnstageVolumeResponse{}, nil
	} else {
		blockPath, err = utils.GetDeviceFromMount(ns.mounter, stagingTargetPath)
		if err != nil {
			return nil, err
		}

		if blockPath != "" {
			// if mount
			if err := ns.mounter.Unmount(stagingTargetPath); err != nil {
				return nil, err
			}
		} else {
			// if populate
			if err := os.RemoveAll(stagingTargetPath); err != nil {
				return nil, err
			}
		}
	}

	if mb, ok := ns.manager.(Block); ok {
		if err := mb.DetachBlock(ctx, blockPath); err != nil {
			return nil, err
		}
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *Server) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {

	// Validate Arguments
	targetPath := req.GetTargetPath()
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Volume ID must be provided")
	}
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Target Path must be provided")
	}

	if acquired := ns.volumeLocks.TryAcquire(volumeID); !acquired {
		return nil, status.Errorf(codes.Aborted, utils.VolumeOperationAlreadyExistsFmt, volumeID)
	}
	defer ns.volumeLocks.Release(volumeID)

	if err := ns.mounter.Unmount(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "Unmount failed: %v\nUnmounting arguments: %s\n", err, targetPath)
	}

	// cleanup target path
	if err := os.Remove(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "cleanup failed: %w", err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil

}

// 	log := ns.log.WithValues("volume_id", request.VolumeId, "target_path", request.TargetPath)
// 	ns.manager.UnmanageVolume(request.GetVolumeId())
// 	log.Info("Stopped management of volume")

// 	mnt, err := ns.mounter.IsMountPoint(request.GetTargetPath())
// 	if err != nil {
// 		return nil, err
// 	}
// 	if mnt {
// 		if err := ns.mounter.Unmount(request.GetTargetPath()); err != nil {
// 			return nil, err
// 		}

// 		log.Info("Unmounted targetPath")
// 	}

// 	if err := ns.store.RemoveVolume(request.GetVolumeId()); err != nil {
// 		return nil, err
// 	}

// 	log.Info("Removed data directory")

// 	return &csi.NodeUnpublishVolumeResponse{}, nil
// }

func (ns *Server) NodeGetVolumeStats(ctx context.Context, request *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeGetVolumeStats not implemented")
}

func (ns *Server) NodeExpandVolume(ctx context.Context, request *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeExpandVolume not implemented")
}

func (ns *Server) NodeGetCapabilities(ctx context.Context, request *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {

	cap := make([]*csi.NodeServiceCapability, 0)

	// csi-runtime always support staging
	cap = append(cap, &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME}}})

	if _, ok := ns.manager.(Expand); ok {
		cap = append(cap, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_EXPAND_VOLUME}}})
	}

	// return &csi.NodeGetCapabilitiesResponse{
	// 	Capabilities: []*csi.NodeServiceCapability{
	// 		{
	// 			Type: &csi.NodeServiceCapability_Rpc{
	// 				Rpc: &csi.NodeServiceCapability_RPC{
	// 					Type: csi.NodeServiceCapability_RPC_UNKNOWN,
	// 				},
	// 			},
	// 		},
	// 	},
	// }, nil

	return &csi.NodeGetCapabilitiesResponse{Capabilities: cap}, nil
}

func (ns *Server) NodeGetInfo(ctx context.Context, request *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	resp := &csi.NodeGetInfoResponse{
		NodeId:            ns.nodeID,
		MaxVolumesPerNode: ns.maxVolumesPerNode,
	}
	if ns.segments != nil {
		resp.AccessibleTopology = &csi.Topology{
			Segments: ns.segments,
		}
	}
	return resp, nil
}

func loggerForMetadata(log logr.Logger, meta metadata.Metadata) logr.Logger {
	return log.WithValues("pod_name", meta.VolumeContext["csi.storage.k8s.io/pod.name"])
}
