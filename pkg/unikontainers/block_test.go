// Copyright (c) 2023-2026, Nubificus LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikontainers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urunc-dev/urunc/pkg/unikontainers/types"
)

func TestGetBlockDevice(t *testing.T) {
	// Create a mock partition
	tmpMnt := types.BlockDevParams{
		Source:     "proc",
		MountPoint: "/proc",
		FsType:     "proc",
		ID:         "",
	}

	rootFs, err := getMountInfo("/proc")
	assert.NoError(t, err, "Expected no error in getting block device")
	assert.Equal(t, tmpMnt.Source, rootFs.Source, "Incorrect image")
	assert.Equal(t, tmpMnt.MountPoint, rootFs.MountPoint, "Incorrect mountpoint")
	assert.Equal(t, tmpMnt.FsType, rootFs.FsType, "Expected filesystem type to be proc")
	assert.Equal(t, tmpMnt.ID, rootFs.ID, "Expected ID to be empty")
}

func TestHandleExplicitBlockImage(t *testing.T) {
	tests := []struct {
		name       string
		blockImg   string
		mountPoint string
		blkDev     string
		wantID     string
		wantErr    bool
	}{
		{
			name:       "no block image returns empty params",
			blockImg:   "",
			mountPoint: "/",
			blkDev:     "",
			wantID:     "",
		},
		{
			name:       "missing mountpoint is an error",
			blockImg:   "/.boot/rootfs",
			mountPoint: "",
			blkDev:     "",
			wantErr:    true,
		},
		{
			name:       "root mount with no device name defaults to rootfs",
			blockImg:   "/.boot/rootfs",
			mountPoint: "/",
			blkDev:     "",
			wantID:     "rootfs",
		},
		{
			name:       "device name annotation overrides default for root mount",
			blockImg:   "/.boot/rootfs",
			mountPoint: "/",
			blkDev:     "storage",
			wantID:     "storage",
		},
		{
			name:       "non-root mount with device name uses that name",
			blockImg:   "/.boot/data",
			mountPoint: "/data",
			blkDev:     "data",
			wantID:     "data",
		},
		{
			name:       "non-root mount without device name has empty ID",
			blockImg:   "/.boot/data",
			mountPoint: "/data",
			blkDev:     "",
			wantID:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := handleExplicitBlockImage(tt.blockImg, tt.mountPoint, tt.blkDev)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantID, got.ID)
		})
	}
}
