//go:build windows

/*
Copyright 2020 The Kubernetes Authors.
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
	"fmt"
	"strings"

	mounter "github.com/guilhem/csi-runtime/mount-manager"
	"k8s.io/mount-utils"
)

const (
	defaultFsType = "ntfs"
)

func formatAndMount(source, target, fstype string, options []string, m *mount.SafeFormatAndMount) error {
	if !strings.EqualFold(fstype, defaultFsType) {
		return fmt.Errorf("GCE PD CSI driver can only supports %s file system, it does not support %s", defaultWindowsFsType, fstype)
	}
	proxy, ok := m.Interface.(mounter.CSIProxyMounter)
	if !ok {
		return fmt.Errorf("could not cast to csi proxy class")
	}
	return proxy.FormatAndMount(source, target, fstype, options)
}

// Before mounting (which means creating symlink) in Windows, the targetPath should
// not exist. Currently kubelet creates the path beforehand, this is a workaround to
// remove the path first.
func preparePublishPath(path string, m *mount.SafeFormatAndMount) error {
	proxy, ok := m.Interface.(mounter.CSIProxyMounter)
	if !ok {
		return fmt.Errorf("could not cast to csi proxy class")
	}
	exists, err := proxy.ExistsPath(path)
	if err != nil {
		return err
	}
	if exists {
		return proxy.RemovePodDir(path)
	}
	return nil
}

// // Before staging (which means creating symlink) in Windows, the targetPath should
// // not exist.
// func prepareStagePathForMount(path string, m *mount.SafeFormatAndMount) error {
// 	return nil
// }

func cleanupPublishPath(path string, m *mount.SafeFormatAndMount) error {
	proxy, ok := m.Interface.(mounter.CSIProxyMounter)
	if !ok {
		return fmt.Errorf("could not cast to csi proxy class")
	}
	return proxy.RemovePodDir(path)
}

func cleanupStagePath(path string, m *mount.SafeFormatAndMount) error {
	proxy, ok := m.Interface.(mounter.CSIProxyMounter)
	if !ok {
		return fmt.Errorf("could not cast to csi proxy class")
	}
	return proxy.UnmountDevice(path)
}

func disableDevice(devicePath string) error {
	// This is a no-op on windows.
	return nil
}

func getBlockSizeBytes(devicePath string, m *mount.SafeFormatAndMount) (int64, error) {
	proxy, ok := m.Interface.(mounter.CSIProxyMounter)
	if !ok {
		return 0, fmt.Errorf("could not cast to csi proxy class")
	}
	return proxy.GetDiskTotalBytes(devicePath)
}
