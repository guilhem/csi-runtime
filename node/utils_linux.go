//go:build !windows

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
	"os"
	"strconv"
	"strings"

	"k8s.io/mount-utils"
)

const (
	defaultFsType = "ext4"
)

func formatAndMount(source, target, fstype string, options []string, m *mount.SafeFormatAndMount) error {
	return m.FormatAndMount(source, target, fstype, options)
}

func preparePublishPath(path string, m *mount.SafeFormatAndMount) error {
	return os.MkdirAll(path, 0750)
}

// func prepareStagePathForMount(path string, m *mount.SafeFormatAndMount) error {
// 	return os.MkdirAll(path, 0750)
// }

func cleanupPublishPath(path string, m *mount.SafeFormatAndMount) error {
	return mount.CleanupMountPoint(path, m, false /* bind mount */)
}

func cleanupStagePath(path string, m *mount.SafeFormatAndMount) error {
	return cleanupPublishPath(path, m)
}

func getBlockSizeBytes(devicePath string, m *mount.SafeFormatAndMount) (int64, error) {
	output, err := m.Exec.Command("blockdev", "--getsize64", devicePath).CombinedOutput()
	if err != nil {
		return -1, fmt.Errorf("error when getting size of block volume at path %s: output: %s, err: %v", devicePath, string(output), err)
	}
	strOut := strings.TrimSpace(string(output))
	gotSizeBytes, err := strconv.ParseInt(strOut, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("failed to parse %s into an int size", strOut)
	}
	return gotSizeBytes, nil
}
