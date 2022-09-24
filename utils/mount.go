/*
Copyright 2019 The Kubernetes Authors.
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

package utils

import (
	"errors"
	"os"
	"path/filepath"

	"k8s.io/mount-utils"
)

var ErrNoDevice = errors.New("can't find device")

// GetDeviceNameFromMount given a mnt point, find the device from /proc/mounts
// returns the device name, reference count, and error code.
func GetDeviceFromPath(mountPath string) (string, error) {

	slTarget, err := filepath.EvalSymlinks(mountPath)
	if err != nil {
		return "", err
	}

	// notMnt, err := mounter.IsLikelyNotMountPoint(slTarget)
	// if err != nil {
	// 	return "", err
	// }

	// if notMnt {
	file, err := os.Stat(slTarget)
	if err != nil {
		return "", err
	}
	if file.Mode() == os.ModeDevice {
		return slTarget, nil
	}
	return "", nil
}

func GetDeviceFromMount(mounter mount.Interface, mountPath string) (string, error) {

	slTarget, err := filepath.EvalSymlinks(mountPath)
	if err != nil {
		return "", err
	}

	notMnt, err := mounter.IsLikelyNotMountPoint(slTarget)
	if err != nil {
		return "", err
	}

	if notMnt {
		return "", nil
	}

	mps, err := mounter.List()
	if err != nil {
		return "", err
	}

	// FIXME if multiple devices mounted on the same mount path, only the first one is returned.
	// If mountPath is symlink, need get its target path.
	for i := range mps {
		if mps[i].Path == slTarget {
			return mps[i].Device, nil
		}
	}

	return "", nil
}

func BindMount(source, destination string, readOnly bool, mounter mount.Interface) error {
	fstype := ""
	options := []string{"bind"}
	if readOnly {
		options = append(options, "ro")
	}

	return mounter.Mount(source, destination, fstype, options)
}
