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

package hypervisors

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urunc-dev/urunc/pkg/unikontainers/types"
	"github.com/urunc-dev/urunc/pkg/unikontainers/unikernels"
)

// TestHVTBuildExecCmdMirageBlockName verifies that the block device name a
// MirageOS image specifies (through the com.urunc.unikernel.blkDev annotation,
// which urunc threads into BlockDevParams.ID) reaches the solo5-hvt command
// line as "--block:<name>=<path>". It exercises the full chain
// Mirage.Init -> Mirage.MonitorBlockCli -> HVT.BuildExecCmd.
func TestHVTBuildExecCmdMirageBlockName(t *testing.T) {
	tests := []struct {
		name      string
		blockID   string
		wantBlock string
	}{
		{
			name:      "custom device name reaches solo5 cli",
			blockID:   "storage",
			wantBlock: "--block:storage=/dev/vda",
		},
		{
			name:      "alternate device name reaches solo5 cli",
			blockID:   "mydata",
			wantBlock: "--block:mydata=/dev/vda",
		},
		{
			name:      "empty device name falls back to storage default",
			blockID:   "",
			wantBlock: "--block:storage=/dev/vda",
		},
		{
			name:      "generic rootfs id falls back to storage default",
			blockID:   "rootfs",
			wantBlock: "--block:storage=/dev/vda",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mirage, err := unikernels.New(unikernels.MirageUnikernel)
			assert.NoError(t, err)

			err = mirage.Init(types.UnikernelParams{
				Monitor: string(HvtVmm),
				Block: []types.BlockDevParams{
					{ID: tt.blockID, Source: "/dev/vda"},
				},
			})
			assert.NoError(t, err)

			hvt := &HVT{binaryPath: "/opt/urunc/bin/solo5-hvt"}
			cmd, err := hvt.BuildExecCmd(types.ExecArgs{
				MemSizeB:      256 * 1024 * 1024,
				UnikernelPath: "/.boot/kernel",
			}, mirage)
			assert.NoError(t, err)

			joined := strings.Join(cmd, " ")
			assert.Contains(t, joined, tt.wantBlock,
				"solo5-hvt command line must attach the block with the expected name")
		})
	}
}

// TestHVTBuildExecCmdMirageMultipleBlocks verifies that multiple MirageOS block
// devices are all attached on the solo5-hvt command line, each with its own
// device name.
func TestHVTBuildExecCmdMirageMultipleBlocks(t *testing.T) {
	mirage, err := unikernels.New(unikernels.MirageUnikernel)
	assert.NoError(t, err)

	err = mirage.Init(types.UnikernelParams{
		Monitor: string(HvtVmm),
		Block: []types.BlockDevParams{
			{ID: "storage", Source: "/dev/vda"},
			{ID: "data", Source: "/dev/vdb"},
		},
	})
	assert.NoError(t, err)

	hvt := &HVT{binaryPath: "/opt/urunc/bin/solo5-hvt"}
	cmd, err := hvt.BuildExecCmd(types.ExecArgs{
		MemSizeB:      256 * 1024 * 1024,
		UnikernelPath: "/.boot/kernel",
	}, mirage)
	assert.NoError(t, err)

	joined := strings.Join(cmd, " ")
	assert.Contains(t, joined, "--block:storage=/dev/vda")
	assert.Contains(t, joined, "--block:data=/dev/vdb")
}
